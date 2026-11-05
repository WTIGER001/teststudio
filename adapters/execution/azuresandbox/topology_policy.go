package azuresandbox

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

const sandboxFirewallGroup = "sandbox-deny"

func sandboxFirewallRules() n.FirewallPolicyRuleCollectionGroup {
	return n.FirewallPolicyRuleCollectionGroup{Properties: &n.FirewallPolicyRuleCollectionGroupProperties{
		Priority: to.Ptr[int32](400), RuleCollections: []n.FirewallPolicyRuleCollectionClassification{
			&n.FirewallPolicyFilterRuleCollection{Name: to.Ptr("deny-egress"), Priority: to.Ptr[int32](100), RuleCollectionType: to.Ptr(n.FirewallPolicyRuleCollectionTypeFirewallPolicyFilterRuleCollection),
				Action: &n.FirewallPolicyFilterRuleCollectionAction{Type: to.Ptr(n.FirewallPolicyFilterRuleCollectionActionTypeDeny)},
				Rules:  []n.FirewallPolicyRuleClassification{&n.Rule{Name: to.Ptr("deny-egress"), RuleType: to.Ptr(n.FirewallPolicyRuleTypeNetworkRule), SourceAddresses: []*string{to.Ptr("*")}, DestinationAddresses: []*string{to.Ptr("*")}, DestinationPorts: []*string{to.Ptr("*")}, IPProtocols: []*n.FirewallPolicyRuleNetworkProtocol{to.Ptr(n.FirewallPolicyRuleNetworkProtocolAny)}}}},
		}}}
}

func (s Settings) TopologyDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	p := s.Placement
	t := p.Network.Topology
	return map[string]any{
		t.Firewall: p.firewallBinding().FirewallDefinition(s.Worker.Region),
		t.Policy:   p.firewallBinding().PolicyDefinition(s.Worker.Region),
		t.Policy + "/ruleCollectionGroups/" + sandboxFirewallGroup: sandboxFirewallRules(),
		t.PublicIP:         map[string]any{"location": s.Worker.Region, "sku": map[string]string{"name": "Standard", "tier": "Regional"}, "properties": map[string]any{"publicIPAllocationMethod": "Static", "publicIPAddressVersion": "IPv4"}},
		p.firewallSubnet(): map[string]any{"properties": map[string]any{"addressPrefix": t.FirewallPrefix, "defaultOutboundAccess": false}},
	}, nil
}
