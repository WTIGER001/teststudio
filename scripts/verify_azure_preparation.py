import json


def verify_preparation(template, root):
    resources = template['resources']
    apps = [resource for resource in resources if resource['type'] == 'Microsoft.App/containerApps']
    assert len(apps) == 1
    app = apps[0]
    assert template['parameters']['workerEnabled']['defaultValue'] is False
    assert app['condition'] == "[parameters('workerEnabled')]"
    assert app['properties']['managedEnvironmentId'].startswith('[extensionResourceId(')
    assert app['properties']['workloadProfileName'] == 'Consumption'
    configuration = app['properties']['configuration']
    assert 'ingress' not in configuration
    assert configuration['activeRevisionsMode'] == 'Single'
    identities = [resource for resource in resources if resource['type'] == 'Microsoft.ManagedIdentity/userAssignedIdentities']
    assert len(identities) == 4
    assert app['identity']['type'] == 'UserAssigned'
    lifecycles = configuration['identitySettings']
    assert len(lifecycles) == 4
    assert [item['lifecycle'] for item in lifecycles] == ['Main', 'Main', 'None', 'None']
    for item, suffix in zip(lifecycles, ['source-writer', 'source-reader', 'source-pull', 'source-startup']):
        assert suffix in item['identity']
    checks = ['One optional headless assembly app uses a selected existing environment and four distinct managed identities with runtime access only for reader and writer']
    process = app['properties']['template']
    assert process['terminationGracePeriodSeconds'] == 90
    assert process['scale'] == {'minReplicas': 1, 'maxReplicas': 1}
    assert len(process['containers']) == 1 and 'initContainers' not in process
    container = process['containers'][0]
    assert container['command'] == ['/app/teststudio-prepare']
    assert container['args'] == "[variables('args')]" and 'probes' not in container
    args = template['variables']['args']
    for flag in ['--assembly-only=true', '--require-postgres=true', '--execution-location=azure-container-apps', '--azure-assembly-env=TESTSTUDIO_AZURE_ASSEMBLY', '--database-env=TESTSTUDIO_DATABASE', '--build-directory=/private/source-journals']:
        assert flag in args
    assert '--registry-go-runtime=' in args and '--registry-playwright-runtime=' in args
    assert '--go-runtime=' not in args and '--playwright-runtime=' not in args and '--github-sources' not in args
    assert '@' in container['image'] and "parameters('imageDigest')" in container['image']
    assert template['parameters']['imageDigest']['minLength'] == template['parameters']['imageDigest']['maxLength'] == 71
    checks.append('Digest-addressed service image starts only teststudio-prepare in PostgreSQL assembly-only mode with bounded always-on scaling and no discovery or HTTP listener')
    assert container['env'] == [{'name': 'TESTSTUDIO_DATABASE', 'secretRef': 'database'}, {'name': 'TESTSTUDIO_AZURE_ASSEMBLY', 'secretRef': 'assembly-configuration'}]
    assert len(configuration['secrets']) == 2
    assert all('source-startup' in secret['identity'] and 'keyVaultUrl' in secret and 'value' not in secret for secret in configuration['secrets'])
    assert len(configuration['registries']) == 1 and 'source-pull' in configuration['registries'][0]['identity']
    checks.append('Database and private assembly configuration use startup-only Key Vault references while platform image pull selects its separate identity')
    assert container['volumeMounts'] == [{'volumeName': 'source-journals', 'mountPath': '/private/source-journals'}]
    assert process['volumes'] == [{'name': 'source-journals', 'storageType': 'AzureFile', 'storageName': "[parameters('journalStorageName')]", 'mountOptions': 'dir_mode=0700,file_mode=0600,uid=65532,gid=65532,cache=none'}]
    dockerfile = (root / 'deploy/service/Dockerfile').read_text()
    assert 'USER 65532:65532' in dockerfile and 'COPY teststudio teststudio-health teststudio-prepare /app/' in dockerfile
    checks.append('Existing persistent Azure Files journal storage mounts privately for the service image user instead of using an ephemeral volume')
    deployments = [resource for resource in resources if resource['type'] == 'Microsoft.Resources/deployments']
    assert len(deployments) == 4 and sum('Microsoft.Resources/deployments' in item for item in app['dependsOn']) == 4
    for suffix, principal in [('source-push', 'source-writer'), ('source-pull', 'source-pull'), ('source-read', 'source-reader')]:
        deployment = next(item for item in deployments if suffix in item['name'])
        assert principal in deployment['properties']['parameters']['principalId']['value']
    secret_deployment = next(item for item in deployments if 'bindings' in item['properties']['parameters'])
    bindings = secret_deployment['properties']['parameters']['bindings']['value']
    assert len(bindings) == 2 and all('source-startup' in binding['principalId'] for binding in bindings)
    grants = [resource for deployment in deployments for resource in deployment['properties']['template']['resources'] if resource['type'] == 'Microsoft.Authorization/roleAssignments']
    push = next(grant for grant in grants if '8311e382-0749-4cb8-b61a-304f252e45ec' in grant['properties']['roleDefinitionId'])
    pull = next(grant for grant in grants if '7f951dda-4ed3-4680-a7ca-43fe172d538d' in grant['properties']['roleDefinitionId'])
    assert 'Microsoft.ContainerRegistry/registries' in push['scope'] and push['scope'] == pull['scope']
    read = next(deployment for deployment in deployments if 'installationKey' in deployment['properties']['parameters'])
    assert read['subscriptionId'] == '[subscription().subscriptionId]'
    role = next(resource for resource in read['properties']['template']['resources'] if resource['type'] == 'Microsoft.Authorization/roleDefinitions')
    assert role['properties']['permissions'] == [{'actions': ['*/read', 'Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action'], 'notActions': [], 'dataActions': [], 'notDataActions': []}]
    checks.append('Registry push, registry pull, metadata observation, and named-secret grants remain separate without execution dispatch or deadline administration grants')
    document = json.dumps(template)
    assert 'listKeys' not in document and 'getSecret' not in document and 'docker.sock' not in document
    assert 'amazonaws' not in document and 'AWS_ACCESS_KEY' not in document
    assert set(template['outputs']) == {'assemblyBinding'}
    output = template['outputs']['assemblyBinding']['value']
    assert set(output['writer']) == {'identity', 'clientId', 'principalId', 'tenantId', 'assignment'}
    assert output['journalDirectory'] == '/private/source-journals' and 'readerClientId' in output
    host = output['host']
    assert host['image'] == container['image'] and host['environment'] == app['properties']['managedEnvironmentId']
    assert host['application'] == output['application']
    assert host['readerAssignment'] == output['readerAssignment'] and host['readerRole'] == output['readerRole']
    assert host['journalStorageName'] == "[parameters('journalStorageName')]" and host['journalAccount'] == "[parameters('journalAccount')]" and host['journalShare'] == "[parameters('journalShare')]"
    assert host['databaseSecretUrl'] == configuration['secrets'][0]['keyVaultUrl'] and host['configurationSecretUrl'] == configuration['secrets'][1]['keyVaultUrl']
    assert host['goRuntime'] == "[parameters('goRuntime')]" and host['playwrightRuntime'] == "[parameters('playwrightRuntime')]"
    checks.append('Bootstrap outputs expose only installation bindings and templates retrieve no secret values, storage keys, or AWS configuration')
    return checks
