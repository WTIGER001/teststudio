import argparse
import json
import os
from pathlib import Path
import re
import subprocess

from lib.local_kubernetes import LocalAPI
from lib.local_kubernetes_cluster import LocalCluster
from lib.local_kubernetes_images import import_image


def main():
    parser = argparse.ArgumentParser(description='Enable desktop recording only on the owned local k3s installation')
    parser.add_argument('--image', required=True)
    arguments = parser.parse_args()
    assert re.fullmatch('sha256:[a-f0-9]{64}', arguments.image)
    root = Path.cwd()
    installation = root / '.data/local-k3s'
    output = root / 'output/local-k3s/recording'
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    cluster = LocalCluster(installation)
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']
    state_file = output / 'installation.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {}
    if state:
        assert state['sourceImage'] == arguments.image, 'Observe the existing recording upgrade before selecting another image'
    state['sourceImage'] = arguments.image

    def checkpoint():
        state_file.write_text(json.dumps(state, indent=2) + '\n')

    nodes = json.loads(cluster.execute(*kube, 'get', 'nodes', '-o', 'json'))['items']
    assert len(nodes) == 1 and nodes[0]['metadata']['name'] == 'k3d-teststudio-local-server-0'
    assert any(row['type'] == 'Ready' and row['status'] == 'True' for row in nodes[0]['status']['conditions'])
    assert all(pod['status']['phase'] in ['Succeeded', 'Failed'] for pod in cluster.pods('teststudio-workers')), 'Drain workers before upgrading the service'
    selected = installation / 'controller-values.json'
    previous = installation / 'before-recording-values.json'
    if not previous.exists():
        previous.write_bytes(selected.read_bytes())
        os.chmod(previous, 0o600)
    values = json.loads(previous.read_text())
    assert values['controller']['enabled'] and values['controller']['replicas'] == 2 and values['workers']['gateway']['enabled']
    assert values['persistence']['existingClaim'] == 'teststudio-shared-evidence'
    if 'previousRuns' not in state:
        api = LocalAPI(installation)
        api.login()
        try:
            state['previousRuns'] = {project: sorted(row['id'] for row in api.call('GET', '/api/v1/projects/' + project + '/runs')['items']) for project in ['k3s-coded', 'k3s-network']}
            assert len(state['previousRuns']['k3s-coded']) == 12
        finally:
            api.logout()
        checkpoint()
    if not state.get('reference'):
        state['reference'] = import_image(cluster, installation, 'recording-service', arguments.image, 'docker.io/teststudio/service')
        checkpoint()
    assert state['reference'] in cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'list')
    values['image'].update(repository='docker.io/teststudio/service', digest=state['reference'].split('@', 1)[1])
    values['recordingCompanion'] = True
    candidate = installation / 'recording-values.json'
    candidate.write_text(json.dumps(values, indent=2) + '\n')
    os.chmod(candidate, 0o600)
    environment = {key: value for key, value in os.environ.items() if not key.startswith(('HELM_', 'KUBE'))}
    environment['HELM_DRIVER'] = 'secret'
    helm = [str(root / '.data/kubernetes-tools/helm')]
    context = ['--namespace', 'teststudio-local', '--kubeconfig', str(installation / 'kubeconfig.yaml'), '--kube-context', 'k3d-teststudio-local']
    history = json.loads(subprocess.check_output(helm + ['history', 'teststudio', *context, '--output', 'json'], env=environment, text=True))
    if state.get('upgradeAttempted'):
        revision = next((row for row in history if row['revision'] == state['expectedRevision']), None)
        assert revision and revision['status'] == 'deployed', 'Observe the saved Helm revision; do not replay an uncertain upgrade'
    else:
        current = max(history, key=lambda row: row['revision'])
        assert current['status'] == 'deployed'
        state.update(upgradeAttempted=True, previousRevision=current['revision'], expectedRevision=current['revision'] + 1)
        checkpoint()
        print('Upgrading only the owned local service image and recorder setting.', flush=True)
        with (output / 'helm-upgrade.log').open('w') as log:
            subprocess.run(helm + ['upgrade', 'teststudio', str(root / 'deploy/kubernetes'), *context, '--values', str(candidate), '--wait', '--timeout', '120s'], check=True, env=environment, stdout=log, stderr=subprocess.STDOUT, timeout=150)
    for name in ['teststudio', 'teststudio-controller', 'teststudio-prepare']:
        cluster.execute(*kube, 'rollout', 'status', 'deployment/' + name, '-n', 'teststudio-local', '--timeout=15s')
    deployments = json.loads(cluster.execute(*kube, 'get', 'deployments', '-n', 'teststudio-local', '-o', 'json'))['items']
    proof = []
    for item in deployments:
        if item['metadata']['name'] not in ['teststudio', 'teststudio-controller', 'teststudio-prepare']:
            continue
        container = item['spec']['template']['spec']['containers'][0]
        enabled = '--recording-companion=true' in container['args']
        assert enabled == (item['metadata']['name'] == 'teststudio')
        assert container['image'] == state['reference'] and item['status']['readyReplicas'] == item['spec']['replicas']
        proof.append({'name': item['metadata']['name'], 'replicas': item['spec']['replicas'], 'ready': item['status']['readyReplicas'], 'recordingCompanion': enabled, 'image': container['image']})
    assert len(proof) == 3
    selected.write_bytes(candidate.read_bytes())
    os.chmod(selected, 0o600)
    state.update(deployed=True, helmRevision=state['expectedRevision'], valuesFile='.data/local-k3s/controller-values.json', deployments=proof, cloudOperations=False)
    checkpoint()
    print(json.dumps({'deployed': True, 'helmRevision': state['helmRevision'], 'deployments': proof, 'cloudOperations': False}), flush=True)


if __name__ == '__main__':
    main()
