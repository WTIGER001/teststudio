param name string
param location string
param environmentId string
param image string
param registryServer string
param pullIdentityId string
param startupIdentityId string
param runtimeIdentityId string
param runtimeClientId string
param observerIdentityId string = ''
param vaultUri string
param databaseSecretName string
param storageServiceUrl string
param evidenceContainer string
param uiOrigin string
param artifactOrigin string
@allowed(['api', 'artifacts'])
param role string
param oidcConfigurationSecretName string = ''
param oidcClients array = []
param bootstrapIssuer string = ''
param bootstrapSubject string = ''
param localAccounts bool = false
@minValue(1)
@maxValue(10)
param replicas int = 1

var api = role == 'api'
var port = api ? 8040 : 8041
var observerEnabled = api && !empty(observerIdentityId)
var identities = union({
  '${pullIdentityId}': {}
  '${startupIdentityId}': {}
  '${runtimeIdentityId}': {}
}, observerEnabled ? {
  '${observerIdentityId}': {}
} : {})
var identitySettings = concat([
  { identity: pullIdentityId, lifecycle: 'None' }
  { identity: startupIdentityId, lifecycle: 'None' }
  { identity: runtimeIdentityId, lifecycle: 'Main' }
], observerEnabled ? [
  { identity: observerIdentityId, lifecycle: 'Main' }
] : [])
var secrets = concat([
  {
    name: 'database'
    keyVaultUrl: '${vaultUri}secrets/${databaseSecretName}'
    identity: startupIdentityId
  }
], api ? [
  {
    name: 'oidc-configuration'
    keyVaultUrl: '${vaultUri}secrets/${oidcConfigurationSecretName}'
    identity: startupIdentityId
  }
] : [], map(oidcClients, client => {
  name: client.name
  keyVaultUrl: '${vaultUri}secrets/${client.secretName}'
  identity: startupIdentityId
}))
var args = concat([
  '--serve-role=${role}'
  '--require-postgres=true'
  '--public-url=${uiOrigin}'
  '--artifact-origin=${artifactOrigin}'
  '--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION'
  '--execution-location=azure-container-apps'
], api ? [
  '--oidc-config-env=TESTSTUDIO_OIDC_CONFIGURATION'
  '--oidc-secret-refs=${join(map(oidcClients, client => 'env:${client.environmentName}'), ',')}'
  '--bootstrap-issuer=${bootstrapIssuer}'
  '--bootstrap-subject=${bootstrapSubject}'
  '--local-accounts=${string(localAccounts)}'
] : [])
var env = concat([
  {
    name: 'TESTSTUDIO_DATABASE'
    secretRef: 'database'
  }
  {
    name: 'TESTSTUDIO_ARTIFACT_CONFIGURATION'
    value: string({
      driver: 'azure'
      azure: {
        serviceUrl: storageServiceUrl
        container: evidenceContainer
        prefix: 'teststudio'
        managedIdentityClientId: runtimeClientId
      }
    })
  }
], api ? [
  {
    name: 'TESTSTUDIO_OIDC_CONFIGURATION'
    secretRef: 'oidc-configuration'
  }
] : [], map(oidcClients, client => {
  name: client.environmentName
  secretRef: client.name
}))

resource app 'Microsoft.App/containerApps@2025-07-01' = {
  name: name
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: identities
  }
  properties: {
    managedEnvironmentId: environmentId
    configuration: {
      activeRevisionsMode: 'Single'
      identitySettings: identitySettings
      registries: [
        { server: registryServer, identity: pullIdentityId }
      ]
      secrets: secrets
      ingress: {
        external: true
        allowInsecure: false
        targetPort: port
        transport: 'http'
        traffic: [
          { latestRevision: true, weight: 100 }
        ]
      }
    }
    template: {
      terminationGracePeriodSeconds: 90
      containers: [
        {
          name: 'service'
          image: image
          args: args
          env: env
          resources: {
            cpu: json('0.5')
            memory: '1Gi'
          }
          probes: [
            {
              type: 'Startup'
              httpGet: { path: '/healthz', port: port, scheme: 'HTTP' }
              initialDelaySeconds: 1
              periodSeconds: 3
              timeoutSeconds: 2
              failureThreshold: 30
            }
            {
              type: 'Readiness'
              httpGet: { path: '/healthz', port: port, scheme: 'HTTP' }
              periodSeconds: 10
              timeoutSeconds: 2
              failureThreshold: 3
            }
            {
              type: 'Liveness'
              httpGet: { path: '/livez', port: port, scheme: 'HTTP' }
              periodSeconds: 30
              timeoutSeconds: 2
              failureThreshold: 3
            }
          ]
        }
      ]
      scale: {
        minReplicas: replicas
        maxReplicas: replicas
      }
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

output appName string = app.name
