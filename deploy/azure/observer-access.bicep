targetScope = 'subscription'

param installationKey string
param principalId string

resource role 'Microsoft.Authorization/roleDefinitions@2022-04-01' = {
  name: guid(installationKey, 'teststudio-observer-role')
  properties: {
    roleName: 'TestStudio observer ${uniqueString(installationKey)}'
    description: 'Read installation metadata and observe DNS resolver policy associations.'
    type: 'CustomRole'
    assignableScopes: [subscription().id]
    permissions: [
      {
        actions: ['*/read', 'Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action']
        notActions: []
        dataActions: []
        notDataActions: []
      }
    ]
  }
}

resource assignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(installationKey, principalId, 'teststudio-observer-assignment')
  properties: {
    principalId: principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: role.id
  }
}

output roleId string = role.id
output assignmentId string = assignment.id
