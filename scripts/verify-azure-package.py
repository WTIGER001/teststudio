import argparse
import json
import os
import re
from pathlib import Path
import subprocess


ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description='Compile and check Azure packaging locally without Azure CLI, account access, or deployment.')
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument('--bicep', type=Path)
    source.add_argument('--compiled-template', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    template_path = output / 'service.json'
    if args.bicep:
        environment = {key: value for key, value in os.environ.items() if not key.startswith(('AZURE_', 'ARM_', 'AWS_'))}
        result = subprocess.run([str(args.bicep.resolve()), 'build', str(ROOT / 'deploy/azure/main.bicep'), '--no-restore', '--outfile', str(template_path)], capture_output=True, text=True, env=environment, timeout=60)
        (output / 'compile.log').write_text(result.stdout + result.stderr)
        assert result.returncode == 0 and 'Warning' not in result.stderr, 'Bicep compile or schema validation failed; inspect compile.log'
    elif args.compiled_template.resolve() != template_path:
        template_path.write_text(args.compiled_template.read_text())
    template = json.loads(template_path.read_text())
    resources = template['resources']
    checks = ['Bicep-generated local modules pass packaging checks without registry restoration or cloud calls']
    deployments = [resource for resource in resources if resource['type'] == 'Microsoft.Resources/deployments']
    apps = {deployment['properties']['parameters']['role']['value']: deployment for deployment in deployments if 'role' in deployment['properties']['parameters']}
    assert set(apps) == {'api', 'artifacts'}
    api_parameters = apps['api']['properties']['parameters']
    artifact_parameters = apps['artifacts']['properties']['parameters']
    assert api_parameters['uiOrigin'] == artifact_parameters['uiOrigin'] and api_parameters['artifactOrigin'] == artifact_parameters['artifactOrigin']
    assert template['outputs']['uiUrl']['value'] != template['outputs']['artifactUrl']['value']
    assert 'apiName' in template['outputs']['uiUrl']['value'] and 'artifactName' in template['outputs']['artifactUrl']['value']
    assert 'bootstrapIssuer' in api_parameters and 'oidcConfigurationSecretName' in api_parameters
    assert not any(name in artifact_parameters for name in ['bootstrapIssuer', 'bootstrapSubject', 'oidcConfigurationSecretName', 'oidcClients', 'localAccounts'])
    checks.append('Two apps share explicit installation origins while the artifact role receives no sign-in or bootstrap configuration')
    for role, deployment in apps.items():
        nested = deployment['properties']['template']
        app = next(resource for resource in nested['resources'] if resource['type'] == 'Microsoft.App/containerApps')
        configuration = app['properties']['configuration']
        assert configuration['ingress']['external'] and not configuration['ingress']['allowInsecure']
        assert configuration['ingress']['targetPort'] == "[variables('port')]"
        assert nested['variables']['port'] == "[if(variables('api'), 8040, 8041)]"
        assert configuration['activeRevisionsMode'] == 'Single'
        assert configuration['identitySettings'] == "[variables('identitySettings')]"
        identity_settings = nested['variables']['identitySettings']
        assert re.findall(r"createObject\('identity', parameters\('([^']+)'\), 'lifecycle', '([^']+)'\)", identity_settings) == [('pullIdentityId', 'None'), ('startupIdentityId', 'None'), ('runtimeIdentityId', 'Main'), ('observerIdentityId', 'Main')]
        assert "if(variables('observerEnabled')," in identity_settings
        assert nested['variables']['observerEnabled'] == "[and(variables('api'), not(empty(parameters('observerIdentityId'))))]"
        assert app['identity'] == {'type': 'UserAssigned', 'userAssignedIdentities': "[variables('identities')]"}
        assert "if(variables('observerEnabled')," in nested['variables']['identities']
        assert nested['parameters']['observerIdentityId']['defaultValue'] == ''
        container = app['properties']['template']['containers'][0]
        assert 'command' not in container and container['args'] == "[variables('args')]"
        settings = json.dumps(nested['variables'])
        assert '--serve-role={0}' in settings and '--require-postgres=true' in settings
        assert 'managedIdentityClientId' in settings and "parameters('runtimeClientId')" in settings
        assert 'secretRef' in settings and 'keyVaultUrl' in settings
        assert '--go-images' not in settings and '--playwright-images' not in settings
        probes = {probe['type']: probe for probe in container['probes']}
        assert set(probes) == {'Startup', 'Readiness', 'Liveness'}
        assert all(probe['httpGet']['port'] == "[variables('port')]" for probe in probes.values())
        assert probes['Readiness']['httpGet']['path'] == probes['Startup']['httpGet']['path'] == '/healthz'
        assert probes['Liveness']['httpGet']['path'] == '/livez'
        assert app['properties']['template']['scale'] == {'minReplicas': "[parameters('replicas')]", 'maxReplicas': "[parameters('replicas')]"}
        assert nested['parameters']['replicas']['minValue'] == 1
    checks.append('Each HTTPS ingress targets its own listener with independent readiness, liveness, and always-on replica settings')
    checks.append('Platform-only registry and startup identities are withheld from runtime while Blob access selects the exact runtime client')
    identities = [resource for resource in resources if resource['type'] == 'Microsoft.ManagedIdentity/userAssignedIdentities']
    assert len(identities) == 6
    assert len([identity for identity in identities if 'condition' not in identity]) == 5
    observer = next(identity for identity in identities if 'condition' in identity)
    assert observer['condition'] == "[parameters('observerIdentityEnabled')]"
    assert template['parameters']['observerIdentityEnabled']['defaultValue'] is False
    assert 'observerIdentityEnabled' in api_parameters['observerIdentityId']
    assert 'observerIdentityId' not in artifact_parameters
    observer_access = next(deployment for deployment in deployments if 'installationKey' in deployment['properties']['parameters'])
    assert observer_access['condition'] == observer['condition']
    assert observer_access['subscriptionId'] == '[subscription().subscriptionId]'
    observer_resources = observer_access['properties']['template']['resources']
    assert len(observer_resources) == 2
    observer_role = next(resource for resource in observer_resources if resource['type'] == 'Microsoft.Authorization/roleDefinitions')
    observer_grant = next(resource for resource in observer_resources if resource['type'] == 'Microsoft.Authorization/roleAssignments')
    assert observer_role['properties']['type'] == 'CustomRole'
    assert observer_role['properties']['assignableScopes'] == ['[subscription().id]']
    assert observer_role['properties']['permissions'] == [{'actions': ['*/read', 'Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action'], 'notActions': [], 'dataActions': [], 'notDataActions': []}]
    assert set(observer_grant['properties']) == {'principalId', 'principalType', 'roleDefinitionId'}
    assert observer_grant['properties']['principalType'] == 'ServicePrincipal'
    assert 'observerBinding' in template['outputs'] and 'observerIdentityEnabled' in template['outputs']['observerBinding']['value']
    checks.append('Optional observer identity and fixed subscription read role attach only to the API main container; disabled remains the default')
    observer_binding = template['outputs']['observerBinding']['value']
    runtime_fields = ['runtimeClientId', 'databaseSecretUrl', 'oidcConfigurationSecretUrl', 'oidcClientOneSecretUrl', 'oidcClientTwoSecretUrl', 'blobServiceUrl', 'evidenceContainer', 'uiOrigin', 'artifactOrigin', 'bootstrapIssuer', 'bootstrapSubject', 'localAccounts', 'replicas']
    assert "'runtime', createObject(" in observer_binding
    assert all("'" + field + "'," in observer_binding for field in runtime_fields)
    assert "'replicas', parameters('apiReplicas')" in observer_binding
    assert "'localAccounts', parameters('localAccounts')" in observer_binding
    assert "'runtimeClientId', reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', format('{0}-api-runtime', parameters('installationName'))), '2023-01-31').clientId" in observer_binding
    assert "'databaseSecretUrl', format('{0}secrets/{1}', reference(extensionResourceId(format('/subscriptions/{0}/resourceGroups/{1}', subscription().subscriptionId, parameters('keyVaultResourceGroup')), 'Microsoft.KeyVault/vaults', parameters('keyVaultName')), '2023-07-01').vaultUri, parameters('databaseSecretName'))" in observer_binding
    checks.append('Observer output carries the fixed API runtime, origins, identity client, replica limit, and Key Vault references without secret retrieval')
    assert api_parameters['runtimeIdentityId'] != artifact_parameters['runtimeIdentityId']
    assert api_parameters['startupIdentityId'] != artifact_parameters['startupIdentityId']
    storage = next(resource for resource in resources if resource['type'] == 'Microsoft.Storage/storageAccounts')
    assert not storage['properties']['allowSharedKeyAccess'] and not storage['properties']['allowBlobPublicAccess']
    assert storage['properties']['supportsHttpsTrafficOnly'] and storage['properties']['minimumTlsVersion'] == 'TLS1_2'
    container = next(resource for resource in resources if resource['type'] == 'Microsoft.Storage/storageAccounts/blobServices/containers')
    assert container['properties']['publicAccess'] == 'None'
    lock = next(resource for resource in resources if resource['type'] == 'Microsoft.Authorization/locks')
    assert lock['properties']['level'] == 'CanNotDelete' and 'Microsoft.Storage/storageAccounts' in lock['scope']
    blob_deployments = [deployment for deployment in deployments if 'roleId' in deployment['properties']['parameters']]
    assert len(blob_deployments) == 2
    assert {deployment['properties']['parameters']['roleId']['value'] for deployment in blob_deployments} == {'ba92f5b4-2d11-453d-a403-e96b0029c9fe', '2a2b9908-6ea1-4ae2-8e65-a410df84e7d1'}
    for deployment in blob_deployments:
        grant = deployment['properties']['template']['resources'][0]
        assert grant['type'] == 'Microsoft.Authorization/roleAssignments'
        assert 'Microsoft.Storage/storageAccounts/blobServices/containers' in grant['scope']
        assert "parameters('principalId')" in grant['name'] and "parameters('roleId')" in grant['name']
    assert not any(resource['type'] == 'Microsoft.Storage/storageAccounts/managementPolicies' for resource in resources)
    checks.append('Dedicated private Blob storage uses OAuth, scoped writer and reader roles, deletion protection, and no external expiry policy')
    secret_deployment = next(deployment for deployment in deployments if 'bindings' in deployment['properties']['parameters'])
    secret_template = secret_deployment['properties']['template']
    grants = secret_template['resources']
    assert len(grants) == 1 and grants[0]['type'] == 'Microsoft.Authorization/roleAssignments'
    assert 'Microsoft.KeyVault/vaults/secrets' in grants[0]['scope']
    assert '4633458b-17de-408a-b874-0445c86b69e6' in grants[0]['properties']['roleDefinitionId']
    document = json.dumps(template)
    assert 'listKeys' not in document and 'getSecret' not in document and 'docker.sock' not in document
    assert 'amazonaws' not in document and 'AWS_ACCESS_KEY' not in document
    assert not any('secret' in name.lower() for name in template['outputs'])
    checks.append('Secret grants remain scoped to named Key Vault secrets and templates retrieve no secret values, storage keys, or AWS settings')
    assert template['parameters']['imageDigest']['minLength'] == template['parameters']['imageDigest']['maxLength'] == 71
    assert '@' in api_parameters['image']['value'] and 'imageDigest' in api_parameters['image']['value']
    assert 'image' in api_parameters and api_parameters['image'] == artifact_parameters['image']
    assert 'imagePull' in api_parameters['pullIdentityId']['value'] or 'image-pull' in api_parameters['pullIdentityId']['value']
    checks.append('Both roles use the same digest-addressed service image and a dedicated registry pull identity')
    summary = {'passed': True, 'scope': 'local compilation and packaging contracts only; no Azure service or deployment operation', 'compiler': template['metadata']['_generator'], 'reusedCompiledTemplate': bool(args.compiled_template), 'topLevelResources': len(resources), 'checks': checks}
    (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(summary, indent=2))


if __name__ == '__main__':
    main()
