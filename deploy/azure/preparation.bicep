@minLength(3)
@maxLength(20)
param installationName string
param location string
param environmentName string
param environmentResourceGroup string
@minLength(1)
param journalStorageName string
@minLength(3)
@maxLength(24)
param journalAccount string
@minLength(3)
@maxLength(63)
param journalShare string
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
param assemblyConfigurationSecretName string
param goRuntime string = ''
param playwrightRuntime string = ''
param workerEnabled bool = false

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

resource writer 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-source-writer'
  location: location
}
resource reader 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-source-reader'
  location: location
}
resource pull 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-source-pull'
  location: location
}
resource startup 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${installationName}-source-startup'
  location: location
}

module pushAccess './registry-push-access.bicep' = {
  name: '${installationName}-source-push'
  scope: resourceGroup(registryResourceGroup)
  params: { registryName: registryName, principalId: writer.properties.principalId }
}
module pullAccess './registry-access.bicep' = {
  name: '${installationName}-source-pull'
  scope: resourceGroup(registryResourceGroup)
  params: { registryName: registryName, principalId: pull.properties.principalId }
}
module readAccess './observer-access.bicep' = {
  name: '${installationName}-source-read'
  scope: subscription()
  params: { installationKey: '${resourceGroup().id}/${installationName}/source-reader', principalId: reader.properties.principalId }
}
module secretAccess './secret-access.bicep' = {
  name: '${installationName}-source-secrets'
  scope: resourceGroup(keyVaultResourceGroup)
  params: {
    vaultName: keyVaultName
    bindings: [
      { secretName: databaseSecretName, principalId: startup.properties.principalId }
      { secretName: assemblyConfigurationSecretName, principalId: startup.properties.principalId }
    ]
  }
}

var args = concat([
  '--database-env=TESTSTUDIO_DATABASE'
  '--require-postgres=true'
  '--assembly-only=true'
  '--execution-location=azure-container-apps'
  '--azure-assembly-env=TESTSTUDIO_AZURE_ASSEMBLY'
  '--build-directory=/private/source-journals'
], empty(goRuntime) ? [] : ['--registry-go-runtime=${goRuntime}'], empty(playwrightRuntime) ? [] : ['--registry-playwright-runtime=${playwrightRuntime}'])
var image = '${registry.properties.loginServer}/${imageRepository}@${imageDigest}'

resource preparation 'Microsoft.App/containerApps@2025-07-01' = if (workerEnabled) {
  name: '${installationName}-preparation'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${writer.id}': {}
      '${reader.id}': {}
      '${pull.id}': {}
      '${startup.id}': {}
    }
  }
  properties: {
    managedEnvironmentId: environment.id
    workloadProfileName: 'Consumption'
    configuration: {
      activeRevisionsMode: 'Single'
      identitySettings: [
        { identity: writer.id, lifecycle: 'Main' }
        { identity: reader.id, lifecycle: 'Main' }
        { identity: pull.id, lifecycle: 'None' }
        { identity: startup.id, lifecycle: 'None' }
      ]
      registries: [{ server: registry.properties.loginServer, identity: pull.id }]
      secrets: [
        { name: 'database', keyVaultUrl: '${vault.properties.vaultUri}secrets/${databaseSecretName}', identity: startup.id }
        { name: 'assembly-configuration', keyVaultUrl: '${vault.properties.vaultUri}secrets/${assemblyConfigurationSecretName}', identity: startup.id }
      ]
    }
    template: {
      terminationGracePeriodSeconds: 90
      containers: [
        {
          name: 'preparation'
          image: image
          command: ['/app/teststudio-prepare']
          args: args
          env: [
            { name: 'TESTSTUDIO_DATABASE', secretRef: 'database' }
            { name: 'TESTSTUDIO_AZURE_ASSEMBLY', secretRef: 'assembly-configuration' }
          ]
          resources: { cpu: 2, memory: '4Gi' }
          volumeMounts: [{ volumeName: 'source-journals', mountPath: '/private/source-journals' }]
        }
      ]
      volumes: [
        {
          name: 'source-journals'
          storageType: 'AzureFile'
          storageName: journalStorageName
          mountOptions: 'dir_mode=0700,file_mode=0600,uid=65532,gid=65532,cache=none'
        }
      ]
      scale: { minReplicas: 1, maxReplicas: 1 }
    }
  }
  dependsOn: [pushAccess, pullAccess, readAccess, secretAccess]
}

output assemblyBinding object = {
  application: resourceId('Microsoft.App/containerApps', '${installationName}-preparation')
  environment: environment.id
  readerClientId: reader.properties.clientId
  readerIdentity: reader.id
  readerAssignment: readAccess.outputs.assignmentId
  readerRole: readAccess.outputs.roleId
  writer: {
    identity: writer.id
    clientId: writer.properties.clientId
    principalId: writer.properties.principalId
    tenantId: writer.properties.tenantId
    assignment: pushAccess.outputs.assignmentId
  }
  journalStorageName: journalStorageName
  journalDirectory: '/private/source-journals'
  host: {
    application: resourceId('Microsoft.App/containerApps', '${installationName}-preparation')
    environment: environment.id
    image: image
    readerIdentity: reader.id
    readerPrincipalId: reader.properties.principalId
    readerAssignment: readAccess.outputs.assignmentId
    readerRole: readAccess.outputs.roleId
    pullIdentity: pull.id
    startupIdentity: startup.id
    journalStorageName: journalStorageName
    journalAccount: journalAccount
    journalShare: journalShare
    databaseSecretUrl: '${vault.properties.vaultUri}secrets/${databaseSecretName}'
    configurationSecretUrl: '${vault.properties.vaultUri}secrets/${assemblyConfigurationSecretName}'
    goRuntime: goRuntime
    playwrightRuntime: playwrightRuntime
  }
}
