import argparse
import json
from pathlib import Path
import subprocess
import time
import uuid


def main():
    parser = argparse.ArgumentParser(description='Verify the installed local worker RuntimeClass, private TLS exchange, and actual network denial with one temporary Pod.')
    parser.add_argument('--image', required=True)
    parser.add_argument('--binary', type=Path, default=Path('.data/local-k3s/isolation-check'))
    parser.add_argument('--installation', type=Path, default=Path('.data/local-k3s'))
    parser.add_argument('--output', type=Path, default=Path('output/local-k3s/isolation'))
    args = parser.parse_args()
    ownership = json.loads((args.installation / 'ownership.json').read_text())
    assert ownership['cluster'] == 'teststudio-local' and ownership['profile'] == 'teststudio-k3s'
    assert args.image.startswith('docker.io/teststudio/local-isolation-check@sha256:')
    args.output.mkdir(parents=True, exist_ok=True)
    base = ['docker', '--host', ownership['dockerHost'], 'exec', '-i', 'k3d-teststudio-local-server-0', 'kubectl']

    def kubectl(*arguments, value=None, check=True):
        return subprocess.run([*base, *arguments], input=None if value is None else json.dumps(value).encode(), capture_output=True, check=check, timeout=35)

    namespace = 'teststudio-workers'
    name = 'teststudio-netcheck-' + uuid.uuid4().hex[:10]
    executable = '/tmp/' + name
    try:
        subprocess.run(['docker', '--host', ownership['dockerHost'], 'cp', str(args.binary), 'k3d-teststudio-local-server-0:' + executable], check=True, capture_output=True, timeout=20)
        baseline = subprocess.run(['docker', '--host', ownership['dockerHost'], 'exec', 'k3d-teststudio-local-server-0', executable, '--baseline'], check=True, capture_output=True, timeout=20)
        reachable = json.loads(baseline.stdout)
        assert reachable['passed'] and len(reachable['reachableLocalServices']) == 4
        (args.output / 'baseline.json').write_text(json.dumps(reachable, indent=2) + '\n')
    finally:
        subprocess.run(['docker', '--host', ownership['dockerHost'], 'exec', 'k3d-teststudio-local-server-0', 'rm', '-f', executable], check=True, capture_output=True, timeout=20)
    config = {'apiVersion': 'v1', 'kind': 'ConfigMap', 'metadata': {'name': name, 'namespace': namespace}, 'data': {'ca.crt': (args.installation / 'fixture-tls/ca.crt').read_text()}}
    pod = {'apiVersion': 'v1', 'kind': 'Pod', 'metadata': {'name': name, 'namespace': namespace, 'labels': {'teststudio.io/worker': 'isolated-v1', 'teststudio.io/local-verification': name}}, 'spec': {'runtimeClassName': 'teststudio-isolated', 'serviceAccountName': 'teststudio-runner', 'restartPolicy': 'Never', 'activeDeadlineSeconds': 45, 'terminationGracePeriodSeconds': 5, 'automountServiceAccountToken': False, 'enableServiceLinks': False, 'securityContext': {'runAsNonRoot': True, 'runAsUser': 1000, 'runAsGroup': 1000, 'seccompProfile': {'type': 'RuntimeDefault'}}, 'containers': [{'name': 'check', 'image': args.image, 'imagePullPolicy': 'IfNotPresent', 'securityContext': {'allowPrivilegeEscalation': False, 'readOnlyRootFilesystem': True, 'capabilities': {'drop': ['ALL']}}, 'resources': {'requests': {'cpu': '100m', 'memory': '64Mi'}, 'limits': {'cpu': '1', 'memory': '256Mi'}}, 'volumeMounts': [{'name': 'ca', 'mountPath': '/ca', 'readOnly': True}]}], 'volumes': [{'name': 'ca', 'configMap': {'name': name}}]}}
    try:
        kubectl('create', '-f', '-', value=config)
        kubectl('create', '-f', '-', value=pod)
        deadline = time.monotonic() + 50
        while True:
            observed = json.loads(kubectl('get', 'pod', name, '--namespace', namespace, '--output', 'json').stdout)
            if observed['status']['phase'] in ['Succeeded', 'Failed']:
                break
            assert time.monotonic() < deadline, 'The diagnostic remains live; inspect its Pod before another run'
            time.sleep(0.5)
        (args.output / 'pod.json').write_text(json.dumps(observed, indent=2) + '\n')
        logs = kubectl('logs', name, '--namespace', namespace).stdout.decode()
        (args.output / 'diagnostic.log').write_text(logs)
        assert observed['status']['phase'] == 'Succeeded', logs
        summary = json.loads(logs)
        assert summary['passed'] and len(summary['deniedLocalServices']) == 4
        summary['podUID'] = observed['metadata']['uid']
        summary['image'] = args.image
        (args.output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    finally:
        cleanup = {}
        for kind in ['pod', 'configmap']:
            deleted = kubectl('delete', kind, name, '--namespace', namespace, '--ignore-not-found=true', '--wait=true', '--timeout=20s', check=False)
            remaining = kubectl('get', kind, name, '--namespace', namespace, '--ignore-not-found=true', '--output', 'name', check=False)
            cleanup[kind] = deleted.returncode == 0 and remaining.returncode == 0 and not remaining.stdout.strip()
        (args.output / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
        assert all(cleanup.values()), 'An owned diagnostic resource remains; inspect cleanup.json'
    print(json.dumps(summary))


if __name__ == '__main__':
    main()
