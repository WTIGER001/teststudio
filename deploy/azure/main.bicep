@minLength(3)
@maxLength(20)
param installationName string
param location string
param environmentName string
param environmentResourceGroup string
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
@minLength(1)
@maxLength(127)
param oidcConfigurationSecretName string
@maxLength(127)
param oidcClientOneSecretName string = ''
@maxLength(127)
param oidcClientTwoSecretName string = ''
@minLength(1)
param bootstrapIssuer string
@minLength(1)
@maxLength(512)
param bootstrapSubject string
param localAccounts bool = false
@minValue(1)
@maxValue(10)
param apiReplicas int = 1
@minValue(1)
@maxValue(10)
param artifactReplicas int = 1

resource environment 'Microsoft.App/managedEnvironments@2025-07-01' existing = {
  name: environmentName
  scope: resourceGroup(environmentResourceGroup)
}
resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: registryName
  scope: resourceGroup(registryResourceGroup)
}
resource vault 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: keyVaultName
  scope: resourceGroup(keyVaultResourceGroup)
}

var apiName = '${installationName}-api'
var artifactName = '${installationName}-artifacts'
var uiOrigin = 'https://${apiName}.${environment.properties.defaultDomain}'
var artifactOrigin = 'https://${artifactName}.${environment.properties.defaultDomain}'
var image = '${registry.properties.loginServer}/${imageRepository}@${imageDigest}'
var oidcClients = concat(empty(oidcClientOneSecretName) ? [] : [
  { name: 'oidc-client-one', environmentName: 'TESTSTUDIO_OIDC_CLIENT_ONE', secretName: oidcClientOneSecretName }
], empty(oidcClientTwoSecretName) ? [] : [
  { name: 'oidc-client-two', environmentName: 'TESTSTUDIO_OIDC_CLIENT_TWO', secretName: oidcClientTwoSecretName }
])

resource apiRuntime 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-api-runtime'
  location: location
}
resource artifactRuntime 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-artifact-runtime'
  location: location
}
resource apiStartup 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-api-startup'
  location: location
}
resource artifactStartup 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-artifact-startup'
  location: location
}
resource imagePull 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-image-pull'
  location: location
}

resource storage 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: 'ts${uniqueString(resourceGroup().id, installationName)}'
  location: location
  kind: 'StorageV2'
  sku: { name: 'Standard_LRS' }
  properties: {
    allowBlobPublicAccess: false
    allowSharedKeyAccess: false
    defaultToOAuthAuthentication: true
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
    publicNetworkAccess: 'Enabled'
  }
}
resource blobService 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' = {
  parent: storage
  name: 'default'
  properties: {
    deleteRetentionPolicy: { enabled: false }
    containerDeleteRetentionPolicy: { enabled: false }
  }
}
resource evidence 'Microsoft.Storage/storageAccounts/blobServices/containers@2023-05-01' = {
  parent: blobService
  name: 'evidence'
  properties: { publicAccess: 'None' }
}
resource storageLock 'Microsoft.Authorization/locks@2020-05-01' = {
  scope: storage
  name: 'preserve-teststudio-evidence'
  properties: {
    level: 'CanNotDelete'
    notes: 'Remove only after reviewing TestStudio evidence retention and backups.'
  }
}
module apiBlobAccess './blob-access.bicep' = {
  name: '${installationName}-blob-writer'
  params: {
    accountName: storage.name
    containerName: evidence.name
    principalId: apiRuntime.properties.principalId
    roleId: 'ba92f5b4-2d11-453d-a403-e96b0029c9fe'
  }
}
module artifactBlobAccess './blob-access.bicep' = {
  name: '${installationName}-blob-reader'
  params: {
    accountName: storage.name
    containerName: evidence.name
    principalId: artifactRuntime.properties.principalId
    roleId: '2a2b9908-6ea1-4ae2-8e65-a410df84e7d1'
  }
}

module secretAccess './secret-access.bicep' = {
  name: '${installationName}-secret-access'
  scope: resourceGroup(keyVaultResourceGroup)
  params: {
    vaultName: keyVaultName
    bindings: concat([
      { secretName: databaseSecretName, principalId: apiStartup.properties.principalId }
      { secretName: databaseSecretName, principalId: artifactStartup.properties.principalId }
      { secretName: oidcConfigurationSecretName, principalId: apiStartup.properties.principalId }
    ], map(oidcClients, client => {
      secretName: client.secretName
      principalId: apiStartup.properties.principalId
    }))
  }
}
module registryAccess './registry-access.bicep' = {
  name: '${installationName}-registry-access'
  scope: resourceGroup(registryResourceGroup)
  params: { registryName: registryName, principalId: imagePull.properties.principalId }
}

module api './app.bicep' = {
  name: '${installationName}-api'
  dependsOn: [secretAccess, registryAccess, apiBlobAccess]
  params: {
    name: apiName
    role: 'api'
    location: location
    environmentId: environment.id
    image: image
    registryServer: registry.properties.loginServer
    pullIdentityId: imagePull.id
    startupIdentityId: apiStartup.id
    runtimeIdentityId: apiRuntime.id
    runtimeClientId: apiRuntime.properties.clientId
    vaultUri: vault.properties.vaultUri
    databaseSecretName: databaseSecretName
    oidcConfigurationSecretName: oidcConfigurationSecretName
    oidcClients: oidcClients
    storageServiceUrl: storage.properties.primaryEndpoints.blob
    evidenceContainer: evidence.name
    uiOrigin: uiOrigin
    artifactOrigin: artifactOrigin
    bootstrapIssuer: bootstrapIssuer
    bootstrapSubject: bootstrapSubject
    localAccounts: localAccounts
    replicas: apiReplicas
  }
}
module artifacts './app.bicep' = {
  name: '${installationName}-artifacts'
  dependsOn: [secretAccess, registryAccess, artifactBlobAccess]
  params: {
    name: artifactName
    role: 'artifacts'
    location: location
    environmentId: environment.id
    image: image
    registryServer: registry.properties.loginServer
    pullIdentityId: imagePull.id
    startupIdentityId: artifactStartup.id
    runtimeIdentityId: artifactRuntime.id
    runtimeClientId: artifactRuntime.properties.clientId
    vaultUri: vault.properties.vaultUri
    databaseSecretName: databaseSecretName
    storageServiceUrl: storage.properties.primaryEndpoints.blob
    evidenceContainer: evidence.name
    uiOrigin: uiOrigin
    artifactOrigin: artifactOrigin
    replicas: artifactReplicas
  }
}

output uiUrl string = uiOrigin
output artifactUrl string = artifactOrigin
output storageAccountName string = storage.name
output apiAppName string = api.outputs.appName
output artifactAppName string = artifacts.outputs.appName
