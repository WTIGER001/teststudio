import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import uuid


def main():
    parser = argparse.ArgumentParser(description='Check hosted preparation startup locally without registry or cluster access.')
    parser.add_argument('--native-build', type=Path, required=True)
    parser.add_argument('--service-image', type=Path, required=True)
    parser.add_argument('--settings', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    native = json.loads(args.native_build.read_text())
    image = json.loads(args.service_image.read_text())['image']
    settings = json.loads(args.settings.read_text())
    checks, cleanup = [], {}

    def verify(command, expected, label):
        result = subprocess.run(command, capture_output=True, text=True, timeout=15)
        (args.output / (label + '.log')).write_text(result.stderr)
        assert result.returncode != 0 and expected in result.stderr, label + ' did not reject invalid startup'
        assert 'flag provided but not defined' not in result.stderr
        checks.append(label)

    with tempfile.TemporaryDirectory(prefix='teststudio-preparation-startup-') as directory:
        root = Path(directory)
        config = root / 'settings.json'
        config.write_text(json.dumps(settings))
        database = root / 'unopened.db'
        common = ['--database=' + str(database)]
        worker = ['--kubernetes-workers-config=' + str(config), '--execution-location=kubernetes']
        verify([native['preparation'], *common, *worker], 'requires a PostgreSQL', 'native-postgres-required')
        verify([native['preparation'], *common, '--kubernetes-workers-config=' + str(config)], 'requires Kubernetes settings', 'native-location-required')
        verify([native['preparation'], *common, *worker, '--go-runtime=sha256:' + 'a' * 64], 'without local source runtime flags', 'native-local-runtime-rejected')
        verify([native['preparation'], *common, '--registry-go-runtime=registry.example.test/teststudio/source@sha256:' + 'a' * 64], 'requires Kubernetes settings', 'native-registry-settings-required')
        verify([native['server'], *common, *worker, '--prepared-sources', '--source-bundles', '--source-bundle-runners=go', '--worker-addr=127.0.0.1:8042', '--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test'], 'requires a PostgreSQL', 'native-api-discovery-profile')
        verify([native['server'], *common, '--source-bundles', '--source-bundle-runners=go,node'], 'source bundle runners must select', 'native-invalid-source-runners')
        assert not database.exists(), 'rejected startup initialized a database'

    for label, entrypoint, extra, expected in [
        ('image-postgres-required', '/app/teststudio-prepare', ['--execution-location=kubernetes'], 'requires a PostgreSQL'),
        ('image-local-runtime-rejected', '/app/teststudio-prepare', ['--execution-location=kubernetes', '--playwright-runtime=sha256:' + 'a' * 64], 'without local source runtime flags'),
        ('image-single-source-runner', '/app/teststudio', ['--execution-location=kubernetes', '--prepared-sources', '--source-bundles', '--source-bundle-runners=go', '--worker-addr=0.0.0.0:8042', '--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test'], 'requires a PostgreSQL'),
    ]:
        name = 'teststudio-preparation-guard-' + uuid.uuid4().hex[:12]
        try:
            verify(['docker', 'run', '--rm', '--name', name, '--network=none', '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--no-healthcheck', '--env', 'TESTSTUDIO_WORKER_CONFIGURATION=' + json.dumps(settings), '--entrypoint=' + entrypoint, image, '--database=/tmp/unopened.db', '--kubernetes-workers-env=TESTSTUDIO_WORKER_CONFIGURATION', *extra], expected, label)
        finally:
            subprocess.run(['docker', 'rm', '-fv', name], capture_output=True, text=True, timeout=10)
            state = subprocess.run(['docker', 'container', 'inspect', name], capture_output=True, text=True, timeout=10)
            cleanup[name] = state.returncode != 0 and 'no such' in state.stderr.lower()
            (args.output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
            assert cleanup[name], 'owned preparation probe remains after cleanup'
    binaries = {key: hashlib.sha256(Path(native[key]).read_bytes()).hexdigest() for key in ['server', 'preparation']}
    summary = {'passed': True, 'clusterContact': False, 'registryContact': False, 'image': image, 'nativeBinarySHA256': binaries, 'checks': checks}
    (args.output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps({'passed': True, 'checks': len(checks), 'clusterContact': False}))


if __name__ == '__main__':
    main()
