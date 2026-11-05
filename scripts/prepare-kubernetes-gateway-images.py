import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import io
import tarfile
import urllib.request

from lib.local_kubernetes_cluster import LocalCluster
from lib.local_kubernetes_images import import_image


def main():
    root = Path.cwd()
    installation = root / '.data/local-k3s'
    output = root / 'output/local-k3s/gateway'
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    cluster = LocalCluster(installation)
    state_file = output / 'images.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {}

    def checkpoint():
        state_file.write_text(json.dumps(state, indent=2) + '\n')

    def imported(name, image, destination):
        previous = state.get(name, {})
        if previous.get('image') == image and previous['reference'] in cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'list'):
            return
        print('Importing owned local image: ' + name, flush=True)
        reference = import_image(cluster, installation, name, image, destination)
        state[name] = {'image': image, 'reference': reference}
        checkpoint()

    service = json.loads((root / 'output/local-k3s/service-gateway-build.json').read_text())['image']
    imported('gateway-service', service, 'docker.io/teststudio/service')
    source = root / 'tests/kubernetes/environment'
    key = hashlib.sha256((source / 'main.go').read_bytes() + (source / 'Dockerfile').read_bytes() + subprocess.check_output(['go', 'version'])).hexdigest()
    if state.get('targetBuild', {}).get('key') != key:
        with tempfile.TemporaryDirectory(prefix='teststudio-gateway-target-') as directory:
            context = Path(directory)
            subprocess.run(['go', 'build', '-trimpath', '-o', str(context / 'target'), str(source / 'main.go')], env={**os.environ, 'CGO_ENABLED': '0', 'GOOS': 'linux', 'GOARCH': 'arm64'}, check=True)
            subprocess.run(['docker', 'build', '--pull=false', '-q', '-f', str(source / 'Dockerfile'), '-t', 'teststudio/gateway-target:local', directory], check=True)
        image = subprocess.check_output(['docker', 'image', 'inspect', 'teststudio/gateway-target:local', '--format', '{{.Id}}'], text=True).strip()
        state['targetBuild'] = {'key': key, 'image': image}
        checkpoint()
    imported('gateway-target', state['targetBuild']['image'], 'docker.io/teststudio/gateway-target')
    for runner in ['go', 'playwright']:
        base = json.loads((root / ('output/local-k3s/' + runner + '-gateway-runtime.json')).read_text())['image']
        name = runner + 'Trusted'
        context = installation / 'trusted-runtime-build'
        key = hashlib.sha256(base.encode() + (context / 'Dockerfile').read_bytes() + (context / 'exchange-ca.crt').read_bytes()).hexdigest()
        if state.get(name, {}).get('key') != key:
            print('Adding the local fixture CA: ' + runner, flush=True)
            tag = 'teststudio/' + runner + '-gateway-trusted:local'
            subprocess.run(['docker', 'build', '--pull=false', '-q', '--build-arg', 'RUNTIME_IMAGE=' + base, '-t', tag, str(context)], check=True)
            image = subprocess.check_output(['docker', 'image', 'inspect', tag, '--format', '{{.Id}}'], text=True).strip()
            state[name] = {'key': key, 'image': image, 'base': base}
            checkpoint()
    gateway = (root / 'output/local-k3s/gateway-image.txt').read_text().strip()
    images = {'gateway': gateway, 'go-runtime': state['goTrusted']['image'], 'playwright-runtime': state['playwrightTrusted']['image']}
    pending = {name: image for name, image in images.items() if state.get(name, {}).get('image') != image}
    if pending:
        tool = installation / 'tools/crane'
        metadata = json.loads((installation / 'crane-tool.json').read_text())
        binary_hash = hashlib.sha256(tool.read_bytes()).hexdigest()
        if not metadata.get('binarySha256'):
            assert metadata['url'] == 'https://github.com/google/go-containerregistry/releases/download/v0.21.5/go-containerregistry_Linux_arm64.tar.gz'
            with urllib.request.urlopen(metadata['url'], timeout=30) as response:
                archive = response.read(32 << 20)
            assert hashlib.sha256(archive).hexdigest() == metadata['sha256']
            with tarfile.open(fileobj=io.BytesIO(archive), mode='r:gz') as bundle:
                assert hashlib.sha256(bundle.extractfile('crane').read()).hexdigest() == binary_hash
            metadata['binarySha256'] = binary_hash
            (installation / 'crane-tool.json').write_text(json.dumps(metadata, indent=2) + '\n')
        assert binary_hash == metadata['binarySha256']
        node = cluster.execute('mktemp', '-d', '/tmp/teststudio-gateway-publisher-XXXXXXXXXX').strip()
        assert re.fullmatch('/tmp/teststudio-gateway-publisher-[A-Za-z0-9]{10}', node)
        try:
            for file, name in [(tool, 'crane'), (installation / 'registry-publisher/config.json', 'config.json'), (installation / 'fixture-tls/ca.crt', 'ca.crt')]:
                subprocess.run(cluster.command[:3] + ['cp', str(file), cluster.command[-1] + ':' + node + '/' + name], check=True)
            cluster.execute('chmod', '700', node + '/crane')
            cluster.execute('chmod', '600', node + '/config.json')
            for name, image in pending.items():
                assert re.fullmatch('sha256:[a-f0-9]{64}', image)
                archive = installation / ('gateway-publish-' + name + '.tar')
                node_archive = node + '/' + name + '.tar'
                destination = 'registry.teststudio.test/teststudio/' + name + ':gateway-' + image[7:23]
                print('Publishing immutable gateway profile: ' + name, flush=True)
                try:
                    subprocess.run(['docker', 'tag', image, 'teststudio/' + name + ':gateway-local'], check=True)
                    subprocess.run(['docker', 'save', '--output', str(archive), 'teststudio/' + name + ':gateway-local'], check=True)
                    os.chmod(archive, 0o600)
                    subprocess.run(cluster.command[:3] + ['cp', str(archive), cluster.command[-1] + ':' + node_archive], check=True)
                    command = ['env', 'DOCKER_CONFIG=' + node, 'SSL_CERT_FILE=' + node + '/ca.crt', node + '/crane']
                    subprocess.run(cluster.command + command + ['push', node_archive, destination], check=True, timeout=360)
                    digest = cluster.execute(*command, 'digest', destination).strip()
                    assert re.fullmatch('sha256:[a-f0-9]{64}', digest)
                    state[name] = {'image': image, 'reference': destination.split(':', 1)[0] + '@' + digest}
                    checkpoint()
                finally:
                    archive.unlink(missing_ok=True)
                    cluster.execute('rm', '-f', node_archive)
        finally:
            cluster.execute('rm', '-rf', node)
    print(json.dumps({'imagesReady': True, 'published': sorted(images), 'cloudOperations': False}), flush=True)


if __name__ == '__main__':
    main()
