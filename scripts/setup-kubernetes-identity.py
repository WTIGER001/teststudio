import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess

from lib.local_kubernetes_cluster import LocalCluster
from lib.local_kubernetes_images import import_image


def main():
    root = Path.cwd()
    installation = root / '.data/local-k3s'
    output = root / 'output/local-k3s/identity'
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    cluster = LocalCluster(installation)
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']
    reference = output / 'installation.json'
    state = json.loads(reference.read_text()) if reference.exists() else {}

    def checkpoint():
        reference.write_text(json.dumps(state, indent=2) + '\n')

    def apply(objects):
        subprocess.run(cluster.command[:-1] + ['-i', cluster.command[-1]] + kube + ['apply', '-f', '-'], input=json.dumps({'apiVersion': 'v1', 'kind': 'List', 'items': objects}), text=True, check=True, timeout=30)

    source = root / 'tests/kubernetes/fixture'
    identity_text = (source / 'secondary-identity.json').read_text()
    identity = json.loads(identity_text)
    build = installation / 'secondary-identity-build'
    build.mkdir(parents=True, exist_ok=True, mode=0o700)
    key = hashlib.sha256((source / 'main.go').read_bytes() + (source / 'Dockerfile').read_bytes() + subprocess.check_output(['go', 'version'])).hexdigest()
    if state.get('sourceKey') != key:
        environment = dict(os.environ, CGO_ENABLED='0', GOOS='linux', GOARCH='arm64', GOWORK='off', GOFLAGS='')
        subprocess.run(['go', 'build', '-trimpath', '-ldflags=-s -w', '-o', str(build / 'fixture'), './tests/kubernetes/fixture'], env=environment, check=True)
        with (output / 'image-build.log').open('w') as log:
            subprocess.run(['docker', 'build', '--platform', 'linux/arm64', '--network=none', '--pull=false', '-f', str(source / 'Dockerfile'), '-t', 'teststudio/secondary-identity:local-k3s', str(build)], stdout=log, stderr=subprocess.STDOUT, check=True)
        state['image'] = import_image(cluster, installation, 'secondary-identity', 'teststudio/secondary-identity:local-k3s', 'docker.io/teststudio/secondary-identity')
        state['sourceKey'] = key
        checkpoint()
    tls = installation / 'secondary-identity-tls'
    tls.mkdir(parents=True, exist_ok=True, mode=0o700)
    if not (tls / 'server.crt').exists():
        (tls / 'extensions').write_text('subjectAltName=DNS:identity-two.teststudio.test\nkeyUsage=digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\n')
        with (output / 'certificate-setup.log').open('w') as log:
            subprocess.run(['openssl', 'req', '-new', '-newkey', 'rsa:2048', '-nodes', '-keyout', str(tls / 'server.key'), '-out', str(tls / 'server.csr'), '-subj', '/CN=identity-two.teststudio.test'], check=True, stdout=log, stderr=subprocess.STDOUT)
            subprocess.run(['openssl', 'x509', '-req', '-in', str(tls / 'server.csr'), '-CA', str(installation / 'fixture-tls/ca.crt'), '-CAkey', str(installation / 'fixture-tls/ca.key'), '-set_serial', '0x' + os.urandom(16).hex(), '-out', str(tls / 'server.crt'), '-days', '30', '-sha256', '-extfile', str(tls / 'extensions')], check=True, stdout=log, stderr=subprocess.STDOUT)
        os.chmod(tls / 'server.key', 0o600)
    subprocess.run(['openssl', 'verify', '-CAfile', str(installation / 'fixture-tls/ca.crt'), str(tls / 'server.crt')], check=True)
    name = 'teststudio-secondary-identity'
    namespace = 'teststudio-local'
    meta = lambda value: {'name': value, 'namespace': namespace}
    labels = {'app.kubernetes.io/name': name}
    pod = {'automountServiceAccountToken': False, 'securityContext': {'runAsNonRoot': True, 'runAsUser': 65532, 'runAsGroup': 65532, 'seccompProfile': {'type': 'RuntimeDefault'}}, 'containers': [{'name': 'identity', 'image': state['image'], 'imagePullPolicy': 'IfNotPresent', 'args': ['--identity-config=/identity/config.json'], 'securityContext': {'readOnlyRootFilesystem': True, 'allowPrivilegeEscalation': False, 'capabilities': {'drop': ['ALL']}}, 'resources': {'requests': {'cpu': '50m', 'memory': '32Mi'}, 'limits': {'cpu': '500m', 'memory': '128Mi'}}, 'readinessProbe': {'httpGet': {'path': '/health', 'port': 8080}, 'periodSeconds': 2}, 'volumeMounts': [{'name': 'tls', 'mountPath': '/tls', 'readOnly': True}, {'name': 'identity', 'mountPath': '/identity', 'readOnly': True}]}], 'volumes': [{'name': 'tls', 'secret': {'secretName': name + '-tls'}}, {'name': 'identity', 'configMap': {'name': name}}]}
    apply([
        {'apiVersion': 'v1', 'kind': 'Secret', 'metadata': meta(name + '-tls'), 'type': 'kubernetes.io/tls', 'data': {'tls.crt': base64.b64encode((tls / 'server.crt').read_bytes()).decode(), 'tls.key': base64.b64encode((tls / 'server.key').read_bytes()).decode()}},
        {'apiVersion': 'v1', 'kind': 'ConfigMap', 'metadata': meta(name), 'data': {'config.json': identity_text}},
        {'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': meta(name), 'spec': {'replicas': 1, 'selector': {'matchLabels': labels}, 'template': {'metadata': {'labels': labels, 'annotations': {'checksum/config': hashlib.sha256(identity_text.encode()).hexdigest()}}, 'spec': pod}}},
        {'apiVersion': 'v1', 'kind': 'Service', 'metadata': meta(name), 'spec': {'selector': labels, 'ports': [{'name': 'https', 'port': 443, 'targetPort': 8443}, {'name': 'http', 'port': 80, 'targetPort': 8080}]}},
        {'apiVersion': 'networking.k8s.io/v1', 'kind': 'Ingress', 'metadata': meta(name), 'spec': {'ingressClassName': 'traefik', 'tls': [{'hosts': ['identity-two.teststudio.test'], 'secretName': name + '-tls'}], 'rules': [{'host': 'identity-two.teststudio.test', 'http': {'paths': [{'path': '/', 'pathType': 'Prefix', 'backend': {'service': {'name': name, 'port': {'name': 'http'}}}}]}}]}},
        {'apiVersion': 'networking.k8s.io/v1', 'kind': 'NetworkPolicy', 'metadata': meta(name), 'spec': {'podSelector': {'matchLabels': labels}, 'policyTypes': ['Ingress', 'Egress'], 'ingress': [{'from': [{'podSelector': {'matchLabels': {'app.kubernetes.io/name': 'teststudio', 'app.kubernetes.io/instance': 'teststudio', 'app.kubernetes.io/component': 'api'}}}], 'ports': [{'protocol': 'TCP', 'port': 8443}]}, {'from': [{'namespaceSelector': {'matchLabels': {'kubernetes.io/metadata.name': 'kube-system'}}, 'podSelector': {'matchLabels': {'app.kubernetes.io/name': 'traefik'}}}], 'ports': [{'protocol': 'TCP', 'port': 8080}]}], 'egress': []}},
    ])
    cluster.execute(*kube, 'rollout', 'status', 'deployment/' + name, '-n', namespace, '--timeout=15s')
    service = json.loads(cluster.execute(*kube, 'get', 'service', name, '-n', namespace, '-o', 'json'))
    address = service['spec']['clusterIP']
    entry = 'identity-two.teststudio.test:53 {\n errors\n cache 5\n hosts {\n  ' + address + ' identity-two.teststudio.test\n  fallthrough\n }\n}\n'
    dns = json.loads(cluster.execute(*kube, 'get', 'configmap', 'coredns-custom', '-n', 'kube-system', '-o', 'json'))
    if dns.get('data', {}).get('teststudio-secondary.server') != entry:
        cluster.execute(*kube, 'patch', 'configmap', 'coredns-custom', '-n', 'kube-system', '--type=merge', '-p', json.dumps({'data': {'teststudio-secondary.server': entry}}))
        state['dnsRestartArmed'] = True
        checkpoint()
        cluster.execute(*kube, 'rollout', 'restart', 'deployment/coredns', '-n', 'kube-system')
    cluster.execute(*kube, 'rollout', 'status', 'deployment/coredns', '-n', 'kube-system', '--timeout=15s')
    state.update(issuer=identity['issuer'], providerId='local-k3s-review', serviceIP=address, ready=True)
    checkpoint()
    print(json.dumps({'ready': True, 'issuer': identity['issuer'], 'providerId': state['providerId'], 'firstProviderUnchanged': True}), flush=True)


if __name__ == '__main__':
    main()
