import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import tempfile

import yaml


def main():
    parser = argparse.ArgumentParser(description='Render and validate Kubernetes packaging locally without reading a kubeconfig or contacting a cluster.')
    parser.add_argument('--helm', default='helm')
    parser.add_argument('--kubeconform', default='kubeconform')
    parser.add_argument('--output', type=Path, required=True)
    arguments = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    chart = root / 'deploy/kubernetes'
    output = arguments.output.resolve()
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    cache = output / 'schemas'
    cache.mkdir(exist_ok=True)
    source = json.loads((chart / 'schema-source.json').read_text())
    environment = {key: value for key, value in os.environ.items() if not key.startswith(('HELM_KUBE', 'KUBERNETES_SERVICE'))}
    environment.update(KUBECONFIG=os.devnull)
    environment.pop('HELM_DRIVER', None)
    environment.pop('HELM_MEMORY_DRIVER_DATA', None)

    def execute(command, success=True):
        result = subprocess.run(command, capture_output=True, text=True, env=environment, timeout=45)
        if success and result.returncode:
            raise RuntimeError(result.stderr or result.stdout)
        return result

    base = yaml.safe_load((chart / 'values.example.yaml').read_text())
    report = {'clusterContact': False, 'schemaSource': source, 'validProfiles': [], 'rejectedConfigurations': [], 'passed': False}
    with tempfile.TemporaryDirectory(prefix='teststudio-kubernetes-values-') as directory:
        temporary = Path(directory)

        def render(name, values, expected=True):
            value_file = temporary / (name + '.yaml')
            value_file.write_text(yaml.safe_dump(values, sort_keys=False))
            command = [arguments.helm, 'template', 'fixture', str(chart), '--namespace', 'teststudio-fixture', '--kube-version', source['kubernetesVersion'], '--values', str(value_file)]
            result = execute(command, success=expected)
            if not expected:
                assert result.returncode != 0, name + ' was unexpectedly accepted'
                report['rejectedConfigurations'].append(name)
                return
            manifest = output / (name + '.yaml')
            manifest.write_text(result.stdout)
            objects = [item for item in yaml.safe_load_all(result.stdout) if item]
            by_kind = {}
            for item in objects:
                by_kind.setdefault(item['kind'], []).append(item)
                assert item['metadata']['namespace'] == 'teststudio-fixture'
            assert not any(kind in by_kind for kind in ['Secret', 'Role', 'RoleBinding', 'ClusterRole', 'ClusterRoleBinding'])
            deployment = by_kind['Deployment'][0]
            pod = deployment['spec']['template']['spec']
            assert pod['automountServiceAccountToken'] is False
            assert pod['securityContext']['runAsNonRoot']
            assert pod['securityContext']['runAsUser'] == 65532
            assert pod['nodeSelector']['kubernetes.io/arch'] == values['image']['architecture']
            container = pod['containers'][0]
            assert container['securityContext']['readOnlyRootFilesystem']
            assert container['securityContext']['allowPrivilegeEscalation'] is False
            assert container['securityContext']['capabilities']['drop'] == ['ALL']
            assert container['image'] == values['image']['repository'] + '@' + values['image']['digest']
            assert '--database-env=TESTSTUDIO_DATABASE' in container['args'] and '--require-postgres=true' in container['args']
            assert all('secretKeyRef' in item['valueFrom'] and 'value' not in item for item in container['env'])
            references = ','.join('env:' + name for name in sorted(values['oidc']['credentials']))
            assert '--oidc-secret-refs=' + references in container['args']
            assert container['startupProbe']['httpGet']['path'] == container['readinessProbe']['httpGet']['path'] == '/healthz'
            assert container['livenessProbe']['httpGet']['path'] == '/livez'
            config = by_kind['ConfigMap'][0]['data']
            providers = json.loads(config['providers.json'])
            store = json.loads(config['artifact-store.json'])
            assert providers == values['oidc']['providers']
            driver = values.get('artifactStore', {}).get('driver', 'filesystem')
            assert store['driver'] == driver
            if driver == 'filesystem':
                assert store == {'driver': 'filesystem', 'directory': '/data/artifacts'}
                claim = next(item for item in pod['volumes'] if item['name'] == 'data')['persistentVolumeClaim']['claimName']
                if values.get('persistence', {}).get('existingClaim'):
                    assert claim == values['persistence']['existingClaim'] and 'PersistentVolumeClaim' not in by_kind
                else:
                    assert by_kind['PersistentVolumeClaim'][0]['metadata']['name'] == claim
                    assert by_kind['PersistentVolumeClaim'][0]['metadata']['annotations']['helm.sh/resource-policy'] == 'keep'
            else:
                assert store == values['artifactStore'] and 'PersistentVolumeClaim' not in by_kind
                assert 'emptyDir' in next(item for item in pod['volumes'] if item['name'] == 'data')
            services = {item['metadata']['name']: item for item in by_kind['Service']}
            ingress = by_kind['Ingress'][0]
            assert len(ingress['spec']['rules']) == 2
            for rule, origin, port in zip(ingress['spec']['rules'], [values['origins']['ui'], values['origins']['artifacts']], ['api', 'artifacts']):
                assert rule['host'] == origin.removeprefix('https://')
                backend = rule['http']['paths'][0]['backend']['service']
                service = services[backend['name']]
                assert backend['port']['name'] == port and service['spec']['ports'][0]['targetPort'] == port
                assert service['spec']['selector'] == deployment['spec']['selector']['matchLabels']
            policy = by_kind['NetworkPolicy'][0]['spec']
            assert policy['policyTypes'] == ['Ingress'] and policy['podSelector']['matchLabels'] == deployment['spec']['selector']['matchLabels']
            peer = policy['ingress'][0]['from'][0]
            assert peer['namespaceSelector']['matchLabels']['kubernetes.io/metadata.name'] == values['networkPolicy']['ingressNamespace']
            assert peer['podSelector']['matchLabels'] == values['networkPolicy']['ingressPodLabels']
            replicas = values.get('replicas', 1)
            assert deployment['spec']['replicas'] == replicas
            assert ('PodDisruptionBudget' in by_kind) == (replicas > 1)
            if driver == 'filesystem' and values.get('persistence', {}).get('accessMode', 'ReadWriteOnce') == 'ReadWriteOnce':
                assert deployment['spec']['strategy']['type'] == 'Recreate'
            else:
                assert deployment['spec']['strategy']['type'] == 'RollingUpdate'
            checked = execute([arguments.kubeconform, '-strict', '-summary', '-exit-on-error', '-kubernetes-version', source['kubernetesVersion'], '-schema-location', source['schemaLocation'], '-cache', str(cache), str(manifest)])
            (output / (name + '-schema.log')).write_text(checked.stdout)
            report['validProfiles'].append({'name': name, 'resources': len(objects), 'schemaValidated': True})
            return deployment['spec']['template']['metadata']['annotations']['checksum/config']

        values_file = temporary / 'lint.yaml'
        values_file.write_text(yaml.safe_dump(base, sort_keys=False))
        lint = execute([arguments.helm, 'lint', str(chart), '--strict', '--values', str(values_file)])
        (output / 'lint.log').write_text(lint.stdout)
        original_checksum = render('filesystem-single', base)
        azure = copy.deepcopy(base)
        azure['replicas'] = 2
        azure['artifactStore'] = {'driver': 'azure', 'azure': {'serviceUrl': 'https://teststudiofixture.blob.core.windows.net', 'container': 'evidence', 'prefix': 'installation'}}
        azure['oidc']['providers'].append({'id': 'second-provider', 'name': 'Second sign-in', 'issuer': 'https://second-identity.example.test', 'clientId': 'second-studio', 'autoProvision': False})
        assert render('azure-multiple', azure) != original_checksum
        shared = copy.deepcopy(base)
        shared.update(replicas=2, persistence={'existingClaim': 'fixture-shared-evidence', 'accessMode': 'ReadWriteMany'})
        render('filesystem-shared', shared)
        s3 = copy.deepcopy(base)
        s3['artifactStore'] = {'driver': 's3', 's3': {'region': 'us-east-1', 'bucket': 'teststudio-fixture-evidence', 'prefix': 'installation'}}
        s3['image']['architecture'] = 'arm64'
        render('s3-config', s3)
        invalid = {}
        value = copy.deepcopy(base); value['image']['digest'] = ''; invalid['missing-image-digest'] = value
        value = copy.deepcopy(base); value['origins']['artifacts'] = value['origins']['ui']; invalid['same-evidence-origin'] = value
        value = copy.deepcopy(base); value['database']['value'] = 'private-inline-canary'; invalid['inline-database-secret'] = value
        value = copy.deepcopy(base); value['oidc']['credentials'] = {}; invalid['unmapped-identity-secret'] = value
        value = copy.deepcopy(base); value['replicas'] = 2; invalid['multiple-rwo-replicas'] = value
        value = copy.deepcopy(base); value['bootstrap']['issuer'] = 'https://foreign.example.test'; invalid['foreign-bootstrap-issuer'] = value
        value = copy.deepcopy(base); value['podLabels'] = {'app.kubernetes.io/instance': 'another-installation'}; invalid['selector-label-override'] = value
        value = copy.deepcopy(base); value['oidc']['providers'].append(copy.deepcopy(value['oidc']['providers'][0])); invalid['duplicate-provider-id'] = value
        value = copy.deepcopy(base); value['image']['repository'] += ':latest'; invalid['mutable-image-tag'] = value
        for name, values in invalid.items():
            render(name, values, expected=False)
    report['passed'] = True
    report['tools'] = {'helm': execute([arguments.helm, 'version', '--short']).stdout.strip(), 'kubeconform': execute([arguments.kubeconform, '-v']).stdout.strip()}
    (output / 'summary.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps({'passed': True, 'validProfiles': len(report['validProfiles']), 'rejectedConfigurations': len(report['rejectedConfigurations']), 'clusterContact': False}))


if __name__ == '__main__':
    main()
