package azuresandbox

import (
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
)

func (h HostSettings) networkRules() []any {
	ips := []string{h.Registry.LoginIP, h.Registry.DataIP}
	slices.Sort(ips)
	rules := azureworker.PlatformNetworkRules("controller", h.SubnetPrefix, ips)
	result := []any{}
	for _, rule := range rules[:len(rules)-1] {
		result = append(result, objectMap(rule))
	}
	result = append(result, networkRule("controller-firewall-https", "Outbound", "Allow", "Tcp", h.SubnetPrefix, "Internet", "443", 310), objectMap(rules[len(rules)-1]))
	return result
}

func (h HostSettings) firewallRules(s Settings) n.FirewallPolicyRuleCollectionGroup {
	group := azureworker.PlatformFirewallRules("controller", h.SubnetPrefix)
	rule := group.Properties.RuleCollections[0].(*n.FirewallPolicyFilterRuleCollection).Rules[0].(*n.ApplicationRule)
	rule.TargetFqdns = nil
	for _, domain := range h.dependencyDomains(s) {
		rule.TargetFqdns = append(rule.TargetFqdns, to.Ptr(domain))
	}
	return group
}

func (s Settings) HostNetworkDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	h, r := s.Host, s.Host.Network
	result := h.resolverBinding(s).Definitions(s.Worker.Region)
	result[r.SecurityGroup] = map[string]any{"location": s.Worker.Region, "properties": map[string]any{"flushConnection": true, "securityRules": h.networkRules()}}
	result[r.RouteTable] = map[string]any{"location": s.Worker.Region, "properties": map[string]any{"disableBgpRoutePropagation": true, "routes": []any{map[string]any{"name": "controller-egress", "properties": map[string]any{"addressPrefix": "0.0.0.0/0", "nextHopType": "VirtualAppliance", "nextHopIpAddress": r.FirewallIP}}}}}
	result[h.Subnet] = map[string]any{"properties": map[string]any{"addressPrefix": h.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": r.SecurityGroup}, "routeTable": map[string]string{"id": r.RouteTable}, "delegations": []any{map[string]any{"name": "controller-service", "properties": map[string]string{"serviceName": "Microsoft.App/environments"}}}}}
	binding := h.firewallBinding()
	result[binding.Subnet] = map[string]any{"properties": map[string]any{"addressPrefix": r.Topology.FirewallPrefix, "defaultOutboundAccess": false}}
	result[r.Topology.Firewall] = binding.FirewallDefinition(s.Worker.Region)
	result[r.Topology.Policy] = binding.PolicyDefinition(s.Worker.Region)
	result[r.Topology.PublicIP] = map[string]any{"location": s.Worker.Region, "sku": map[string]string{"name": "Standard", "tier": "Regional"}, "properties": map[string]string{"publicIPAllocationMethod": "Static", "publicIPAddressVersion": "IPv4"}}
	result[r.Topology.Policy+"/ruleCollectionGroups/controller-platform"] = h.firewallRules(s)
	return result, nil
}
