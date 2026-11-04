import argparse
import json
from pathlib import Path
import runpy
import socket
import sqlite3
import time
import uuid


support = runpy.run_path(str(Path(__file__).with_name('verify-service-image.py')))
docker, request = support['docker'], support['request']


def main():
    parser = argparse.ArgumentParser(description='Check separate service roles sequentially in one fresh local installation without cloud access or browsers.')
    parser.add_argument('--image', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    metadata = json.loads(docker('image', 'inspect', args.image).stdout)[0]
    assert args.image == metadata['Id'] and metadata['Config']['User'] == '65532:65532'
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    suffix = uuid.uuid4().hex[:12]
    volume = 'teststudio-roles-data-' + suffix
    containers = []
    cleanup = {}
    summary = {'passed': False, 'image': args.image, 'architecture': metadata['Architecture'], 'checks': [], 'cloudOperations': False, 'browserRuns': False}
    sockets = []
    for _ in range(2):
        item = socket.socket()
        item.bind(('127.0.0.1', 0))
        sockets.append(item)
    api_port, artifact_port = [item.getsockname()[1] for item in sockets]
    for item in sockets:
        item.close()

    def cold_mode(container, role):
        destination = output / (role + '.db')
        docker('cp', container + ':/data/service.db', str(destination))
        connection = sqlite3.connect('file:' + str(destination) + '?mode=ro&immutable=1', uri=True)
        row = connection.execute("SELECT revision,payload FROM records WHERE bucket='local-auth-mode' AND id='installation'").fetchone()
        assert connection.execute("SELECT count(*) FROM records WHERE bucket IN ('projects','runs','schedules','providers','users')").fetchone()[0] == 0
        connection.close()
        assert row and json.loads(row[1])['enabled']
        return row

    try:
        docker('volume', 'create', '--label', 'teststudio.verification=service-roles', volume)
        before = None
        for role, active_port in [('api', api_port), ('artifacts', artifact_port)]:
            name = 'teststudio-role-' + role + '-' + suffix
            containers.append(name)
            extra = ['--artifact-addr=invalid-unused-address', '--local-accounts=true'] if role == 'api' else ['--addr=invalid-unused-address']
            docker('run', '--detach', '--name', name, '--platform=linux/' + metadata['Architecture'], '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--tmpfs', '/tmp:rw,nosuid,nodev,uid=65532,gid=65532,mode=1777', '--mount', 'type=volume,source=' + volume + ',target=/data', '--env', 'TESTSTUDIO_DATABASE=/data/service.db', '--publish', '127.0.0.1:' + str(api_port) + ':8040', '--publish', '127.0.0.1:' + str(artifact_port) + ':8041', args.image, '--serve-role=' + role, '--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test', *extra)
            origin = 'http://127.0.0.1:' + str(active_port)
            ready = False
            for _ in range(100):
                try:
                    status, data = request(origin + '/healthz')
                    ready = status == 200 and json.loads(data)['service'] == 'teststudio'
                except (OSError, ValueError):
                    pass
                if ready:
                    break
                state = json.loads(docker('inspect', name).stdout)[0]['State']
                assert state['Running'], 'The selected service role stopped before readiness'
                time.sleep(0.1)
            assert ready, 'The selected service role did not become ready'
            docker('exec', name, '/app/teststudio-health', '--url=http://127.0.0.1:' + str(8040 if role == 'api' else 8041) + '/livez')
            inactive = docker('exec', name, '/app/teststudio-health', '--url=http://127.0.0.1:' + str(8041 if role == 'api' else 8040) + '/livez', check=False)
            assert inactive.returncode != 0
            if role == 'api':
                status, body = request(origin + '/')
                assert status == 200 and b'<ts-root' in body
                summary['checks'].append('API-only role serves UI and health without binding the unused artifact address')
            else:
                for path in ['/', '/auth/providers', '/api/v1/projects']:
                    status, body = request(origin + path)
                    assert status == 404 and b'<ts-root' not in body
                summary['checks'].append('Artifact-only role serves health and private routes without API, sign-in, UI, or the unused API address')
            docker('stop', '--time=10', name)
            current = cold_mode(name, role)
            if before is None:
                before = current
            else:
                assert current == before, 'Artifact-only startup changed the saved local authentication mode'
                summary['checks'].append('Artifact startup preserves API-owned local authentication mode and leaves initial product registries empty')
            docker('rm', '--volumes', name)
        for value, extra in [('unsupported', []), ('artifacts', ['--local-accounts=true'])]:
            name = 'teststudio-role-invalid-' + uuid.uuid4().hex[:12]
            containers.append(name)
            result = docker('run', '--name', name, '--platform=linux/' + metadata['Architecture'], '--network=none', args.image, '--serve-role=' + value, *extra, check=False)
            assert result.returncode != 0
            assert ('serve-role must be' if value == 'unsupported' else 'cannot configure identity') in result.stderr
        summary['checks'].append('Unknown roles and artifact identity initialization flags fail before opening product storage')
        summary['passed'] = True
        print(json.dumps({'passed': True, 'checks': len(summary['checks']), 'architecture': metadata['Architecture'], 'output': str(output)}))
    finally:
        for name in containers:
            before = docker('inspect', name, check=False)
            anonymous = []
            if before.returncode == 0:
                anonymous = [mount['Name'] for mount in json.loads(before.stdout)[0].get('Mounts', []) if mount['Type'] == 'volume' and mount['Name'] != volume]
            docker('rm', '--force', '--volumes', name, check=False)
            state = docker('inspect', name, check=False)
            cleanup[name] = state.returncode != 0 and 'no such' in state.stderr.lower()
            for item in anonymous:
                state = docker('volume', 'inspect', item, check=False)
                cleanup[item] = state.returncode != 0 and 'no such volume' in state.stderr.lower()
        docker('volume', 'rm', volume, check=False)
        state = docker('volume', 'inspect', volume, check=False)
        cleanup[volume] = state.returncode != 0 and 'no such volume' in state.stderr.lower()
        (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        (output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
        assert all(cleanup.values()), 'An owned verification resource remains; inspect cleanup.json'


if __name__ == '__main__':
    main()
