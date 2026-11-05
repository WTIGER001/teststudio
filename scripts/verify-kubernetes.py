import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import tempfile

import yaml
from lib.kubernetes_preparation import check_preparation


def main():
    parser = argparse.ArgumentParser(description='Render and validate Kubernetes packaging locally without reading a kubeconfig or contacting a cluster.')
    parser.add_argument('--helm', default='helm')
    parser.add_argument('--kubeconform', default='kubeconform')
    parser.add_argument('--output', type=Path, required=True)
    selection = parser.add_mutually_exclusive_group()
    selection.add_argument('--workers-only', action='store_true')
    selection.add_argument('--preparation-only', action='store_true')
    selection.add_argument('--routing-only', action='store_true')
    parser.add_argument('--pod', type=Path)
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
                if item['kind'] not in ['Namespace', 'ClusterRole', 'ClusterRoleBinding']:
                    assert item['metadata']['namespace'] in ['teststudio-fixture', values.get('workers', {}).get('namespace')]
            enabled = values.get('workers', {}).get('enabled', False)
            assert 'Secret' not in by_kind
            if not enabled:
                assert not any(kind in by_kind for kind in ['Role', 'RoleBinding', 'ClusterRole', 'ClusterRoleBinding'])
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
                    assert claim == values['persistence']['existingClaim'] and not any(item['metadata']['name'] == claim for item in by_kind.get('PersistentVolumeClaim', []))
                else:
                    assert by_kind['PersistentVolumeClaim'][0]['metadata']['name'] == claim
                    assert by_kind['PersistentVolumeClaim'][0]['metadata']['annotations']['helm.sh/resource-policy'] == 'keep'
            else:
                assert store == values['artifactStore'] and not any(item['metadata']['name'] == 'fixture-artifacts' for item in by_kind.get('PersistentVolumeClaim', []))
                assert 'emptyDir' in next(item for item in pod['volumes'] if item['name'] == 'data')
            services = {item['metadata']['name']: item for item in by_kind['Service']}
            ingress = next(item for item in by_kind['Ingress'] if item['metadata']['name'] == 'fixture')
            assert len(ingress['spec']['rules']) == 2
            for rule, origin, port in zip(ingress['spec']['rules'], [values['origins']['ui'], values['origins']['artifacts']], ['api', 'artifacts']):
                assert rule['host'] == origin.removeprefix('https://')
                backend = rule['http']['paths'][0]['backend']['service']
                service = services[backend['name']]
                assert backend['port']['name'] == port and service['spec']['ports'][0]['targetPort'] == port
                assert service['spec']['selector'] == deployment['spec']['selector']['matchLabels']
            policy = next(item for item in by_kind['NetworkPolicy'] if item['metadata']['name'] == 'fixture')['spec']
            assert policy['policyTypes'] == ['Ingress'] and policy['podSelector']['matchLabels'] == deployment['spec']['selector']['matchLabels']
            peer = policy['ingress'][0]['from'][0]
            assert peer['namespaceSelector']['matchLabels']['kubernetes.io/metadata.name'] == values['networkPolicy']['ingressNamespace']
            assert peer['podSelector']['matchLabels'] == values['networkPolicy']['ingressPodLabels']
            replicas = values.get('replicas', 1)
            assert deployment['spec']['replicas'] == replicas
            assert any(item['metadata']['name'] == 'fixture' for item in by_kind.get('PodDisruptionBudget', [])) == (replicas > 1)
            if driver == 'filesystem' and values.get('persistence', {}).get('accessMode', 'ReadWriteOnce') == 'ReadWriteOnce':
                assert deployment['spec']['strategy']['type'] == 'Recreate'
            else:
                assert deployment['spec']['strategy']['type'] == 'RollingUpdate'
            if enabled:
                worker = values['workers']
                settings = json.loads(config['kubernetes-workers.json'])
                assert '--worker-addr=0.0.0.0:8042' in container['args']
                assert '--kubernetes-workers-config=/etc/teststudio/kubernetes-workers.json' in container['args']
                assert settings['worker']['namespace'] == worker['namespace']
                assert settings['worker']['exchangeOrigin'] == worker['origin']
                assert settings['worker']['architecture'] == values['image']['architecture']
                assert settings['worker']['tokenFile'] == '/var/run/teststudio-controller/token'
                assert settings['worker']['exchangePort'] == worker['privateIngress'].get('port', 443)
                assert set(item['name'] for item in container['env']) >= set(worker['registry']['credentials'])
                identity = next(item for item in pod['volumes'] if item['name'] == 'controller-identity')['projected']
                assert identity['sources'][0]['serviceAccountToken']['expirationSeconds'] == 3600
                namespace = by_kind['Namespace'][0]
                assert namespace['metadata']['name'] == worker['namespace']
                assert namespace['metadata']['labels']['pod-security.kubernetes.io/enforce'] == 'restricted'
                assert namespace['metadata']['annotations']['helm.sh/resource-policy'] == 'keep'
                account = next(item for item in by_kind['ServiceAccount'] if item['metadata']['namespace'] == worker['namespace'])
                assert account['automountServiceAccountToken'] is False and not account['metadata'].get('annotations', {})
                bindings = by_kind['RoleBinding'] + by_kind['ClusterRoleBinding']
                subjects = [{'kind': 'ServiceAccount', 'name': 'fixture', 'namespace': 'teststudio-fixture'}]
                if values.get('preparation', {}).get('enabled'):
                    subjects.append({'kind': 'ServiceAccount', 'name': 'fixture-prepare', 'namespace': 'teststudio-fixture'})
                assert all(item['subjects'] == subjects for item in bindings)
                role = by_kind['Role'][0]
                assert role['metadata']['namespace'] == worker['namespace']
                assert {resource for rule in role['rules'] for resource in rule['resources']} == {'pods', 'serviceaccounts', 'networkpolicies'}
                cluster_role = by_kind['ClusterRole'][0]
                assert all(rule['verbs'] == ['get'] and rule['resourceNames'] for rule in cluster_role['rules'])
                egress = next(item for item in by_kind['NetworkPolicy'] if item['metadata']['namespace'] == worker['namespace'])
                (output / (name + '-settings.json')).write_text(json.dumps(settings, indent=2) + '\n')
                (output / (name + '-policy.json')).write_text(json.dumps(egress['spec'], indent=2) + '\n')
                assert egress['metadata']['annotations']['helm.sh/resource-policy'] == 'keep'
                assert egress['spec']['policyTypes'] == ['Ingress', 'Egress'] and egress['spec']['ingress'] == []
                assert egress['spec']['podSelector']['matchLabels'] == {'teststudio.io/worker': 'isolated-v1'}
                assert len(egress['spec']['egress']) == 2
                exchange = egress['spec']['egress'][0]
                assert exchange['ports'] == [{'protocol': 'TCP', 'port': worker['privateIngress'].get('port', 443)}]
                assert exchange['to'][0]['podSelector']['matchLabels'] == worker['privateIngress']['podLabels']
                private_routes = [item for item in by_kind['Ingress'] if item['metadata']['name'] == 'fixture-worker']
                if worker['privateIngress'].get('enabled', True):
                    assert len(private_routes) == 1
                    assert private_routes[0]['spec']['ingressClassName'] == worker['privateIngress']['className']
                    assert private_routes[0]['spec']['rules'][0]['host'] == worker['origin'].removeprefix('https://')
                else:
                    assert not private_routes
                assert policy['ingress'][1]['ports'] == [{'protocol': 'TCP', 'port': 8042}]
            else:
                assert 'kubernetes-workers.json' not in config
                assert not any(item['name'] == 'controller-identity' for item in pod['volumes'])
                assert not any(item.startswith('--kubernetes-workers') for item in container['args'])
            preparation_settings = check_preparation(objects, values)
            if preparation_settings is not None:
                (output / (name + '-preparation-settings.json')).write_text(json.dumps(preparation_settings, indent=2) + '\n')
            checked = execute([arguments.kubeconform, '-strict', '-summary', '-exit-on-error', '-kubernetes-version', source['kubernetesVersion'], '-schema-location', source['schemaLocation'], '-cache', str(cache), str(manifest)])
            (output / (name + '-schema.log')).write_text(checked.stdout)
            report['validProfiles'].append({'name': name, 'resources': len(objects), 'schemaValidated': True})
            return deployment['spec']['template']['metadata']['annotations']['checksum/config']

        values_file = temporary / 'lint.yaml'
        values_file.write_text(yaml.safe_dump(base, sort_keys=False))
        lint = execute([arguments.helm, 'lint', str(chart), '--strict', '--values', str(values_file)])
        (output / 'lint.log').write_text(lint.stdout)
        original_checksum = render('filesystem-single', base)
        if arguments.routing_only:
            enabled = copy.deepcopy(base)
            enabled.update(yaml.safe_load((chart / 'workers.example.yaml').read_text()))
            render('managed-private-routing', enabled)
            external = copy.deepcopy(enabled)
            external['workers']['privateIngress'].update(enabled=False, className='', tlsSecret='')
            render('external-private-routing', external)
            for name, selected, change in [
                ('managed-missing-class', enabled, {'className': ''}),
                ('managed-missing-tls', enabled, {'tlsSecret': ''}),
                ('external-missing-peer', external, {'namespace': ''}),
                ('external-empty-peer-labels', external, {'podLabels': {}}),
            ]:
                invalid = copy.deepcopy(selected)
                invalid['workers']['privateIngress'].update(change)
                render(name, invalid, expected=False)
        elif arguments.preparation_only:
            enabled = copy.deepcopy(base)
            enabled.update(yaml.safe_load((chart / 'workers.example.yaml').read_text()))
            render('workers-without-preparation', enabled)
            defaults = yaml.safe_load((chart / 'values.yaml').read_text())['preparation']
            selected = yaml.safe_load((chart / 'preparation.example.yaml').read_text())['preparation']
            defaults.update({key: value for key, value in selected.items() if key != 'registry'})
            defaults['registry'].update(selected['registry'])
            enabled['preparation'] = defaults
            render('preparation-single', enabled)
            go_only = copy.deepcopy(enabled)
            go_only['preparation']['playwrightRuntime'] = ''
            render('preparation-go-only', go_only)
            drain = copy.deepcopy(enabled)
            drain['preparation'].update(acceptNewWork=False, goRuntime='', playwrightRuntime='')
            drain['preparation']['persistence']['existingClaim'] = 'existing-private-journals'
            render('preparation-cleanup-only', drain)
            shared = copy.deepcopy(enabled)
            shared['image']['architecture'] = 'arm64'
            shared['workers']['registry']['caSecret'] = 'registry-trust'
            shared['preparation']['replicas'] = 2
            shared['preparation']['persistence']['accessMode'] = 'ReadWriteMany'
            shared['preparation']['registry'] = {'usernameEnvironment': '', 'passwordEnvironment': '', 'bearerEnvironment': 'TESTSTUDIO_REGISTRY_WRITER_BEARER', 'credentials': {'TESTSTUDIO_REGISTRY_WRITER_BEARER': {'existingSecret': 'writer-bearer', 'key': 'token'}}}
            render('preparation-shared-bearer', shared)
            invalid = {}
            value = copy.deepcopy(enabled); value['workers']['enabled'] = False; invalid['preparation-without-workers'] = value
            value = copy.deepcopy(enabled); value['preparation'].update(goRuntime='', playwrightRuntime=''); invalid['preparation-without-runtime'] = value
            value = copy.deepcopy(enabled); value['preparation']['goRuntime'] = 'registry.example.test/teststudio/source:latest'; invalid['preparation-mutable-runtime'] = value
            value = copy.deepcopy(enabled); value['preparation']['goRuntime'] = 'foreign.example.test/teststudio/source@sha256:' + 'a' * 64; invalid['preparation-foreign-runtime'] = value
            value = copy.deepcopy(enabled); value['preparation']['replicas'] = 2; invalid['preparation-shared-rwo'] = value
            value = copy.deepcopy(enabled); value['preparation']['persistence']['existingClaim'] = 'fixture-artifacts'; invalid['preparation-shared-artifact-claim'] = value
            value = copy.deepcopy(enabled); value['persistence'] = {'existingClaim': 'fixture-prepare-journals'}; invalid['preparation-default-journal-collision'] = value
            value = copy.deepcopy(enabled); value['preparation']['registry']['credentials']['TESTSTUDIO_REGISTRY_WRITER_PASSWORD']['existingSecret'] = 'teststudio-registry-reader'; invalid['preparation-reader-writer-secret'] = value
            value = copy.deepcopy(enabled); value['preparation']['registry']['credentials'] = {}; invalid['preparation-unmapped-writer'] = value
            value = copy.deepcopy(enabled); value['preparation']['registry']['bearerEnvironment'] = 'TESTSTUDIO_REGISTRY_WRITER_BEARER'; invalid['preparation-mixed-auth'] = value
            value = copy.deepcopy(enabled); value['preparation']['registry']['password'] = 'inline-private-canary'; invalid['preparation-inline-password'] = value
            value = copy.deepcopy(enabled); value['preparation']['registry']['credentials']['TESTSTUDIO_REGISTRY_UNUSED'] = {'existingSecret': 'unused-secret', 'key': 'password'}; invalid['preparation-unused-secret'] = value
            for name, values in invalid.items():
                render(name, values, expected=False)
        elif arguments.workers_only:
            worker_values = yaml.safe_load((chart / 'workers.example.yaml').read_text())
            enabled = copy.deepcopy(base)
            enabled.update(worker_values)
            render('private-workers', enabled)
            invalid = {}
            value = copy.deepcopy(enabled); value['workers']['namespace'] = 'teststudio-fixture'; invalid['shared-worker-namespace'] = value
            value = copy.deepcopy(enabled); value['workers']['origin'] = value['origins']['ui']; invalid['public-worker-origin'] = value
            value = copy.deepcopy(enabled); value['workers']['privateIngress']['className'] = value['ingress']['className']; invalid['public-worker-ingress'] = value
            value = copy.deepcopy(enabled); value['workers']['registry']['credentials'] = {}; invalid['unmapped-registry-credential'] = value
            for name, values in invalid.items():
                render(name, values, expected=False)
        else:
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
    if arguments.pod:
        worker_pod = json.loads(arguments.pod.read_text())
        spec = worker_pod['spec']
        assert worker_pod['kind'] == 'Pod' and spec['restartPolicy'] == 'Never'
        assert spec['automountServiceAccountToken'] is False and spec['enableServiceLinks'] is False
        assert spec['securityContext']['runAsUser'] == spec['securityContext']['fsGroup'] == 1000
        assert spec['activeDeadlineSeconds'] > 0 and spec['runtimeClassName']
        runner = spec['containers'][0]
        assert len(spec['containers']) == 1 and runner['command'] == ['/runner/teststudio-job']
        assert {item['name'] for item in runner['env']} == {'TESTSTUDIO_JOB_EXECUTION', 'TESTSTUDIO_JOB_TOKEN', 'TESTSTUDIO_JOB_INSTANCE'}
        assert runner['securityContext']['allowPrivilegeEscalation'] is False
        assert runner['securityContext']['readOnlyRootFilesystem'] is True
        assert all('emptyDir' in volume for volume in spec['volumes'])
        checked = execute([arguments.kubeconform, '-strict', '-summary', '-exit-on-error', '-kubernetes-version', source['kubernetesVersion'], '-schema-location', source['schemaLocation'], '-cache', str(cache), str(arguments.pod.resolve())])
        (output / 'worker-pod-schema.log').write_text(checked.stdout)
        report['workerPodSchemaValidated'] = True
    report['passed'] = True
    report['tools'] = {'helm': execute([arguments.helm, 'version', '--short']).stdout.strip(), 'kubeconform': execute([arguments.kubeconform, '-v']).stdout.strip()}
    (output / 'summary.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps({'passed': True, 'validProfiles': len(report['validProfiles']), 'rejectedConfigurations': len(report['rejectedConfigurations']), 'clusterContact': False}))


if __name__ == '__main__':
    main()
