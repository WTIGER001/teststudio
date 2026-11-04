@minLength(3)
@maxLength(12)
param installationName string
param location string
param infrastructureSubnetId string
param infrastructureResourceGroup string
param registryName string
param registryResourceGroup string
@minLength(1)
param imageRepository string
@minLength(71)
@maxLength(71)
param imageDigest string
param keyVaultName string
param keyVaultResourceGroup string
@minLength(1)
@maxLength(127)
param databaseSecretName string
param storageAccountName string
param storageResourceGroup string
@minLength(3)
@maxLength(63)
param evidenceContainer string
@minValue(1)
@maxValue(10)
param replicas int = 1

var appName = '${installationName}-worker-backend'
var environmentName = '${installationName}-backend'

resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: registryName
  scope: resourceGroup(registryResourceGroup)
}
resource vault 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: keyVaultName
  scope: resourceGroup(keyVaultResourceGroup)
}
resource storage 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: storageAccountName
  scope: resourceGroup(storageResourceGroup)
}

resource environment 'Microsoft.App/managedEnvironments@2025-07-01' = {
  name: environmentName
  location: location
  properties: {
    publicNetworkAccess: 'Disabled'
    infrastructureResourceGroup: infrastructureResourceGroup
    vnetConfiguration: {
      infrastructureSubnetId: infrastructureSubnetId
      internal: true
    }
    appLogsConfiguration: { destination: 'none' }
    workloadProfiles: [
      { name: 'Consumption', workloadProfileType: 'Consumption' }
    ]
  }
}

resource pull 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-backend-pull'
  location: location
}
resource startup 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-backend-startup'
  location: location
}
resource runtime 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-backend-runtime'
  location: location
}

module registryAccess './registry-access.bicep' = {
  name: '${installationName}-backend-registry'
  scope: resourceGroup(registryResourceGroup)
  params: { registryName: registryName, principalId: pull.properties.principalId }
}
module secretAccess './secret-access.bicep' = {
  name: '${installationName}-backend-secret'
  scope: resourceGroup(keyVaultResourceGroup)
  params: {
    vaultName: keyVaultName
    bindings: [
      { secretName: databaseSecretName, principalId: startup.properties.principalId }
    ]
  }
}
module blobAccess './blob-access.bicep' = {
  name: '${installationName}-backend-artifacts'
  scope: resourceGroup(storageResourceGroup)
  params: {
    accountName: storageAccountName
    containerName: evidenceContainer
    principalId: runtime.properties.principalId
    roleId: 'ba92f5b4-2d11-453d-a403-e96b0029c9fe'
  }
}

resource app 'Microsoft.App/containerApps@2025-07-01' = {
  name: appName
  location: location
  dependsOn: [registryAccess, secretAccess, blobAccess]
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${pull.id}': {}
      '${startup.id}': {}
      '${runtime.id}': {}
    }
  }
  properties: {
    managedEnvironmentId: environment.id
    workloadProfileName: 'Consumption'
    configuration: {
      activeRevisionsMode: 'Single'
      maxInactiveRevisions: 1
      identitySettings: [
        { identity: pull.id, lifecycle: 'None' }
        { identity: startup.id, lifecycle: 'None' }
        { identity: runtime.id, lifecycle: 'Main' }
      ]
      registries: [
        { server: registry.properties.loginServer, identity: pull.id }
      ]
      secrets: [
        {
          name: 'database'
          keyVaultUrl: '${vault.properties.vaultUri}secrets/${databaseSecretName}'
          identity: startup.id
        }
      ]
      ingress: {
        external: true
        allowInsecure: false
        targetPort: 8042
        transport: 'http'
        traffic: [
          { latestRevision: true, weight: 100 }
        ]
      }
    }
    template: {
      terminationGracePeriodSeconds: 30
      containers: [
        {
          name: 'worker-backend'
          image: '${registry.properties.loginServer}/${imageRepository}@${imageDigest}'
          command: ['/app/teststudio']
          args: [
            '--serve-role=worker'
            '--worker-addr=0.0.0.0:8042'
            '--database-env=TESTSTUDIO_DATABASE'
            '--require-postgres=true'
            '--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION'
            '--execution-location=azure-container-apps'
          ]
          env: [
            { name: 'TESTSTUDIO_DATABASE', secretRef: 'database' }
            {
              name: 'TESTSTUDIO_ARTIFACT_CONFIGURATION'
              value: string({
                driver: 'azure'
                azure: {
                  serviceUrl: storage.properties.primaryEndpoints.blob
                  container: evidenceContainer
                  prefix: 'teststudio'
                  managedIdentityClientId: runtime.properties.clientId
                }
              })
            }
          ]
          resources: { cpu: json('0.5'), memory: '1Gi' }
          probes: [
            {
              type: 'Startup'
              httpGet: { path: '/healthz', port: 8042, scheme: 'HTTP' }
              initialDelaySeconds: 1
              periodSeconds: 3
              timeoutSeconds: 2
              failureThreshold: 30
            }
            {
              type: 'Readiness'
              httpGet: { path: '/healthz', port: 8042, scheme: 'HTTP' }
              periodSeconds: 10
              timeoutSeconds: 2
              failureThreshold: 3
            }
            {
              type: 'Liveness'
              httpGet: { path: '/livez', port: 8042, scheme: 'HTTP' }
              periodSeconds: 30
              timeoutSeconds: 2
              failureThreshold: 3
            }
          ]
        }
      ]
      scale: { minReplicas: replicas, maxReplicas: replicas }
    }
  }
}

resource authentication 'Microsoft.App/containerApps/authConfigs@2025-07-01' = {
  parent: app
  name: 'current'
  properties: {
    platform: { enabled: false }
  }
}

output backendOrigin string = 'https://${appName}.${environment.properties.defaultDomain}'
output backendApplicationId string = app.id
output backendEnvironmentId string = environment.id
