import json


def check_controller(objects, values):
    selected = values.get('controller', {})
    deployments = [item for item in objects if item['kind'] == 'Deployment']
    api = next(item for item in deployments if item['metadata']['name'] == 'fixture')
    api_container = api['spec']['template']['spec']['containers'][0]
    enabled = selected.get('enabled', False)
    assert ('--execution-worker=false' in api_container['args']) == enabled
    resources = [item for item in objects if item['metadata']['name'].startswith('fixture-controller')]
    if not enabled:
        assert not resources
        return
    deployment = next(item for item in resources if item['kind'] == 'Deployment')
    pod = deployment['spec']['template']['spec']
    container = pod['containers'][0]
    assert deployment['spec']['replicas'] == selected['replicas']
    assert pod['serviceAccountName'] == 'fixture-controller' != api['spec']['template']['spec']['serviceAccountName']
    assert pod['automountServiceAccountToken'] is False and pod['securityContext']['runAsUser'] == 65532
    assert pod['nodeSelector'] == api['spec']['template']['spec']['nodeSelector']
    assert container['command'] == ['/app/teststudio'] and '--serve-role=controller' in container['args']
    assert set(container['args']) >= {'--addr=0.0.0.0:8043', '--execution-location=kubernetes', '--database-env=TESTSTUDIO_DATABASE', '--require-postgres=true', '--kubernetes-workers-config=/etc/teststudio/kubernetes-workers.json', '--artifact-store-config=/etc/teststudio/artifact-store.json', '--public-url=' + values['origins']['ui']}
    assert not any(arg.startswith(('--ui=', '--oidc-', '--bootstrap-', '--worker-addr=', '--source-', '--execution-worker=')) for arg in container['args'])
    assert container['image'] == api_container['image'] and container['securityContext'] == api_container['securityContext']
    assert container['resources'] == selected['resources']
    assert container['ports'] == [{'name': 'health', 'containerPort': 8043}]
    assert container['readinessProbe']['httpGet'] == {'path': '/healthz', 'port': 'health'}
    assert container['livenessProbe']['httpGet'] == {'path': '/livez', 'port': 'health'}
    env = {item['name']: item['valueFrom']['secretKeyRef'] for item in container['env']}
    assert set(env) == {'TESTSTUDIO_DATABASE', *values['workers']['registry']['credentials']}
    for name, reference in values['workers']['registry']['credentials'].items():
        assert env[name] == {'name': reference['existingSecret'], 'key': reference['key']}
    config = next(item for item in resources if item['kind'] == 'ConfigMap')
    assert set(config['data']) == {'artifact-store.json', 'kubernetes-workers.json'}
    core = next(item for item in objects if item['kind'] == 'ConfigMap' and item['metadata']['name'] == 'fixture-config')
    assert all(json.loads(config['data'][name]) == json.loads(core['data'][name]) for name in config['data'])
    volumes = {item['name']: item for item in pod['volumes']}
    if values.get('artifactStore', {}).get('driver', 'filesystem') == 'filesystem':
        assert volumes['data']['persistentVolumeClaim']['claimName'] == values['persistence']['existingClaim']
    else:
        assert 'emptyDir' in volumes['data']
    assert volumes['controller-identity']['projected']['sources'][0]['serviceAccountToken']['expirationSeconds'] == 3600
    account = next(item for item in resources if item['kind'] == 'ServiceAccount')
    assert account['metadata'].get('annotations', {}) == selected['serviceAccount']['annotations']
    policy = next(item for item in resources if item['kind'] == 'NetworkPolicy')
    assert policy['spec']['policyTypes'] == ['Ingress'] and policy['spec']['ingress'] == []
    assert policy['spec']['podSelector']['matchLabels'] == deployment['spec']['selector']['matchLabels']
    assert any(item['kind'] == 'PodDisruptionBudget' for item in resources) == (selected['replicas'] > 1)
    assert not any(item['kind'] in ['Ingress', 'Service'] for item in resources)
    for item in objects:
        if item['kind'] == 'Service':
            assert item['spec']['selector'] != deployment['spec']['selector']['matchLabels']
        elif item['kind'] == 'Ingress':
            for rule in item['spec'].get('rules', []):
                assert all(path['backend']['service']['name'] != 'fixture-controller' for path in rule['http']['paths'])
