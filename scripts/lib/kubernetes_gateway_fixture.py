import base64
import hashlib
import json
import os
from pathlib import Path
import secrets
import subprocess


def configure_fixture(root, cluster, images):
    installation = root / '.data/local-k3s'
    private = installation / 'gateway'
    private.mkdir(parents=True, exist_ok=True, mode=0o700)
    credential = private / 'reader'
    if not credential.exists():
        credential.write_text(secrets.token_hex(32))
    os.chmod(credential, 0o600)
    ca = installation / 'fixture-tls/ca.crt'

    def certificate(name, names):
        directory = private / name
        directory.mkdir(exist_ok=True, mode=0o700)
        if not (directory / 'tls.crt').exists():
            extension = directory / 'extensions'
            extension.write_text('subjectAltName=' + ','.join('DNS:' + value for value in names) + '\nkeyUsage=digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\n')
            with (directory / 'openssl.log').open('w') as log:
                subprocess.run(['openssl', 'req', '-new', '-newkey', 'rsa:2048', '-nodes', '-keyout', str(directory / 'tls.key'), '-out', str(directory / 'tls.csr'), '-subj', '/CN=' + names[0]], check=True, stdout=log, stderr=subprocess.STDOUT)
                os.chmod(directory / 'tls.key', 0o600)
                subprocess.run(['openssl', 'x509', '-req', '-in', str(directory / 'tls.csr'), '-CA', str(ca), '-CAkey', str(installation / 'fixture-tls/ca.key'), '-set_serial', '0x' + secrets.token_hex(16), '-out', str(directory / 'tls.crt'), '-days', '30', '-sha256', '-extfile', str(extension)], check=True, stdout=log, stderr=subprocess.STDOUT)
        subprocess.run(['openssl', 'verify', '-CAfile', str(ca), str(directory / 'tls.crt')], check=True)
        return {name: base64.b64encode((directory / name).read_bytes()).decode() for name in ['tls.crt', 'tls.key']}

    gateway_tls = certificate('gateway-tls', ['*.teststudio-workers.svc', '*.teststudio-workers.svc.cluster.local'])
    target_tls = certificate('target-tls', ['gateway-target.teststudio-local.svc', 'gateway-target.teststudio-local.svc.cluster.local'])
    metadata = lambda name, namespace='teststudio-local': {'name': name, 'namespace': namespace}
    secret = lambda name, namespace, data: {'apiVersion': 'v1', 'kind': 'Secret', 'metadata': metadata(name, namespace), 'data': data}
    origin = 'https://gateway-target.teststudio-local.svc:8443'
    bindings = [{'id': 'gateway-reader', 'name': 'Independent local gateway reader', 'projectId': 'k3s-network', 'purpose': 'api-credential', 'file': '/etc/teststudio-secrets/reader', 'network': 'private', 'allowedOrigins': [origin]}]
    labels = {'app.kubernetes.io/name': 'gateway-target'}
    pod = {
        'automountServiceAccountToken': False,
        'securityContext': {'runAsNonRoot': True, 'runAsUser': 1000, 'runAsGroup': 1000, 'fsGroup': 1000, 'seccompProfile': {'type': 'RuntimeDefault'}},
        'containers': [{
            'name': 'target', 'image': images['gateway-target']['reference'], 'imagePullPolicy': 'IfNotPresent',
            'ports': [{'name': 'http', 'containerPort': 8080}, {'name': 'https', 'containerPort': 8443}],
            'readinessProbe': {'httpGet': {'path': '/healthz', 'port': 'http'}, 'periodSeconds': 2},
            'securityContext': {'readOnlyRootFilesystem': True, 'allowPrivilegeEscalation': False, 'capabilities': {'drop': ['ALL']}},
            'resources': {'requests': {'cpu': '25m', 'memory': '16Mi'}, 'limits': {'cpu': '500m', 'memory': '64Mi'}},
            'volumeMounts': [{'name': 'tls', 'mountPath': '/tls', 'readOnly': True}, {'name': 'credentials', 'mountPath': '/credentials', 'readOnly': True}],
        }],
        'volumes': [{'name': 'tls', 'secret': {'secretName': 'gateway-target-tls', 'defaultMode': 0o440}}, {'name': 'credentials', 'secret': {'secretName': 'gateway-fixture-reader', 'defaultMode': 0o440}}],
    }
    objects = [
        secret('teststudio-gateway-tls', 'teststudio-workers', gateway_tls),
        secret('teststudio-gateway-exchange-ca', 'teststudio-workers', {'ca.crt': base64.b64encode(ca.read_bytes()).decode()}),
        secret('teststudio-gateway-ca', 'teststudio-local', {'ca.crt': base64.b64encode(ca.read_bytes()).decode()}),
        secret('gateway-target-tls', 'teststudio-local', target_tls),
        secret('gateway-fixture-reader', 'teststudio-local', {'reader': base64.b64encode(credential.read_bytes()).decode()}),
        secret('teststudio-gateway-credentials', 'teststudio-local', {'reader': base64.b64encode(credential.read_bytes()).decode(), 'bindings.json': base64.b64encode(json.dumps(bindings).encode()).decode()}),
        {'apiVersion': 'apps/v1', 'kind': 'Deployment', 'metadata': metadata('gateway-target'), 'spec': {'replicas': 1, 'selector': {'matchLabels': labels}, 'template': {'metadata': {'labels': labels, 'annotations': {'checksum/fixture': hashlib.sha256(json.dumps(target_tls).encode() + credential.read_bytes()).hexdigest()}}, 'spec': pod}}},
        {'apiVersion': 'v1', 'kind': 'Service', 'metadata': metadata('gateway-target'), 'spec': {'selector': labels, 'ports': [{'name': 'http', 'port': 8080, 'targetPort': 'http'}, {'name': 'https', 'port': 8443, 'targetPort': 'https'}]}},
        {'apiVersion': 'networking.k8s.io/v1', 'kind': 'NetworkPolicy', 'metadata': metadata('gateway-target'), 'spec': {'podSelector': {'matchLabels': labels}, 'policyTypes': ['Ingress', 'Egress'], 'ingress': [{'from': [{'namespaceSelector': {'matchLabels': {'kubernetes.io/metadata.name': 'teststudio-workers'}}, 'podSelector': {'matchLabels': {'teststudio.io/worker': 'gateway-v1'}}}], 'ports': [{'protocol': 'TCP', 'port': 8080}, {'protocol': 'TCP', 'port': 8443}]}], 'egress': []}},
    ]
    subprocess.run(cluster.command[:-1] + ['-i', cluster.command[-1], 'kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml', 'apply', '-f', '-'], input=json.dumps({'apiVersion': 'v1', 'kind': 'List', 'items': objects}), check=True, text=True, timeout=30)
    kube = ['kubectl', '--kubeconfig', '/etc/rancher/k3s/k3s.yaml']
    policy = json.loads(cluster.execute(*kube, 'get', 'networkpolicy', 'worker-exchange', '-n', 'teststudio-private-routing', '-o', 'json'))
    assert policy['spec']['podSelector'] == {'matchLabels': {'app.kubernetes.io/name': 'teststudio-private-ingress'}}
    assert policy['spec']['ingress'][0]['ports'] == [{'port': 8443, 'protocol': 'TCP'}]
    peer = {'namespaceSelector': {'matchLabels': {'kubernetes.io/metadata.name': 'teststudio-workers'}}, 'podSelector': {'matchLabels': {'teststudio.io/worker': 'gateway-v1'}}}
    if peer not in policy['spec']['ingress'][0]['from']:
        patch = [{'op': 'test', 'path': '/metadata/resourceVersion', 'value': policy['metadata']['resourceVersion']}, {'op': 'add', 'path': '/spec/ingress/0/from/-', 'value': peer}]
        cluster.execute(*kube, 'patch', 'networkpolicy', 'worker-exchange', '-n', 'teststudio-private-routing', '--type=json', '-p', json.dumps(patch))
    return {'browserOrigin': 'http://gateway-target.teststudio-local.svc:8080', 'apiOrigin': origin}
