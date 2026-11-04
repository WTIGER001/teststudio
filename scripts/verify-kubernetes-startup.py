import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import uuid


def main():
    parser = argparse.ArgumentParser(description='Check local startup guards without a cluster or cloud account.')
    parser.add_argument('--native-build', type=Path, required=True)
    parser.add_argument('--settings', type=Path, required=True)
    parser.add_argument('--service-image', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    binary = json.loads(args.native_build.read_text())['server']
    settings = json.loads(args.settings.read_text())
    checks = []
    cleanup = {}

    def verify(command, expected, label):
        result = subprocess.run(command, capture_output=True, text=True, timeout=15)
        (args.output / (label + '-stderr.log')).write_text(result.stderr)
        assert result.returncode != 0 and expected in result.stderr, label + ' did not reject invalid startup'
        assert 'flag provided but not defined' not in result.stderr
        checks.append({'name': label, 'rejectedBeforeDispatch': True})

    with tempfile.TemporaryDirectory(prefix='teststudio-worker-startup-') as directory:
        root = Path(directory)
        root.chmod(0o755)
        config = root / 'settings.json'
        config.write_text(json.dumps(settings))
        config.chmod(0o444)
        database = root / 'unopened.db'
        common = ['--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test', '--database=' + str(database)]
        worker = ['--kubernetes-workers-config=' + str(config), '--execution-location=kubernetes']
        verify([binary, *common, *worker, '--worker-addr=127.0.0.1:8042'], 'requires a PostgreSQL', 'native-postgres-required')
        verify([binary, *common, *worker, '--serve-role=artifacts'], 'cannot configure hosted execution', 'native-artifact-role-rejected')
        verify([binary, *common, '--kubernetes-workers-config=' + str(config), '--worker-addr=127.0.0.1:8042'], 'require their execution location', 'native-location-required')
        assert not database.exists(), 'rejected startup initialized a database'
        if args.service_image:
            image = json.loads(args.service_image.read_text())['image']
            for label, extra, expected in [
                ('image-postgres-required', ['--worker-addr=0.0.0.0:8042'], 'requires a PostgreSQL'),
                ('image-artifact-role-rejected', ['--serve-role=artifacts'], 'cannot configure hosted execution'),
            ]:
                name = 'teststudio-worker-guard-' + uuid.uuid4().hex[:12]
                try:
                    command = ['docker', 'run', '--name', name, '--rm', '--network=none', '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--tmpfs', '/tmp:rw,nosuid,nodev,size=16m', '--env', 'TESTSTUDIO_WORKER_CONFIGURATION=' + json.dumps(settings), image, '--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test', '--database-env=', '--database=/tmp/unopened.db', '--execution-location=kubernetes', '--kubernetes-workers-env=TESTSTUDIO_WORKER_CONFIGURATION', *extra]
                    verify(command, expected, label)
                finally:
                    subprocess.run(['docker', 'rm', '-fv', name], capture_output=True, text=True, timeout=10)
                    remaining = subprocess.run(['docker', 'container', 'inspect', name], capture_output=True, text=True, timeout=10)
                    cleanup[name] = remaining.returncode != 0 and 'No such' in remaining.stderr
                    (args.output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
                    assert cleanup[name], 'owned probe cleanup was not confirmed'
    (args.output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
    (args.output / 'summary.json').write_text(json.dumps({'passed': True, 'clusterContact': False, 'checks': checks}, indent=2) + '\n')
    print(json.dumps({'passed': True, 'checks': len(checks), 'clusterContact': False}))


if __name__ == '__main__':
    main()
