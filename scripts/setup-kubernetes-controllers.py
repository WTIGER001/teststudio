import json
import hashlib
import os
from pathlib import Path
import subprocess
import time

from lib.local_kubernetes_cluster import LocalCluster


def main():
    root = Path.cwd()
    output = root / 'output/local-k3s/controllers'
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    installation = root / '.data/local-k3s'
    cluster = LocalCluster(installation)
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']
    state_file = output / 'installation.json'
    state = json.loads(state_file.read_text()) if state_file.exists() else {'namespace': 'teststudio-local', 'previousClaim': 'teststudio-artifacts'}

    def checkpoint():
        state_file.write_text(json.dumps(state, indent=2) + '\n')

    def apply(objects):
        subprocess.run(cluster.command[:-1] + ['-i', cluster.command[-1]] + kube + ['apply', '-f', '-'], input=json.dumps({'apiVersion': 'v1', 'kind': 'List', 'items': objects}), text=True, check=True, timeout=30)

    def imported(tag, source, destination):
        if state.get(tag, {}).get('source') == source:
            rows = cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'list')
            if state[tag]['reference'] in rows:
                return state[tag]['reference']
        archive = installation / (tag + '-import.tar')
        node_archive = '/tmp/teststudio-' + tag + '-import.tar'
        local_tag = 'teststudio/' + tag + ':controllers-local'
        subprocess.run(['docker', 'tag', source, local_tag], check=True)
        try:
            subprocess.run(['docker', 'save', '--output', str(archive), local_tag], check=True)
            os.chmod(archive, 0o600)
            subprocess.run(['docker', '--host', 'unix:///Users/johnbauer/.colima/teststudio-k3s/docker.sock', 'cp', str(archive), 'k3d-teststudio-local-server-0:' + node_archive], check=True)
            cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'import', '--platform', 'linux/arm64', node_archive)
            rows = [row.split() for row in cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'list').splitlines() if row.startswith('docker.io/' + local_tag + ' ')]
            assert len(rows) == 1 and rows[0][2].startswith('sha256:')
            reference = destination + '@' + rows[0][2]
            cluster.execute('ctr', '--namespace', 'k8s.io', 'images', 'tag', '--force', 'docker.io/' + local_tag, reference)
            state[tag] = {'source': source, 'reference': reference}
            checkpoint()
            return reference
        finally:
            cluster.execute('rm', '-f', node_archive)
            archive.unlink(missing_ok=True)

    checkpoint()
    service = json.loads((root / 'output/local-k3s/service-controller-build.json').read_text())
    nfs = json.loads((root / 'output/local-k3s/nfs-image.json').read_text())
    installed = json.loads(cluster.execute(*kube, 'get', 'deployments', '-n', state['namespace'], '-o', 'json'))['items']
    controller = next((item for item in installed if item['metadata']['name'] == 'teststudio-controller'), None)
    core = next((item for item in installed if item['metadata']['name'] == 'teststudio'), None)
    expected = state.get('service-controller', {})
    if controller and core and expected.get('source') == service['image']:
        current = controller['spec']['template']['spec']['containers'][0]['image']
        api_args = core['spec']['template']['spec']['containers'][0]['args']
        if current == expected.get('reference') and '--execution-worker=false' in api_args:
            required = {'teststudio', 'teststudio-controller', 'teststudio-shared-evidence', 'teststudio-prepare'}
            present = {item['metadata']['name'] for item in installed}
            ready = required <= present and all(item['status'].get('readyReplicas', 0) == item['spec']['replicas'] for item in installed if item['metadata']['name'] in required)
            if not ready or controller['spec']['replicas'] != 2:
                raise SystemExit('The existing controller deployment is pending. Observe that deployment; setup will not replace it.')
            state.update(deployed=True, valuesFile='.data/local-k3s/controller-values.json', evidenceClaim='teststudio-shared-evidence')
            checkpoint()
            print(json.dumps({'alreadyDeployed': True, 'controllerReplicas': 2, 'mutations': False}), flush=True)
            return
    image = json.loads(subprocess.check_output(['docker', 'image', 'inspect', nfs['reference']], text=True))[0]
    assert image['Architecture'] == 'arm64'
    service_ref = imported('service-controller', service['image'], 'docker.io/teststudio/service')
    nfs_ref = imported('local-nfs', nfs['reference'], 'docker.io/teststudio/local-nfs')
    nodes = json.loads(cluster.execute(*kube, 'get', 'nodes', '-o', 'json'))['items']
    assert len(nodes) == 1 and nodes[0]['metadata']['name'] == 'k3d-teststudio-local-server-0'
    node_ip = next(row['address'] for row in nodes[0]['status']['addresses'] if row['type'] == 'InternalIP')
    pod_network = nodes[0]['spec']['podCIDR']
    configuration = (root / 'tests/kubernetes/shared-evidence/ganesha.conf').read_text().replace('NODE_ADDRESS', node_ip).replace('POD_NETWORK', pod_network)
    labels = {'app': 'teststudio-shared-evidence'}
    meta = lambda name: {'name': name, 'namespace': state['namespace']}
    apply([
        {'apiVersion': 'v1', 'kind': 'ConfigMap', 'metadata': meta('teststudio-shared-evidence'), 'data': {'ganesha.conf': configuration}},
        {'apiVersion': 'v1', 'kind': 'Service', 'metadata': meta('teststudio-shared-evidence'), 'spec': {'selector': labels, 'ports': [{'name': 'nfs', 'port': 2049, 'targetPort': 2049}]}},
        {'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': meta('teststudio-shared-evidence'), 'spec': {'replicas': 1, 'strategy': {'type': 'Recreate'}, 'selector': {'matchLabels': labels}, 'template': {'metadata': {'labels': labels, 'annotations': {'checksum/config': hashlib.sha256(configuration.encode()).hexdigest()}}, 'spec': {'automountServiceAccountToken': False, 'nodeSelector': {'kubernetes.io/hostname': nodes[0]['metadata']['name']}, 'containers': [{'name': 'nfs', 'image': nfs_ref, 'imagePullPolicy': 'IfNotPresent', 'command': ['/bin/sh', '-c'], 'args': ['/usr/sbin/rpcbind -w && exec /usr/local/bin/ganesha.nfsd -F -f /etc/teststudio-nfs/ganesha.conf -L /dev/stdout -p /run/ganesha.pid'], 'securityContext': {'readOnlyRootFilesystem': True, 'capabilities': {'add': ['DAC_READ_SEARCH', 'SYS_RESOURCE']}}, 'resources': {'requests': {'cpu': '100m', 'memory': '128Mi'}, 'limits': {'cpu': '1', 'memory': '512Mi'}}, 'ports': [{'name': 'nfs', 'containerPort': 2049}], 'readinessProbe': {'tcpSocket': {'port': 'nfs'}, 'periodSeconds': 2}, 'volumeMounts': [{'name': 'evidence', 'mountPath': '/export'}, {'name': 'config', 'mountPath': '/etc/teststudio-nfs', 'readOnly': True}, {'name': 'state', 'mountPath': '/var/lib/nfs'}, {'name': 'run', 'mountPath': '/run'}, {'name': 'temporary', 'mountPath': '/tmp'}]}], 'volumes': [{'name': 'evidence', 'persistentVolumeClaim': {'claimName': state['previousClaim']}}, {'name': 'config', 'configMap': {'name': 'teststudio-shared-evidence'}}, {'name': 'state', 'emptyDir': {}}, {'name': 'run', 'emptyDir': {}}, {'name': 'temporary', 'emptyDir': {}}]}}}},
        {'apiVersion': 'networking.k8s.io/v1', 'kind': 'NetworkPolicy', 'metadata': meta('teststudio-shared-evidence'), 'spec': {'podSelector': {'matchLabels': labels}, 'policyTypes': ['Ingress', 'Egress'], 'ingress': [{'from': [{'ipBlock': {'cidr': node_ip + '/32'}}, {'ipBlock': {'cidr': pod_network}}], 'ports': [{'protocol': 'TCP', 'port': 2049}]}], 'egress': []}},
    ])
    print('Waiting for the owned NFS server.', flush=True)
    cluster.execute(*kube, 'rollout', 'status', 'deployment/teststudio-shared-evidence', '-n', state['namespace'], '--timeout=15s')
    server = json.loads(cluster.execute(*kube, 'get', 'service', 'teststudio-shared-evidence', '-n', state['namespace'], '-o', 'json'))['spec']['clusterIP']
    apply([
        {'apiVersion': 'v1', 'kind': 'PersistentVolume', 'metadata': {'name': 'teststudio-local-shared-evidence'}, 'spec': {'capacity': {'storage': '2Gi'}, 'accessModes': ['ReadWriteMany'], 'persistentVolumeReclaimPolicy': 'Retain', 'storageClassName': '', 'mountOptions': ['vers=4.1', 'proto=tcp', 'hard', 'timeo=30', 'retrans=2'], 'nfs': {'server': server, 'path': '/evidence'}}},
        {'apiVersion': 'v1', 'kind': 'PersistentVolumeClaim', 'metadata': meta('teststudio-shared-evidence'), 'spec': {'accessModes': ['ReadWriteMany'], 'storageClassName': '', 'volumeName': 'teststudio-local-shared-evidence', 'resources': {'requests': {'storage': '2Gi'}}}},
    ])
    policy = json.loads(cluster.execute(*kube, 'get', 'networkpolicy', 'registry', '-n', 'teststudio-registry', '-o', 'json'))
    peer = {'namespaceSelector': {'matchLabels': {'kubernetes.io/metadata.name': state['namespace']}}, 'podSelector': {'matchLabels': {'app.kubernetes.io/component': 'controller', 'app.kubernetes.io/instance': 'teststudio'}}}
    if peer not in policy['spec']['ingress'][0]['from']:
        policy['spec']['ingress'][0]['from'].append(peer)
        for field in ['managedFields', 'resourceVersion', 'uid', 'creationTimestamp', 'generation']:
            policy['metadata'].pop(field, None)
        apply([policy])
    original = installation / 'preparation-values.json'
    rollback = installation / 'before-controller-values.json'
    if not rollback.exists():
        rollback.write_bytes(original.read_bytes())
        os.chmod(rollback, 0o600)
    values = json.loads(original.read_text())
    values['image'].update(repository='docker.io/teststudio/service', digest=service_ref.split('@', 1)[1])
    values['controller'] = {'enabled': True, 'replicas': 2}
    values['persistence'].update(existingClaim='teststudio-shared-evidence', accessMode='ReadWriteMany')
    selected = installation / 'controller-values.json'
    selected.write_text(json.dumps(values, indent=2) + '\n')
    os.chmod(selected, 0o600)
    assert not cluster.pods('teststudio-workers'), 'Drain existing workers before changing the installation'
    if not state.get('deployed'):
        cluster.execute(*kube, 'scale', 'deployment/teststudio', '-n', state['namespace'], '--replicas=0')
        until = time.monotonic() + 100
        while cluster.pods(state['namespace'], '-l', 'app.kubernetes.io/component=api'):
            assert time.monotonic() < until, 'The existing API is still draining; inspect it without another deployment'
            time.sleep(1)
        state['apiDrained'] = True
        checkpoint()
    env = {key: value for key, value in os.environ.items() if not key.startswith(('HELM_', 'KUBE'))}
    env['HELM_DRIVER'] = 'secret'
    with (output / 'helm-upgrade.log').open('w') as log:
        subprocess.run([str(root / '.data/kubernetes-tools/helm'), 'upgrade', 'teststudio', str(root / 'deploy/kubernetes'), '--namespace', state['namespace'], '--kubeconfig', str(installation / 'kubeconfig.yaml'), '--kube-context', 'k3d-teststudio-local', '--values', str(selected), '--wait', '--timeout', '120s'], env=env, check=True, stdout=log, stderr=subprocess.STDOUT)
    state.update(deployed=True, valuesFile=str(selected.relative_to(root)), nfsServer=server, evidenceClaim='teststudio-shared-evidence')
    checkpoint()
    print(json.dumps({'deployed': True, 'controllerReplicas': 2, 'valuesFile': state['valuesFile'], 'preservedBackingClaim': state['previousClaim']}), flush=True)


if __name__ == '__main__':
    main()
