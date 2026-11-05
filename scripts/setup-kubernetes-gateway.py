import json
import os
from pathlib import Path
import subprocess

from lib.kubernetes_gateway_fixture import configure_fixture
from lib.local_kubernetes_cluster import LocalCluster


def main():
    root = Path.cwd()
    installation = root / '.data/local-k3s'
    output = root / 'output/local-k3s/gateway'
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    cluster = LocalCluster(installation)
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']
    state_file = output / 'installation.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {}

    def checkpoint():
        state_file.write_text(json.dumps(state, indent=2) + '\n')

    images = json.loads((output / 'images.json').read_text())
    selected = installation / 'controller-values.json'
    rollback = installation / 'before-gateway-values.json'
    if not rollback.exists():
        rollback.write_bytes(selected.read_bytes())
        os.chmod(rollback, 0o600)
    values = json.loads(rollback.read_text())
    assert values['controller']['enabled'] and values['controller']['replicas'] == 2
    assert values['oidc']['providers'] and values['persistence']['existingClaim'] == 'teststudio-shared-evidence'
    assert not values.get('secretBindings', {}).get('existingSecret'), 'Merge existing credential catalogs explicitly before enabling this fixture'
    state.update(configure_fixture(root, cluster, images))
    checkpoint()
    values['image'].update(repository='docker.io/teststudio/service', digest=images['gateway-service']['reference'].split('@', 1)[1])
    values['workers']['gateway'] = {'enabled': True, 'image': images['gateway']['reference'], 'serviceAccount': 'teststudio-gateway', 'tlsSecret': 'teststudio-gateway-tls', 'exchangeTrustSecret': 'teststudio-gateway-exchange-ca', 'caSecret': 'teststudio-gateway-ca'}
    values['preparation'].update(goRuntime=images['go-runtime']['reference'], playwrightRuntime=images['playwright-runtime']['reference'])
    values['preparation'].setdefault('resources', {}).setdefault('requests', {})['cpu'] = '250m'
    values['secretBindings'] = {'existingSecret': 'teststudio-gateway-credentials'}
    candidate = installation / 'gateway-values.json'
    candidate.write_text(json.dumps(values, indent=2) + '\n')
    os.chmod(candidate, 0o600)
    environment = {key: value for key, value in os.environ.items() if not key.startswith(('HELM_', 'KUBE'))}
    environment['HELM_DRIVER'] = 'secret'
    helm = [str(root / '.data/kubernetes-tools/helm')]
    context = ['--namespace', 'teststudio-local', '--kubeconfig', str(installation / 'kubeconfig.yaml'), '--kube-context', 'k3d-teststudio-local']
    if state.get('deployed') and json.loads(selected.read_text()) != values:
        history = json.loads(subprocess.check_output(helm + ['history', 'teststudio', *context, '--output', 'json'], env=environment, text=True))
        previous = next((row for row in history if row['revision'] == state['expectedRevision']), None)
        assert previous and previous['status'] == 'deployed', 'Inspect the previous revision before preparing a changed installation'
        state.setdefault('previousRevisions', []).append(state['expectedRevision'])
        state.update(upgradeAttempted=False, deployed=False)
        checkpoint()
    if state.get('upgradeAttempted'):
        history = json.loads(subprocess.check_output(helm + ['history', 'teststudio', *context, '--output', 'json'], env=environment, text=True))
        revision = next((row for row in history if row['revision'] == state['expectedRevision']), None)
        assert revision and revision['status'] == 'deployed', 'Observe the saved Helm revision; do not repeat an uncertain upgrade'
    else:
        workers = cluster.pods('teststudio-workers')
        assert all(pod['status']['phase'] in ['Succeeded', 'Failed'] and all('terminated' in row.get('state', {}) for row in pod['status'].get('containerStatuses', [])) for pod in workers), 'Drain active workers before changing this installation'
        history = json.loads(subprocess.check_output(helm + ['history', 'teststudio', *context, '--output', 'json'], env=environment, text=True))
        state.update(upgradeAttempted=True, expectedRevision=max(row['revision'] for row in history) + 1)
        checkpoint()
        print('Upgrading the owned local installation.', flush=True)
        with (output / 'helm-upgrade.log').open('w') as log:
            subprocess.run(helm + ['upgrade', 'teststudio', str(root / 'deploy/kubernetes'), *context, '--values', str(candidate), '--wait', '--timeout', '120s'], check=True, env=environment, stdout=log, stderr=subprocess.STDOUT)
    for name in ['gateway-target', 'teststudio', 'teststudio-controller', 'teststudio-prepare']:
        cluster.execute(*kube, 'rollout', 'status', 'deployment/' + name, '-n', 'teststudio-local', '--timeout=15s')
    selected.write_bytes(candidate.read_bytes())
    os.chmod(selected, 0o600)
    state.update(deployed=True, valuesFile='.data/local-k3s/controller-values.json', helmRevision=state['expectedRevision'], controllerReplicas=2, cloudOperations=False)
    checkpoint()
    print(json.dumps(state), flush=True)


if __name__ == '__main__':
    main()
