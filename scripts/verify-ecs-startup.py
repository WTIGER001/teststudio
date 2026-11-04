import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description='Check ECS startup guards using native binaries without account access or cloud credential discovery.')
    parser.add_argument('--server', type=Path, required=True)
    parser.add_argument('--preparation', type=Path, required=True)
    parser.add_argument('--settings', type=Path, default=Path('deploy/aws/worker-settings.example.json'))
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    settings = json.loads(args.settings.read_text())
    checks = []
    with tempfile.TemporaryDirectory(prefix='teststudio-ecs-startup-') as directory:
        root = Path(directory)
        database = root / 'unopened.db'
        settings_file = root / 'settings.json'
        settings_file.write_text(json.dumps(settings))
        environment = {key: os.environ[key] for key in ['PATH', 'HOME', 'TMPDIR', 'TEMP', 'TMP'] if key in os.environ}
        environment.update({'TESTSTUDIO_ECS_CONFIGURATION': json.dumps(settings), 'AWS_EC2_METADATA_DISABLED': 'true', 'AWS_CONFIG_FILE': str(root / 'absent-config'), 'AWS_SHARED_CREDENTIALS_FILE': str(root / 'absent-credentials')})
        common = ['--database=' + str(database), '--ecs-workers-env=TESTSTUDIO_ECS_CONFIGURATION']
        server = [str(args.server.resolve()), *common, '--execution-location=ecs', '--worker-addr=127.0.0.1:8042', '--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test']
        preparation = [str(args.preparation.resolve()), *common, '--execution-location=ecs']

        def verify(label, command, expected, env=None):
            result = subprocess.run(command, env=env or environment, capture_output=True, text=True, timeout=8)
            (args.output / (label + '.log')).write_text(result.stderr)
            assert result.returncode != 0 and expected in result.stderr, label + ' did not reject invalid startup'
            assert 'flag provided but not defined' not in result.stderr and 'private-canary' not in result.stderr
            assert not database.exists(), label + ' initialized storage before rejecting startup'
            checks.append(label)

        verify('api-requires-postgres', server, 'requires a PostgreSQL')
        verify('api-requires-private-port', [*server, '--worker-addr=127.0.0.1:8043'], 'private listener on port 8042')
        verify('api-requires-ecs-location', [*server, '--execution-location=local'], 'ECS workers require their execution location')
        verify('api-rejects-local-runner', [*server, '--go-images=sha256:' + 'a' * 64], 'without local coded execution')
        verify('api-rejects-equivalent-public-origin', [*server, '--public-url=' + settings['worker']['exchangeOrigin'].upper().replace('HTTPS:', 'https:') + ':443'], 'private worker origin must differ')
        verify('api-artifact-role-rejected', [*server, '--worker-addr=', '--serve-role=artifacts'], 'artifact service role cannot configure hosted execution')
        verify('api-one-provider', [*server, '--kubernetes-workers-config=' + str(root / 'absent-provider')], 'choose one hosted execution provider')
        verify('preparation-cleanup-only-requires-postgres', preparation, 'requires a PostgreSQL')
        verify('preparation-requires-ecs-location', [*preparation, '--execution-location=local'], 'ECS preparation requires its execution location')
        verify('preparation-rejects-local-runtime', [*preparation, '--go-runtime=sha256:' + 'a' * 64], 'without local source runtime flags')
        verify('settings-one-source', [*server, '--ecs-workers-config=' + str(settings_file)], 'requires either a file or an environment reference')
        invalid = {**environment, 'TESTSTUDIO_ECS_CONFIGURATION': json.dumps({**settings, 'private-canary': 'must-not-appear'})}
        verify('settings-reject-private-unknown-field', server, 'valid JSON with supported fields', invalid)

    summary = {'passed': True, 'cloudOperations': False, 'credentialDiscovery': False, 'checks': checks, 'nativeBinarySHA256': {label: hashlib.sha256(path.read_bytes()).hexdigest() for label, path in [('server', args.server), ('preparation', args.preparation)]}}
    (args.output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps({'passed': True, 'checks': len(checks), 'cloudOperations': False}))


if __name__ == '__main__':
    main()
