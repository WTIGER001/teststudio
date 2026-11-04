import argparse
import hashlib
import json
import os
import re
import struct
import subprocess
import tarfile
import tempfile
import time
import uuid
from pathlib import Path


def docker(*args, check=True):
    return subprocess.run(['docker', *args], check=check, capture_output=True, text=True, timeout=40)


def elf_status(data, architecture):
    assert data[:6] == b'\x7fELF\x02\x01', 'Relay binaries require little-endian ELF64.'
    machine = struct.unpack_from('<H', data, 18)[0]
    assert machine == {'amd64': 62, 'arm64': 183}[architecture], 'Binary architecture differs from the image.'
    offset = struct.unpack_from('<Q', data, 32)[0]
    size, count = struct.unpack_from('<HH', data, 54)
    assert size >= 56 and offset + size * count <= len(data), 'ELF program table exceeds the inspected header.'
    assert all(struct.unpack_from('<I', data, offset + size * index)[0] != 3 for index in range(count)), 'Scratch binaries cannot require a dynamic interpreter.'
    return {'machine': machine, 'dynamicInterpreter': False}


def main():
    parser = argparse.ArgumentParser(description='Check one immutable relay image locally without UI, cloud, or runner builds.')
    parser.add_argument('--build-result', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    build = json.loads(args.build_result.read_text())
    assert re.fullmatch(r'sha256:[a-f0-9]{64}', build['image']), 'Supply the immutable builder result.'
    assert build['architecture'] in ('amd64', 'arm64') and build['localOnly'], 'Verification requires a local-only build result.'
    endpoint = docker('context', 'inspect', docker('context', 'show').stdout.strip(), '--format', '{{.Endpoints.docker.Host}}').stdout.strip()
    assert all(value.startswith(('unix://', 'npipe://')) for value in (endpoint, os.environ.get('DOCKER_HOST', endpoint))), 'Verification requires a local Docker socket.'
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    name = 'teststudio-relay-check-' + uuid.uuid4().hex[:16]
    summary = {'image': build['image'], 'architecture': build['architecture'], 'passed': False, 'checks': [], 'cloudCalls': False, 'uiBuilds': False, 'browserChecks': False}
    created = False
    try:
        image = json.loads(docker('image', 'inspect', build['image']).stdout)[0]
        assert image['Architecture'] == build['architecture']
        assert image['Config']['User'] == '65532:65532'
        assert image['Config']['Labels']['teststudio.package'] == 'worker-relay/v1'
        assert image['Config']['Labels']['teststudio.source-key'] == build['sourceKey']
        assert image['Config'].get('Volumes') in (None, {})
        assert image['Config'].get('Env') in (None, [], ['PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin']), 'Image environment contains configuration beyond Docker default PATH.'
        created = True
        docker('run', '--detach', '--name', name, '--platform=linux/' + build['architecture'], '--pull=never', '--network=none', '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges', '--pids-limit=64', '--memory=67108864', '--cpus=0.5', build['image'], '--public-origin=https://workers.example.test', '--backend-origin=https://controller-workers.example.test')
        ready = False
        for _ in range(30):
            probe = docker('exec', name, '/app/teststudio-health', '--url=http://127.0.0.1:8043/healthz', check=False)
            if probe.returncode == 0:
                ready = True
                break
            state = json.loads(docker('container', 'inspect', name).stdout)[0]['State']
            assert state['Running'], 'Owned relay container stopped before process health.'
            time.sleep(0.1)
        assert ready, 'Relay process health did not become available.'
        docker('exec', name, '/app/teststudio-health', '--url=http://127.0.0.1:8043/livez')
        container = json.loads(docker('container', 'inspect', name).stdout)[0]
        assert container['HostConfig']['ReadonlyRootfs']
        assert container['HostConfig']['CapDrop'] == ['ALL']
        assert container['HostConfig']['NetworkMode'] == 'none'
        assert container['Mounts'] == []
        assert container['Config']['User'] == '65532:65532'
        assert container['HostConfig']['Memory'] == 67108864
        assert container['HostConfig']['PidsLimit'] == 64
        summary['checks'] += ['Immutable image metadata and selected architecture', 'Non-root process health with read-only root, no capabilities, no network, and no writable mounts', 'Independent local health and liveness probes']
        expected = {'app/teststudio-exchange': build['artifacts']['binaries']['teststudio-exchange']['hash'], 'app/teststudio-health': build['artifacts']['binaries']['teststudio-health']['hash'], 'etc/ssl/certs/ca-certificates.crt': build['artifacts']['certificateHash']}
        injected = {'.dockerenv', 'etc/hosts', 'etc/hostname', 'etc/resolv.conf'}
        devices = {'dev/console', 'dev/null', 'dev/random', 'dev/tty', 'dev/urandom', 'dev/zero'}
        found = {}
        with tempfile.TemporaryDirectory(prefix='teststudio-relay-export-') as temporary:
            archive = Path(temporary) / 'rootfs.tar'
            docker('export', '--output', str(archive), name)
            with tarfile.open(archive) as files:
                for member in files:
                    filename = member.name.removeprefix('./').removeprefix('/')
                    if not member.isfile():
                        continue
                    if filename in devices and member.size == 0:
                        continue
                    assert filename in expected or filename in injected, 'Unexpected regular file in the minimal relay package: ' + filename
                    if filename not in expected:
                        continue
                    required_mode = 0o005 if filename.startswith('app/') else 0o004
                    assert member.mode & required_mode == required_mode, 'Packaged artifact is not usable by the non-root process.'
                    source = files.extractfile(member)
                    digest = hashlib.sha256()
                    header = source.read(128 << 10)
                    digest.update(header)
                    for chunk in iter(lambda: source.read(128 << 10), b''):
                        digest.update(chunk)
                    assert digest.hexdigest() == expected[filename], 'Packaged artifact differs from its built content.'
                    found[filename] = {'sha256': digest.hexdigest()}
                    if filename.startswith('app/'):
                        found[filename].update(elf_status(header, build['architecture']))
        assert set(found) == set(expected), 'Required relay artifacts are missing.'
        summary['artifacts'] = found
        summary['checks'] += ['Only relay, health binary, approved CA bundle, and known Docker runtime files appear in the exported filesystem', 'Packaged file digests match the builder result', 'Both Linux executables match the target architecture and need no dynamic interpreter']
        docker('stop', '--time=10', name)
        stopped = json.loads(docker('container', 'inspect', name).stdout)[0]['State']
        assert not stopped['Running'] and stopped['ExitCode'] == 0, 'Relay did not stop gracefully.'
        summary['checks'].append('Graceful container shutdown exits successfully')
        summary['passed'] = True
    finally:
        if created:
            docker('rm', '--force', '--volumes', name, check=False)
            absent = docker('container', 'inspect', name, check=False)
            summary['cleanup'] = {'container': name, 'absent': absent.returncode != 0 and 'no such' in absent.stderr.lower()}
        else:
            summary['cleanup'] = {'container': name, 'absent': True}
        (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        if not summary['cleanup']['absent']:
            raise RuntimeError('The owned relay verification container remains; inspect summary.json.')
    print(json.dumps({'passed': summary['passed'], 'checks': len(summary['checks']), 'output': str(output)}))


if __name__ == '__main__':
    main()
