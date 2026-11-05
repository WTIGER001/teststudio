@minLength(3)
@maxLength(20)
param installationName string
param location string
param controllerVnetName string
param sourceSubnetName string = 'preparation'
param sourceSubnetPrefix string
param fileSubnetName string = 'journal-endpoints'
param fileSubnetPrefix string
param databaseIp string
param registryIp string
param registryDataIp string
param fileIp string
@minLength(3)
@maxLength(24)
param journalAccount string
@minLength(3)
@maxLength(63)
param journalShare string
@minValue(1)
@maxValue(5120)
param journalQuotaGiB int = 100

resource controller 'Microsoft.Network/virtualNetworks@2025-03-01' existing = { name: controllerVnetName }

var sourceRules = [
  { name: 'platform-internal-in', direction: 'Inbound', priority: 100, protocol: '*', access: 'Allow', source: sourceSubnetPrefix, destinations: [sourceSubnetPrefix], port: '*' }
  { name: 'load-balancer', direction: 'Inbound', priority: 110, protocol: 'Tcp', access: 'Allow', source: 'AzureLoadBalancer', destinations: [sourceSubnetPrefix], port: '30000-32767' }
  { name: 'deny-in', direction: 'Inbound', priority: 4096, protocol: '*', access: 'Deny', source: '*', destinations: ['*'], port: '*' }
  { name: 'deny-metadata', direction: 'Outbound', priority: 100, protocol: '*', access: 'Deny', source: '*', destinations: ['AzurePlatformIMDS'], port: '*' }
  { name: 'platform-internal-out', direction: 'Outbound', priority: 110, protocol: '*', access: 'Allow', source: sourceSubnetPrefix, destinations: [sourceSubnetPrefix], port: '*' }
  { name: 'platform-dns-udp', direction: 'Outbound', priority: 120, protocol: 'Udp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzurePlatformDNS'], port: '53' }
  { name: 'platform-dns-tcp', direction: 'Outbound', priority: 121, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzurePlatformDNS'], port: '53' }
  { name: 'deny-other-platform-dns', direction: 'Outbound', priority: 130, protocol: '*', access: 'Deny', source: '*', destinations: ['AzurePlatformDNS'], port: '*' }
  { name: 'platform-registry', direction: 'Outbound', priority: 200, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['MicrosoftContainerRegistry'], port: '443' }
  { name: 'platform-registry-cdn', direction: 'Outbound', priority: 210, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzureFrontDoor.FirstParty'], port: '443' }
  { name: 'platform-identity', direction: 'Outbound', priority: 220, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzureActiveDirectory'], port: '443' }
  { name: 'private-endpoints', direction: 'Outbound', priority: 300, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: [registryIp, registryDataIp], port: '443' }
  { name: 'service-AzureResourceManager', direction: 'Outbound', priority: 310, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzureResourceManager'], port: '443' }
  { name: 'service-AzureKeyVault', direction: 'Outbound', priority: 320, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzureKeyVault'], port: '443' }
  { name: 'service-AzureMonitor', direction: 'Outbound', priority: 330, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: ['AzureMonitor'], port: '443' }
  { name: 'database', direction: 'Outbound', priority: 350, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: [databaseIp], port: '5432' }
  { name: 'journal', direction: 'Outbound', priority: 360, protocol: 'Tcp', access: 'Allow', source: sourceSubnetPrefix, destinations: [fileIp], port: '445' }
  { name: 'deny-out', direction: 'Outbound', priority: 4096, protocol: '*', access: 'Deny', source: '*', destinations: ['*'], port: '*' }
]

resource publicIp 'Microsoft.Network/publicIPAddresses@2025-03-01' = {
  name: '${installationName}-preparation-egress'
  location: location
  sku: { name: 'Standard', tier: 'Regional' }
  properties: { publicIPAllocationMethod: 'Static', publicIPAddressVersion: 'IPv4' }
}
resource nat 'Microsoft.Network/natGateways@2025-03-01' = {
  name: '${installationName}-preparation-egress'
  location: location
  sku: { name: 'Standard' }
  properties: { idleTimeoutInMinutes: 10, publicIpAddresses: [{ id: publicIp.id }] }
}
resource sourceSecurity 'Microsoft.Network/networkSecurityGroups@2025-03-01' = {
  name: '${installationName}-preparation-source'
  location: location
  properties: {
    flushConnection: true
    securityRules: [for rule in sourceRules: {
      name: 'preparation-${rule.name}'
      properties: union({
        direction: rule.direction
        priority: rule.priority
        protocol: rule.protocol
        access: rule.access
        sourceAddressPrefix: rule.source
        sourcePortRange: '*'
        destinationPortRange: rule.port
      }, length(rule.destinations) == 1 ? { destinationAddressPrefix: rule.destinations[0] } : { destinationAddressPrefixes: rule.destinations })
    }]
  }
}
resource sourceSubnet 'Microsoft.Network/virtualNetworks/subnets@2025-03-01' = {
  parent: controller
  name: sourceSubnetName
  properties: {
    addressPrefix: sourceSubnetPrefix
    defaultOutboundAccess: false
    networkSecurityGroup: { id: sourceSecurity.id }
    natGateway: { id: nat.id }
    delegations: [{ name: 'container-apps', properties: { serviceName: 'Microsoft.App/environments' } }]
  }
}

resource fileSecurity 'Microsoft.Network/networkSecurityGroups@2025-03-01' = {
  name: '${installationName}-preparation-journal'
  location: location
  properties: {
    flushConnection: true
    securityRules: [
      { name: 'preparation-journal-smb', properties: { direction: 'Inbound', priority: 100, protocol: 'Tcp', access: 'Allow', sourceAddressPrefix: sourceSubnetPrefix, sourcePortRange: '*', destinationAddressPrefix: fileIp, destinationPortRange: '445' } }
      { name: 'endpoint-deny-in', properties: { direction: 'Inbound', priority: 4096, protocol: '*', access: 'Deny', sourceAddressPrefix: '*', sourcePortRange: '*', destinationAddressPrefix: '*', destinationPortRange: '*' } }
      { name: 'endpoint-deny-out', properties: { direction: 'Outbound', priority: 4096, protocol: '*', access: 'Deny', sourceAddressPrefix: '*', sourcePortRange: '*', destinationAddressPrefix: '*', destinationPortRange: '*' } }
    ]
  }
}
resource fileSubnet 'Microsoft.Network/virtualNetworks/subnets@2025-03-01' = {
  parent: controller
  name: fileSubnetName
  properties: {
    addressPrefix: fileSubnetPrefix
    defaultOutboundAccess: false
    networkSecurityGroup: { id: fileSecurity.id }
    privateEndpointNetworkPolicies: 'Enabled'
    privateLinkServiceNetworkPolicies: 'Enabled'
  }
}
resource account 'Microsoft.Storage/storageAccounts@2024-01-01' = {
  name: journalAccount
  location: location
  kind: 'StorageV2'
  sku: { name: 'Standard_LRS' }
  properties: {
    allowBlobPublicAccess: false
    allowSharedKeyAccess: true
    allowCrossTenantReplication: false
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
    publicNetworkAccess: 'Disabled'
    networkAcls: { defaultAction: 'Deny', bypass: 'None', ipRules: [], virtualNetworkRules: [], resourceAccessRules: [] }
  }
}
resource fileService 'Microsoft.Storage/storageAccounts/fileServices@2024-01-01' = { parent: account, name: 'default', properties: {} }
resource share 'Microsoft.Storage/storageAccounts/fileServices/shares@2024-01-01' = {
  parent: fileService
  name: journalShare
  properties: { enabledProtocols: 'SMB', shareQuota: journalQuotaGiB }
}
resource preserve 'Microsoft.Authorization/locks@2020-05-01' = {
  scope: account
  name: 'preserve-teststudio-source-journals'
  properties: { level: 'CanNotDelete', notes: 'Preserve owned source recovery journals during worker replacement.' }
}
resource endpoint 'Microsoft.Network/privateEndpoints@2025-03-01' = {
  name: '${installationName}-preparation-journal'
  location: location
  properties: {
    subnet: { id: fileSubnet.id }
    customNetworkInterfaceName: '${installationName}-preparation-journal'
    privateLinkServiceConnections: [{ name: 'journal-file', properties: { privateLinkServiceId: account.id, groupIds: ['file'] } }]
    ipConfigurations: [{ name: 'file', properties: { groupId: 'file', memberName: 'file', privateIPAddress: fileIp } }]
  }
}
resource zone 'Microsoft.Network/privateDnsZones@2024-06-01' = { name: 'privatelink.file.${environment().suffixes.storage}', location: 'global' }
resource link 'Microsoft.Network/privateDnsZones/virtualNetworkLinks@2024-06-01' = {
  parent: zone
  name: 'preparation-journal'
  location: 'global'
  properties: { virtualNetwork: { id: controller.id }, registrationEnabled: false, resolutionPolicy: 'Default' }
}
resource record 'Microsoft.Network/privateDnsZones/A@2024-06-01' = {
  parent: zone
  name: journalAccount
  properties: { ttl: 60, aRecords: [{ ipv4Address: fileIp }] }
}

output networkBinding object = {
  sourceSecurityGroup: sourceSecurity.id
  natGateway: nat.id
  natPublicIp: publicIp.id
  databaseIp: databaseIp
  storageAccount: account.id
  fileEndpoint: endpoint.id
  fileInterface: resourceId('Microsoft.Network/networkInterfaces', '${installationName}-preparation-journal')
  fileSubnet: fileSubnet.id
  fileSubnetPrefix: fileSubnetPrefix
  fileSecurityGroup: fileSecurity.id
  fileIp: fileIp
  fileZone: zone.id
}
output sourceSubnetId string = sourceSubnet.id
output journalHostBinding object = { journalAccount: account.name, journalShare: share.name, journalStorageType: 'AzureFile' }
