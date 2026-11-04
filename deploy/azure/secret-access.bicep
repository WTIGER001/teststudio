param vaultName string
param bindings array

resource vault 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: vaultName
}

resource secrets 'Microsoft.KeyVault/vaults/secrets@2023-07-01' existing = [for binding in bindings: {
  parent: vault
  name: binding.secretName
}]

resource grants 'Microsoft.Authorization/roleAssignments@2022-04-01' = [for (binding, index) in bindings: {
  scope: secrets[index]
  name: guid(secrets[index].id, binding.principalId, 'teststudio-secret-read')
  properties: {
    principalId: binding.principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4633458b-17de-408a-b874-0445c86b69e6')
  }
}]
