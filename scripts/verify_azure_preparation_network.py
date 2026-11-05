import json


def verify_preparation_network(template):
    resources = template['resources']
    by_type = {}
    for resource in resources:
        by_type.setdefault(resource['type'], []).append(resource)
    assert len(resources) == 14
    assert not any(resource['type'].startswith('Microsoft.App/') for resource in resources)
    groups = by_type['Microsoft.Network/networkSecurityGroups']
    assert len(groups) == 2 and all(group['properties']['flushConnection'] for group in groups)
    rules = template['variables']['sourceRules']
    assert len(rules) == 18
    rule_map = {rule['name']: rule for rule in rules}
    assert len(rule_map) == len(rules)
    assert rule_map['database']['destinations'] == ["[parameters('databaseIp')]"] and rule_map['database']['port'] == '5432'
    assert rule_map['journal']['destinations'] == ["[parameters('fileIp')]"] and rule_map['journal']['port'] == '445'
    assert rule_map['private-endpoints']['destinations'] == ["[parameters('registryIp')]", "[parameters('registryDataIp')]"]
    assert rule_map['private-endpoints']['port'] == '443'
    assert rule_map['deny-out']['access'] == rule_map['deny-in']['access'] == 'Deny'
    for name, direction in [('deny-in', 'Inbound'), ('deny-out', 'Outbound')]:
        rule = rule_map[name]
        assert rule['direction'] == direction and rule['priority'] == 4096 and rule['source'] == '*' and rule['destinations'] == ['*'] and rule['protocol'] == rule['port'] == '*'
    assert rule_map['deny-metadata']['destinations'] == ['AzurePlatformIMDS']
    allowed_tags = {'MicrosoftContainerRegistry', 'AzureFrontDoor.FirstParty', 'AzureActiveDirectory', 'AzureResourceManager', 'AzureKeyVault', 'AzureMonitor'}
    for rule in rules:
        if rule['access'] != 'Allow' or rule['direction'] != 'Outbound':
            continue
        assert rule['source'] == "[parameters('sourceSubnetPrefix')]"
        if rule['name'] == 'platform-internal-out':
            assert rule['destinations'] == ["[parameters('sourceSubnetPrefix')]"]
        elif rule['name'].startswith('platform-dns-'):
            assert rule['destinations'] == ['AzurePlatformDNS'] and rule['port'] == '53'
        elif rule['name'] not in {'database', 'journal', 'private-endpoints'}:
            assert len(rule['destinations']) == 1 and rule['destinations'][0] in allowed_tags and rule['port'] == '443' and rule['protocol'] == 'Tcp'
    source_group = next(group for group in groups if 'preparation-source' in group['name'])
    copy = source_group['properties']['copy']
    assert len(copy) == 1 and copy[0]['name'] == 'securityRules' and copy[0]['count'] == "[length(variables('sourceRules'))]"
    properties = copy[0]['input']['properties']
    for field in ['direction', 'priority', 'protocol', 'access', 'source', 'port', 'destinations']:
        assert "variables('sourceRules')[copyIndex('securityRules')]." + field in properties
    checks = ['Dedicated source NSG permits platform dependencies and exact private database, registry, and SMB destinations, then denies remaining egress']
    nat = by_type['Microsoft.Network/natGateways'][0]
    public_ip = by_type['Microsoft.Network/publicIPAddresses'][0]
    assert nat['sku'] == {'name': 'Standard'} and nat['properties']['idleTimeoutInMinutes'] == 10
    assert len(nat['properties']['publicIpAddresses']) == 1
    assert public_ip['sku'] == {'name': 'Standard', 'tier': 'Regional'}
    assert public_ip['properties'] == {'publicIPAllocationMethod': 'Static', 'publicIPAddressVersion': 'IPv4'}
    subnets = by_type['Microsoft.Network/virtualNetworks/subnets']
    source = next(subnet for subnet in subnets if 'sourceSubnetName' in subnet['name'])
    file = next(subnet for subnet in subnets if 'fileSubnetName' in subnet['name'])
    assert not source['properties']['defaultOutboundAccess'] and 'natGateway' in source['properties'] and 'routeTable' not in source['properties']
    assert source['properties']['delegations'] == [{'name': 'container-apps', 'properties': {'serviceName': 'Microsoft.App/environments'}}]
    assert not file['properties']['defaultOutboundAccess'] and file['properties']['privateEndpointNetworkPolicies'] == file['properties']['privateLinkServiceNetworkPolicies'] == 'Enabled'
    checks.append('Explicit single-address NAT and separate delegated source and private endpoint subnets avoid default outbound access and unexpected routing')
    account = by_type['Microsoft.Storage/storageAccounts'][0]
    p = account['properties']
    assert account['kind'] == 'StorageV2' and account['sku'] == {'name': 'Standard_LRS'}
    assert p['publicNetworkAccess'] == 'Disabled' and p['supportsHttpsTrafficOnly'] and p['minimumTlsVersion'] == 'TLS1_2'
    assert not p['allowBlobPublicAccess'] and p['allowSharedKeyAccess']
    assert p['networkAcls'] == {'defaultAction': 'Deny', 'bypass': 'None', 'ipRules': [], 'virtualNetworkRules': [], 'resourceAccessRules': []}
    share = by_type['Microsoft.Storage/storageAccounts/fileServices/shares'][0]
    assert share['properties'] == {'enabledProtocols': 'SMB', 'shareQuota': "[parameters('journalQuotaGiB')]"}
    assert by_type['Microsoft.Authorization/locks'][0]['properties']['level'] == 'CanNotDelete'
    checks.append('Dedicated SMB share uses private-only account access, no network bypass, HTTPS and TLS requirements, bounded quota, and account deletion protection')
    endpoint = by_type['Microsoft.Network/privateEndpoints'][0]['properties']
    assert endpoint['privateLinkServiceConnections'][0]['properties']['groupIds'] == ['file']
    assert endpoint['ipConfigurations'] == [{'name': 'file', 'properties': {'groupId': 'file', 'memberName': 'file', 'privateIPAddress': "[parameters('fileIp')]"}}]
    file_group = next(group for group in groups if 'preparation-journal' in group['name'])
    file_rules = file_group['properties']['securityRules']
    assert len(file_rules) == 3
    assert file_rules[0]['properties']['sourceAddressPrefix'] == "[parameters('sourceSubnetPrefix')]" and file_rules[0]['properties']['destinationAddressPrefix'] == "[parameters('fileIp')]" and file_rules[0]['properties']['destinationPortRange'] == '445'
    assert file_rules[1]['properties']['access'] == file_rules[2]['properties']['access'] == 'Deny'
    link = by_type['Microsoft.Network/privateDnsZones/virtualNetworkLinks'][0]
    assert not link['properties']['registrationEnabled'] and link['properties']['resolutionPolicy'] == 'Default'
    record = by_type['Microsoft.Network/privateDnsZones/A'][0]
    assert record['properties'] == {'ttl': 60, 'aRecords': [{'ipv4Address': "[parameters('fileIp')]"}]}
    checks.append('Journal-only approved endpoint, fixed private address, source-only SMB rules, and no-fallback controller DNS link form the declared storage path')
    document = json.dumps(template)
    assert 'listKeys' not in document and 'getSecret' not in document and 'amazonaws' not in document and 'docker.sock' not in document
    assert set(template['outputs']) == {'networkBinding', 'sourceSubnetId', 'journalHostBinding'}
    assert set(template['outputs']['networkBinding']['value']) == {'sourceSecurityGroup', 'natGateway', 'natPublicIp', 'databaseIp', 'storageAccount', 'fileEndpoint', 'fileInterface', 'fileSubnet', 'fileSubnetPrefix', 'fileSecurityGroup', 'fileIp', 'fileZone'}
    checks.append('Network bindings contain resource identifiers and addresses only, with no storage key retrieval, app dispatch, or AWS dependency')
    return checks
