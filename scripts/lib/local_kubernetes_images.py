import json
import os
import re
import subprocess


def import_image(cluster, installation, tag, source, destination):
    assert re.fullmatch('[a-z0-9-]+', tag)
    image = json.loads(subprocess.check_output(['docker', 'image', 'inspect', source], text=True))[0]
    assert image['Architecture'] == 'arm64', 'The owned cluster requires a verified ARM64 image'
    archive = installation / (tag + '-import.tar')
    node_archive = '/tmp/teststudio-' + tag + '-import.tar'
    local_tag = 'teststudio/' + tag + ':local-k3s'
    subprocess.run(['docker', 'tag', source, local_tag], check=True)
    try:
        subprocess.run(['docker', 'save', '--output', str(archive), local_tag], check=True)
        os.chmod(archive, 0o600)
        subprocess.run(cluster.command[:3] + ['cp', str(archive), cluster.command[-1] + ':' + node_archive], check=True)
        cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'import', '--platform', 'linux/arm64', node_archive)
        rows = [row.split() for row in cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'list').splitlines() if row.startswith('docker.io/' + local_tag + ' ')]
        assert len(rows) == 1 and rows[0][2].startswith('sha256:')
        reference = destination + '@' + rows[0][2]
        cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'tag', '--force', 'docker.io/' + local_tag, reference)
        return reference
    finally:
        cluster.execute('rm', '-f', node_archive)
        archive.unlink(missing_ok=True)
