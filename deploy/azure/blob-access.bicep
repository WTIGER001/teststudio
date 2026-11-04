param accountName string
param containerName string
param principalId string
@allowed(['ba92f5b4-2d11-453d-a403-e96b0029c9fe', '2a2b9908-6ea1-4ae2-8e65-a410df84e7d1'])
param roleId string

resource account 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: accountName
}
resource service 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' existing = {
  parent: account
  name: 'default'
}
resource container 'Microsoft.Storage/storageAccounts/blobServices/containers@2023-05-01' existing = {
  parent: service
  name: containerName
}
resource access 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: container
  name: guid(container.id, principalId, roleId)
  properties: {
    principalId: principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', roleId)
  }
}
