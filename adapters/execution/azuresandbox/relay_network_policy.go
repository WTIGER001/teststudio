package azuresandbox

import (
	"encoding/json"
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
)

func objectMap(value any) map[string]any {
	data, _ := json.Marshal(value)
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}

func (r RelayNetworkSettings) firewallBinding(s Settings) azureworker.FirewallResourceBinding {
	return azureworker.FirewallResourceBinding{Firewall: r.Topology.Firewall, Policy: r.Topology.Policy, PublicIP: r.Topology.PublicIP, Subnet: r.virtualNetwork(s) + "/subnets/AzureFirewallSubnet", SubnetPrefix: r.Topology.FirewallPrefix, Address: r.FirewallIP}
}

func (r RelayNetworkSettings) rules(s Settings) []any {
	addresses := []string{s.Placement.Services.Backend.Address, r.Registry.LoginIP, r.Registry.DataIP}
	slices.Sort(addresses)
	result := []any{}
	for _, rule := range azureworker.PlatformNetworkRules("relay", s.Placement.Services.BackendEndpoint.RelaySubnetPrefix, addresses) {
		result = append(result, objectMap(rule))
	}
	return result
}

func (r RelayNetworkSettings) registryRules(s Settings) []any {
	allow := networkRule("relay-registry-https", "Inbound", "Allow", "Tcp", s.Placement.Services.BackendEndpoint.RelaySubnetPrefix, "", "443", 100)
	props := allow["properties"].(map[string]any)
	delete(props, "destinationAddressPrefix")
	addresses := []string{r.Registry.LoginIP, r.Registry.DataIP}
	slices.Sort(addresses)
	props["destinationAddressPrefixes"] = addresses
	return []any{allow, networkRule("endpoint-deny-in", "Inbound", "Deny", "*", "*", "*", "*", 4096), networkRule("endpoint-deny-out", "Outbound", "Deny", "*", "*", "*", "*", 4096)}
}

func (r RelayNetworkSettings) resolverBinding(s Settings) azureworker.ResolverBinding {
	name := strings.TrimSuffix(s.Worker.Registry, ".azurecr.io")
	domains := append(azureworker.PlatformDomains(), s.Worker.Registry, name+"."+s.Worker.Region+".data.azurecr.io", name+".privatelink.azurecr.io", name+"."+s.Worker.Region+".data.privatelink.azurecr.io")
	slices.Sort(domains)
	return azureworker.ResolverBinding{Policy: r.Resolver.Policy, AllowList: r.Resolver.AllowList, BlockList: r.Resolver.BlockList, VirtualNetwork: r.virtualNetwork(s), Role: "relay", Domains: domains}
}

func (r RelayNetworkSettings) zoneBinding(s Settings) azureworker.PrivateZoneBinding {
	name := strings.TrimSuffix(s.Worker.Registry, ".azurecr.io")
	return azureworker.PrivateZoneBinding{ID: r.Registry.Zone, VirtualNetwork: r.virtualNetwork(s), LinkName: "relay-network", ResolutionPolicy: true, Records: map[string]string{name: r.Registry.LoginIP, name + "." + s.Worker.Region + ".data": r.Registry.DataIP}}
}

func (s Settings) RelayNetworkDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	r, v := s.Placement.Services.RelayNetwork, s.Placement.Services
	result := r.resolverBinding(s).Definitions(s.Worker.Region)
	result[r.SecurityGroup] = map[string]any{"location": s.Worker.Region, "properties": map[string]any{"flushConnection": true, "securityRules": r.rules(s)}}
	result[r.RouteTable] = map[string]any{"location": s.Worker.Region, "properties": map[string]any{"disableBgpRoutePropagation": true, "routes": []any{map[string]any{"name": "relay-egress", "properties": map[string]any{"addressPrefix": "0.0.0.0/0", "nextHopType": "VirtualAppliance", "nextHopIpAddress": r.FirewallIP}}}}}
	result[v.Relay.Subnet] = map[string]any{"properties": map[string]any{"addressPrefix": v.BackendEndpoint.RelaySubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": r.SecurityGroup}, "routeTable": map[string]string{"id": r.RouteTable}, "delegations": []any{map[string]any{"name": "relay-service", "properties": map[string]string{"serviceName": "Microsoft.App/environments"}}}}}
	result[r.Registry.SecurityGroup] = map[string]any{"location": s.Worker.Region, "properties": map[string]any{"flushConnection": true, "securityRules": r.registryRules(s)}}
	result[r.Registry.Subnet] = map[string]any{"properties": map[string]any{"addressPrefix": r.Registry.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": r.Registry.SecurityGroup}, "privateEndpointNetworkPolicies": "Enabled", "privateLinkServiceNetworkPolicies": "Enabled"}}
	result[r.Topology.Firewall] = r.firewallBinding(s).FirewallDefinition(s.Worker.Region)
	result[r.Topology.Policy] = r.firewallBinding(s).PolicyDefinition(s.Worker.Region)
	result[r.Topology.Policy+"/ruleCollectionGroups/relay-platform"] = azureworker.PlatformFirewallRules("relay", v.BackendEndpoint.RelaySubnetPrefix)
	result[r.Topology.PublicIP] = map[string]any{"location": s.Worker.Region, "sku": map[string]string{"name": "Standard", "tier": "Regional"}, "properties": map[string]string{"publicIPAllocationMethod": "Static", "publicIPAddressVersion": "IPv4"}}
	result[r.virtualNetwork(s)+"/subnets/AzureFirewallSubnet"] = map[string]any{"properties": map[string]any{"addressPrefix": r.Topology.FirewallPrefix, "defaultOutboundAccess": false}}
	zone := r.zoneBinding(s)
	result[zone.ID] = map[string]any{"location": "global"}
	result[zone.ID+"/virtualNetworkLinks/"+zone.LinkName] = zone.LinkDefinition()
	for name, record := range zone.RecordDefinitions() {
		result[zone.ID+"/A/"+name] = record
	}
	result[r.Registry.PullAssignment] = map[string]any{"properties": map[string]any{"principalId": v.Relay.Pull.Principal, "principalType": "ServicePrincipal", "scope": s.Placement.Registry, "roleDefinitionId": "/subscriptions/" + s.Worker.Subscription + pullRole}}
	return result, nil
}
