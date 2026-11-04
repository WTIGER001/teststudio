import argparse
import io
import json
from pathlib import Path
import sqlite3
import subprocess
import tarfile
import tempfile
import time
import uuid


def docker(*arguments, check=True):
    return subprocess.run(['docker', *arguments], check=check, capture_output=True, text=True, timeout=30)


def main():
    parser = argparse.ArgumentParser(description='Check changed ECS flags and journal-volume ownership in one local service image. No cloud or source tests run.')
    parser.add_argument('--image', type=Path, required=True)
    parser.add_argument('--fixture', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output = args.output.resolve()
    args.output.mkdir(parents=True, exist_ok=True, mode=0o700)
    image = json.loads(args.image.read_text())['image']
    settings = json.loads(args.fixture.read_text())['settings']
    containers, volumes, cleanup, checks = [], set(), {}, []
    try:
        for role in ['api', 'preparation']:
            name = 'teststudio-aws-guard-' + uuid.uuid4().hex[:12]
            containers.append(name)
            extra = ['--entrypoint=/app/teststudio-prepare'] if role == 'preparation' else []
            flags = ['--database-env=TESTSTUDIO_DATABASE', '--execution-location=ecs', '--ecs-workers-env=TESTSTUDIO_ECS_CONFIGURATION']
            if role == 'api':
                flags += ['--worker-addr=0.0.0.0:8042', '--public-url=https://studio.example.test', '--artifact-origin=https://evidence.example.test']
            docker('create', '--name', name, '--read-only', '--network=none', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--no-healthcheck', '--env', 'TESTSTUDIO_DATABASE=/tmp/unopened.db', '--env', 'TESTSTUDIO_ECS_CONFIGURATION=' + json.dumps(settings), *extra, image, *flags)
            metadata = json.loads(docker('inspect', name).stdout)[0]
            volumes.update(mount['Name'] for mount in metadata['Mounts'] if mount['Type'] == 'volume')
            result = docker('start', '--attach', name, check=False)
            (args.output / (role + '-startup.log')).write_text(result.stdout + result.stderr)
            status = json.loads(docker('inspect', name).stdout)[0]['State']
            assert not status['Running'] and status['ExitCode'] != 0 and 'requires a PostgreSQL' in result.stderr
            assert 'flag provided but not defined' not in result.stderr
            checks.append(role + ': packaged ECS flags reject SQLite before credentials or provider access')
        name = 'teststudio-aws-journals-' + uuid.uuid4().hex[:12]
        containers.append(name)
        journal = name + '-data'
        docker('volume', 'create', '--label', 'teststudio.verification=aws-journals', journal)
        volumes.add(journal)
        docker('create', '--name', name, '--read-only', '--network=none', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--no-healthcheck', '--mount', 'type=volume,source=' + journal + ',target=/private/source-journals', '--entrypoint=/app/teststudio-prepare', image, '--database=/tmp/probe.db', '--execution-location=local', '--build-directory=/private/source-journals/ownership-probe')
        metadata = json.loads(docker('inspect', name).stdout)[0]
        volumes.update(mount['Name'] for mount in metadata['Mounts'] if mount['Type'] == 'volume')
        docker('start', name)
        observed = False
        for _ in range(60):
            status = json.loads(docker('inspect', name).stdout)[0]['State']
            assert status['Running'], 'journal probe stopped before initialization'
            archive = subprocess.run(['docker', 'cp', name + ':/private/source-journals/ownership-probe', '-'], capture_output=True, timeout=10)
            if archive.returncode == 0:
                with tarfile.open(fileobj=io.BytesIO(archive.stdout), mode='r:*') as data:
                    directory = next(item for item in data.getmembers() if item.isdir())
                    assert directory.uid == directory.gid == 65532 and directory.mode == 0o700
                    observed = True
                break
            time.sleep(0.05)
        assert observed, 'non-root preparation could not initialize its mounted journal child'
        docker('stop', '--time=10', name)
        with tempfile.TemporaryDirectory(prefix='journal-db-', dir=args.output) as temporary:
            database = Path(temporary) / 'probe.db'
            docker('cp', name + ':/tmp/probe.db', str(database))
            with sqlite3.connect('file:' + str(database) + '?mode=ro&immutable=1', uri=True) as connection:
                count = connection.execute("SELECT count(*) FROM sqlite_master WHERE type='table'").fetchone()[0]
                assert count > 1, 'preparation did not initialize its isolated local store'
        checks.append('non-root preparation initializes a private journal child on a real local volume with a read-only root filesystem')
        (args.output / 'summary.json').write_text(json.dumps({'passed': True, 'image': image, 'cloudOperations': False, 'sourceRuns': False, 'efsVerification': False, 'checks': checks}, indent=2) + '\n')
    finally:
        for name in containers:
            docker('rm', '--force', '--volumes', name, check=False)
            cleanup[name] = docker('inspect', name, check=False).returncode != 0
        for name in volumes:
            docker('volume', 'rm', name, check=False)
            cleanup[name] = docker('volume', 'inspect', name, check=False).returncode != 0
        (args.output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
        assert all(cleanup.values()), 'owned package resources remain'
    print(json.dumps({'passed': True, 'checks': len(checks), 'cloudOperations': False}))


if __name__ == '__main__':
    main()
