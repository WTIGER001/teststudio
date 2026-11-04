import argparse
import json
import os
from pathlib import Path
import subprocess


ROOT = Path(__file__).resolve().parents[1]


def check_package(template):
    resources = template['resources']
    kinds = [resource['type'] for resource in resources]
    assert sorted(kinds) == sorted([
        'Microsoft.App/managedEnvironments', 'Microsoft.App/containerApps',
        'Microsoft.App/containerApps/authConfigs',
        *['Microsoft.ManagedIdentity/userAssignedIdentities'] * 3,
        *['Microsoft.Resources/deployments'] * 3,
    ])
    checks = []
    environment = next(resource for resource in resources if resource['type'] == 'Microsoft.App/managedEnvironments')
    properties = environment['properties']
    assert properties['publicNetworkAccess'] == 'Disabled'
    assert properties['vnetConfiguration'] == {'infrastructureSubnetId': "[parameters('infrastructureSubnetId')]", 'internal': True}
    assert properties['infrastructureResourceGroup'] == "[parameters('infrastructureResourceGroup')]"
    assert properties['appLogsConfiguration'] == {'destination': 'none'}
    assert properties['workloadProfiles'] == [{'name': 'Consumption', 'workloadProfileType': 'Consumption'}]
    assert 'identity' not in environment
    checks.append('Dedicated internal environment disables public network access and has no environment identity or telemetry credentials')
    app = next(resource for resource in resources if resource['type'] == 'Microsoft.App/containerApps')
    assert app['properties']['managedEnvironmentId'] == "[resourceId('Microsoft.App/managedEnvironments', variables('environmentName'))]"
    assert app['properties']['workloadProfileName'] == 'Consumption'
    config = app['properties']['configuration']
    assert config['activeRevisionsMode'] == 'Single' and config['maxInactiveRevisions'] == 1
    assert config['ingress'] == {'external': True, 'allowInsecure': False, 'targetPort': 8042, 'transport': 'http', 'traffic': [{'latestRevision': True, 'weight': 100}]}
    authentication = next(resource for resource in resources if resource['type'] == 'Microsoft.App/containerApps/authConfigs')
    assert authentication['properties'] == {'platform': {'enabled': False}}
    assert 'appName' in authentication['name'] and 'current' in authentication['name']
    checks.append('Environment-private HTTPS ingress reaches only the worker port and platform user authentication is disabled explicitly')
    definition = app['properties']['template']
    assert len(definition['containers']) == 1 and set(definition) == {'terminationGracePeriodSeconds', 'containers', 'scale'}
    container = definition['containers'][0]
    assert container['command'] == ['/app/teststudio']
    assert container['args'] == ['--serve-role=worker', '--worker-addr=0.0.0.0:8042', '--database-env=TESTSTUDIO_DATABASE', '--require-postgres=true', '--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION', '--execution-location=azure-container-apps']
    assert container['resources'] == {'cpu': "[json('0.5')]", 'memory': '1Gi'}
    assert set(container) == {'name', 'image', 'command', 'args', 'env', 'resources', 'probes'}
    assert '@{2}' in container['image'] and "parameters('imageDigest')" in container['image']
    assert template['parameters']['imageDigest']['minLength'] == template['parameters']['imageDigest']['maxLength'] == 71
    assert definition['scale'] == {'minReplicas': "[parameters('replicas')]", 'maxReplicas': "[parameters('replicas')]"}
    assert template['parameters']['replicas']['minValue'] == 1
    checks.append('One digest-addressed service image overrides the image entrypoint and supplies only accepted worker-role configuration')
    settings = config['identitySettings']
    assert len(settings) == 3
    for setting, suffix, lifecycle in zip(settings, ['pull', 'startup', 'runtime'], ['None', 'None', 'Main']):
        assert '-backend-' + suffix in setting['identity'] and setting['lifecycle'] == lifecycle
    assert app['identity']['type'] == 'UserAssigned' and len(app['identity']['userAssignedIdentities']) == 3
    assert len(config['registries']) == 1 and config['registries'][0]['identity'] == settings[0]['identity']
    assert len(config['secrets']) == 1 and config['secrets'][0]['name'] == 'database'
    assert config['secrets'][0]['identity'] == settings[1]['identity']
    assert 'secrets/{1}' in config['secrets'][0]['keyVaultUrl'] and 'databaseSecretName' in config['secrets'][0]['keyVaultUrl']
    env = container['env']
    assert env[0] == {'name': 'TESTSTUDIO_DATABASE', 'secretRef': 'database'}
    assert len(env) == 2 and env[1]['name'] == 'TESTSTUDIO_ARTIFACT_CONFIGURATION'
    assert "'driver', 'azure'" in env[1]['value'] and "'prefix', 'teststudio'" in env[1]['value']
    assert '-backend-runtime' in env[1]['value'] and '.clientId' in env[1]['value']
    checks.append('Platform-only pull and database-secret identities remain separate from the selected Blob runtime identity')
    deployments = [resource for resource in resources if resource['type'] == 'Microsoft.Resources/deployments']
    secret = next(resource for resource in deployments if 'bindings' in resource['properties']['parameters'])
    bindings = secret['properties']['parameters']['bindings']['value']
    assert len(bindings) == 1 and '-backend-startup' in bindings[0]['principalId']
    assert bindings[0]['secretName'] == "[parameters('databaseSecretName')]"
    assert secret['resourceGroup'] == "[parameters('keyVaultResourceGroup')]"
    grant = secret['properties']['template']['resources'][0]
    assert 'Microsoft.KeyVault/vaults/secrets' in grant['scope']
    assert '4633458b-17de-408a-b874-0445c86b69e6' in grant['properties']['roleDefinitionId']
    blob = next(resource for resource in deployments if 'roleId' in resource['properties']['parameters'])
    parameters = blob['properties']['parameters']
    assert parameters['roleId']['value'] == 'ba92f5b4-2d11-453d-a403-e96b0029c9fe'
    assert '-backend-runtime' in parameters['principalId']['value']
    assert parameters['accountName']['value'] == "[parameters('storageAccountName')]" and parameters['containerName']['value'] == "[parameters('evidenceContainer')]"
    assert blob['resourceGroup'] == "[parameters('storageResourceGroup')]"
    assert 'Microsoft.Storage/storageAccounts/blobServices/containers' in blob['properties']['template']['resources'][0]['scope']
    registry = next(resource for resource in deployments if 'registryName' in resource['properties']['parameters'])
    assert '-backend-pull' in registry['properties']['parameters']['principalId']['value']
    assert registry['resourceGroup'] == "[parameters('registryResourceGroup')]"
    assert '7f951dda-4ed3-4680-a7ca-43fe172d538d' in registry['properties']['template']['resources'][0]['properties']['roleDefinitionId']
    dependencies = json.dumps(app['dependsOn'])
    assert all(name in dependencies for name in ['environmentName', '-backend-registry', '-backend-secret', '-backend-artifacts'])
    checks.append('Existing scoped grant modules restrict startup to one database secret, runtime to the selected evidence container, and pull to the registry')
    probes = {probe['type']: probe for probe in container['probes']}
    assert set(probes) == {'Startup', 'Readiness', 'Liveness'}
    assert all(probe['httpGet']['port'] == 8042 and probe['httpGet']['scheme'] == 'HTTP' for probe in probes.values())
    assert probes['Startup']['httpGet']['path'] == probes['Readiness']['httpGet']['path'] == '/healthz'
    assert probes['Liveness']['httpGet']['path'] == '/livez'
    assert definition['terminationGracePeriodSeconds'] >= 10
    outputs = template['outputs']
    assert set(outputs) == {'backendOrigin', 'backendApplicationId', 'backendEnvironmentId'}
    assert 'https://' in outputs['backendOrigin']['value'] and '.defaultDomain' in outputs['backendOrigin']['value']
    assert 'appName' in outputs['backendOrigin']['value'] and 'environmentName' in outputs['backendOrigin']['value']
    assert 'Microsoft.App/containerApps' in outputs['backendApplicationId']['value'] and 'Microsoft.App/managedEnvironments' in outputs['backendEnvironmentId']['value']
    document = json.dumps(template)
    assert not any(value in document for value in ['getSecret', 'listKeys', 'docker.sock', 'amazonaws', 'AWS_ACCESS_KEY', 'oidc-', '--public-url', '--ui=', '--source-bundles', 'Microsoft.App/jobs'])
    checks.append('Worker probes, graceful termination, and resource-bound HTTPS outputs add no user API, execution job, secret retrieval, or AWS dependency')
    return checks


def main():
    parser = argparse.ArgumentParser(description='Compile and inspect only the staged Azure worker backend package locally, without Azure CLI or deployment.')
    parser.add_argument('--bicep', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    environment = {key: value for key, value in os.environ.items() if not key.startswith(('AZURE_', 'ARM_', 'AWS_'))}
    path = output / 'backend.json'
    result = subprocess.run([str(args.bicep.resolve()), 'build', str(ROOT / 'deploy/azure/worker-backend.bicep'), '--no-restore', '--outfile', str(path)], capture_output=True, text=True, env=environment, timeout=60)
    (output / 'compile.log').write_text(result.stdout + result.stderr)
    assert result.returncode == 0 and 'Warning' not in result.stderr, 'Inspect the local compile.log for compiler or schema errors'
    template = json.loads(path.read_text())
    checks = check_package(template)
    summary = {'passed': True, 'scope': 'local compilation and backend packaging only; no Azure deployment or service call', 'compiler': template['metadata']['_generator'], 'checks': checks, 'cloudOperations': False, 'imageBuilds': False, 'uiBuilds': False, 'browserRuns': False}
    (output / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(summary, indent=2))


if __name__ == '__main__':
    main()
