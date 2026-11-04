import json


def check_preparation(objects, values):
    preparation = values.get('preparation', {})
    enabled = preparation.get('enabled', False)
    deployments = [item for item in objects if item['kind'] == 'Deployment']
    api = next(item for item in deployments if item['metadata']['name'] == 'fixture')
    api_pod = api['spec']['template']['spec']
    api_container = api_pod['containers'][0]
    expected = enabled and preparation.get('acceptNewWork', True)
    assert ('--source-bundles=true' in api_container['args']) == expected
    assert ('--prepared-sources=true' in api_container['args']) == expected
    upload_profiles = [runner for runner, key in [('go', 'goRuntime'), ('playwright', 'playwrightRuntime')] if preparation.get(key)]
    assert [arg for arg in api_container['args'] if arg.startswith('--source-bundle-runners=')] == (['--source-bundle-runners=' + ','.join(upload_profiles)] if expected else [])
    if not enabled:
        assert len(deployments) == 1
        assert not any(item['metadata']['name'].startswith('fixture-prepare') for item in objects)
        return
    deployment = next(item for item in deployments if item['metadata']['name'] == 'fixture-prepare')
    pod = deployment['spec']['template']['spec']
    container = pod['containers'][0]
    assert len(pod['containers']) == 1 and not pod.get('initContainers')
    assert pod['serviceAccountName'] == 'fixture-prepare' != api_pod['serviceAccountName']
    assert pod['automountServiceAccountToken'] is False
    assert pod['securityContext'] == api_pod['securityContext']
    assert pod['nodeSelector'] == api_pod['nodeSelector']
    assert container['image'] == api_container['image']
    assert container['command'] == ['/app/teststudio-prepare']
    assert container['securityContext'] == api_container['securityContext']
    assert container['resources'] == preparation['resources']
    assert not any(key in container for key in ['ports', 'startupProbe', 'readinessProbe', 'livenessProbe'])
    assert set(container['args']) >= {'--database-env=TESTSTUDIO_DATABASE', '--require-postgres=true', '--execution-location=kubernetes', '--kubernetes-workers-config=/etc/teststudio/kubernetes-workers.json', '--build-directory=/journals/private'}
    runtime_args = [item for item in container['args'] if item.startswith('--registry-')]
    wanted = ['--registry-' + runner + '-runtime=' + preparation[key] for runner, key in [('go', 'goRuntime'), ('playwright', 'playwrightRuntime')] if expected and preparation.get(key)]
    assert runtime_args == wanted
    env = {item['name']: item for item in container['env']}
    assert set(env) == {'TESTSTUDIO_DATABASE', *preparation['registry']['credentials']}
    assert env['TESTSTUDIO_DATABASE'] == next(item for item in api_container['env'] if item['name'] == 'TESTSTUDIO_DATABASE')
    for name, reference in preparation['registry']['credentials'].items():
        assert env[name] == {'name': name, 'valueFrom': {'secretKeyRef': {'name': reference['existingSecret'], 'key': reference['key']}}}
    reader_secrets = {item['valueFrom']['secretKeyRef']['name'] for item in api_container['env'] if item['name'] in values['workers']['registry']['credentials']}
    writer_secrets = {item['valueFrom']['secretKeyRef']['name'] for name, item in env.items() if name != 'TESTSTUDIO_DATABASE'}
    assert not reader_secrets & writer_secrets
    volumes = {item['name']: item for item in pod['volumes']}
    mounts = {item['name']: item for item in container['volumeMounts']}
    assert set(volumes) == set(mounts) == {'config', 'journals', 'temporary', 'controller-identity', *(['registry-ca'] if values['workers']['registry'].get('caSecret') else [])}
    assert mounts['journals']['mountPath'] == '/journals'
    assert 'journals' not in {item['name'] for item in api_pod['volumes']}
    assert volumes['controller-identity'] == next(item for item in api_pod['volumes'] if item['name'] == 'controller-identity')
    claim = preparation['persistence']['existingClaim'] or 'fixture-prepare-journals'
    assert volumes['journals']['persistentVolumeClaim']['claimName'] == claim
    claims = [item for item in objects if item['kind'] == 'PersistentVolumeClaim' and item['metadata']['name'] == claim]
    assert bool(claims) == (not preparation['persistence']['existingClaim'])
    if claims:
        assert claims[0]['metadata']['annotations']['helm.sh/resource-policy'] == 'keep'
        assert claims[0]['spec']['accessModes'] == [preparation['persistence']['accessMode']]
    assert deployment['spec']['replicas'] == preparation['replicas']
    assert deployment['spec']['strategy']['type'] == ('Recreate' if preparation['persistence']['accessMode'] == 'ReadWriteOnce' else 'RollingUpdate')
    assert any(item['kind'] == 'PodDisruptionBudget' and item['metadata']['name'] == 'fixture-prepare' for item in objects) == (preparation['replicas'] > 1)
    config = next(item for item in objects if item['kind'] == 'ConfigMap' and item['metadata']['name'] == 'fixture-prepare-config')
    assert set(config['data']) == {'kubernetes-workers.json'}
    settings = json.loads(config['data']['kubernetes-workers.json'])
    api_settings = json.loads(next(item for item in objects if item['kind'] == 'ConfigMap' and item['metadata']['name'] == 'fixture-config')['data']['kubernetes-workers.json'])
    assert settings['worker'] == api_settings['worker']
    for name in ['origin', 'repositoryPrefix', 'tokenURL', 'tokenService', 'blobRedirectOrigins', 'caFile']:
        assert settings['registry'].get(name) == api_settings['registry'].get(name)
    for name in ['usernameEnvironment', 'passwordEnvironment', 'bearerEnvironment']:
        assert settings['registry'][name] == preparation['registry'][name]
        assert api_settings['registry'][name] == values['workers']['registry'].get(name, '')
    policy = next(item for item in objects if item['kind'] == 'NetworkPolicy' and item['metadata']['name'] == 'fixture-prepare')
    assert policy['spec']['podSelector']['matchLabels'] == deployment['spec']['selector']['matchLabels']
    assert policy['spec']['policyTypes'] == ['Ingress'] and policy['spec']['ingress'] == []
    for item in objects:
        if item['kind'] in ['Service', 'Ingress']:
            assert 'fixture-prepare' not in json.dumps(item)
    return settings
