import argparse
import json
from pathlib import Path
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid


def docker(*arguments, check=True):
    return subprocess.run(['docker', *arguments], check=check, capture_output=True, text=True, timeout=40)


def request(url):
    try:
        with urllib.request.urlopen(url, timeout=2) as response:
            return response.status, response.read(1024 * 1024)
    except urllib.error.HTTPError as error:
        return error.code, error.read(4096)


def main():
    parser = argparse.ArgumentParser(description='Verify one built service image locally without a browser or cloud target.')
    parser.add_argument('--image', required=True)
    parser.add_argument('--output', type=Path, required=True)
    arguments = parser.parse_args()
    if not arguments.image.startswith('sha256:') or len(arguments.image) != 71:
        raise ValueError('Supply the immutable built service image ID')
    root = arguments.output.resolve()
    root.mkdir(parents=True, exist_ok=True, mode=0o700)
    suffix = uuid.uuid4().hex[:12]
    volume = 'teststudio-service-data-' + suffix
    containers = []
    summary = {'image': arguments.image, 'cloudChecks': False, 'browserChecks': False, 'checks': [], 'passed': False}
    cleanup = {}
    metadata = json.loads(docker('image', 'inspect', arguments.image).stdout)[0]
    assert metadata['Config']['User'] == '65532:65532'
    assert metadata['Config']['Labels']['teststudio.package'] == 'service/v1'
    sockets = []
    for _ in range(2):
        listener = socket.socket()
        listener.bind(('127.0.0.1', 0))
        sockets.append(listener)
    api_port, artifact_port = [listener.getsockname()[1] for listener in sockets]
    for listener in sockets:
        listener.close()
    origin = 'http://127.0.0.1:' + str(api_port)
    artifact_origin = 'http://127.0.0.1:' + str(artifact_port)
    try:
        docker('volume', 'create', '--label', 'teststudio.verification=service-package', volume)
        migrations = None
        for iteration in range(2):
            name = 'teststudio-service-check-' + suffix + '-' + str(iteration)
            containers.append(name)
            docker('run', '--detach', '--name', name, '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--tmpfs', '/tmp:rw,nosuid,nodev,size=67108864,uid=65532,gid=65532,mode=1777', '--mount', 'type=volume,source=' + volume + ',target=/data', '--env', 'TESTSTUDIO_DATABASE=/data/service.db', '--publish', '127.0.0.1:' + str(api_port) + ':8040', '--publish', '127.0.0.1:' + str(artifact_port) + ':8041', arguments.image, '--public-url=' + origin, '--artifact-origin=' + artifact_origin)
            ready = False
            for _ in range(100):
                try:
                    status, body = request(origin + '/healthz')
                    ready = status == 200 and json.loads(body) == {'status': 'up', 'service': 'teststudio'}
                except (OSError, ValueError):
                    pass
                if ready:
                    break
                state = json.loads(docker('inspect', name).stdout)[0]['State']
                if not state['Running']:
                    raise RuntimeError('The owned service container stopped before readiness')
                time.sleep(0.1)
            assert ready, 'The service package did not become ready'
            docker('exec', name, '/app/teststudio-health')
            status, body = request(origin + '/livez')
            assert status == 200 and json.loads(body) == {'status': 'up', 'service': 'teststudio'}
            status, html = request(origin + '/')
            assert status == 200 and b'<ts-root' in html
            status, body = request(artifact_origin + '/')
            assert status == 404 and b'<ts-root' not in body
            container = json.loads(docker('inspect', name).stdout)[0]
            assert container['HostConfig']['ReadonlyRootfs']
            assert container['Config']['User'] == '65532:65532'
            assert container['HostConfig']['CapDrop'] == ['ALL']
            docker('stop', '--time=10', name)
            database = root / ('service-' + str(iteration) + '.db')
            docker('cp', name + ':/data/service.db', str(database))
            connection = sqlite3.connect('file:' + str(database) + '?mode=ro&immutable=1', uri=True)
            current = connection.execute('SELECT version, applied_at FROM schema_migrations ORDER BY version').fetchall()
            assert current
            assert connection.execute("SELECT count(*) FROM records WHERE bucket IN ('projects','runs','schedules')").fetchone()[0] == 0
            connection.close()
            if migrations is not None:
                assert current == migrations, 'Container recreation replaced the persisted database'
            migrations = current
            docker('rm', name)
        summary['checks'] += ['Non-root startup with a read-only root filesystem and no Linux capabilities', 'Writable private SQLite and artifact volume', 'Bounded in-image readiness probe', 'Independent process liveness endpoint', 'Built UI served on the API origin', 'Separate artifact origin serves no UI', 'Container recreation preserves the same SQLite migration state', 'Fresh package starts with empty project, run, and schedule registries']
        invalid = 'teststudio-service-invalid-' + suffix
        containers.append(invalid)
        canary = 'private-service-database-canary-' + suffix
        with tempfile.TemporaryDirectory(prefix='teststudio-service-secret-') as directory:
            environment = Path(directory) / 'database.env'
            environment.write_text('TESTSTUDIO_DATABASE=postgres://fixture:' + canary + '@127.0.0.1:invalid/studio\n')
            environment.chmod(0o600)
            result = docker('run', '--name', invalid, '--network=none', '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--env-file', str(environment), arguments.image, check=False)
            assert result.returncode != 0
            assert canary not in result.stdout + result.stderr
            assert 'database initialization failed' in result.stderr
            configured = json.loads(docker('inspect', invalid).stdout)[0]
            assert canary not in json.dumps(configured['Args'])
        summary['checks'].append('Database environment values stay out of process arguments and startup errors')
        guarded = 'teststudio-service-postgres-' + suffix
        containers.append(guarded)
        result = docker('run', '--name', guarded, '--network=none', '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', arguments.image, '--require-postgres=true', check=False)
        assert result.returncode != 0 and 'requires a PostgreSQL database connection' in result.stderr
        summary['checks'].append('Hosted mode rejects the packaged SQLite fallback before opening storage')
        summary['passed'] = True
        print(json.dumps({'passed': True, 'checks': len(summary['checks']), 'output': str(root)}))
    finally:
        for name in containers:
            docker('rm', '--force', name, check=False)
            absent = docker('inspect', name, check=False)
            cleanup[name] = absent.returncode != 0 and 'no such' in absent.stderr.lower()
        docker('volume', 'rm', volume, check=False)
        absent = docker('volume', 'inspect', volume, check=False)
        cleanup[volume] = absent.returncode != 0 and 'no such volume' in absent.stderr.lower()
        (root / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        (root / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
        if not all(cleanup.values()):
            raise RuntimeError('Some owned service verification resources remain; inspect cleanup.json')


if __name__ == '__main__':
    main()
