package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func applicationRule(name, source string, domains []string, deny bool) *n.ApplicationRule {
	protocols := []*n.FirewallPolicyRuleApplicationProtocol{{ProtocolType: to.Ptr(n.FirewallPolicyRuleApplicationProtocolTypeHTTPS), Port: to.Ptr[int32](443)}}
	if deny {
		protocols = append(protocols, &n.FirewallPolicyRuleApplicationProtocol{ProtocolType: to.Ptr(n.FirewallPolicyRuleApplicationProtocolTypeHTTP), Port: to.Ptr[int32](80)})
	}
	rule := &n.ApplicationRule{Name: to.Ptr(name), RuleType: to.Ptr(n.FirewallPolicyRuleTypeApplicationRule), SourceAddresses: []*string{to.Ptr(source)}, Protocols: protocols, TerminateTLS: to.Ptr(false)}
	for _, domain := range domains {
		rule.TargetFqdns = append(rule.TargetFqdns, to.Ptr(domain))
	}
	return rule
}

func (s FirewallSettings) RuleGroupDefinition(c Config) (n.FirewallPolicyRuleCollectionGroup, error) {
	canonical, err := s.canonical(c)
	if err != nil {
		return n.FirewallPolicyRuleCollectionGroup{}, err
	}
	allow := &n.FirewallPolicyFilterRuleCollection{Name: to.Ptr("worker-platform"), RuleCollectionType: to.Ptr(n.FirewallPolicyRuleCollectionTypeFirewallPolicyFilterRuleCollection), Priority: to.Ptr[int32](100), Action: &n.FirewallPolicyFilterRuleCollectionAction{Type: to.Ptr(n.FirewallPolicyFilterRuleCollectionActionTypeAllow)}, Rules: []n.FirewallPolicyRuleClassification{applicationRule("platform-https", canonical.Perimeter.Topology.WorkerSubnetPrefix, platformDomains(), false)}}
	deny := &n.FirewallPolicyFilterRuleCollection{Name: to.Ptr("worker-deny-other-applications"), RuleCollectionType: to.Ptr(n.FirewallPolicyRuleCollectionTypeFirewallPolicyFilterRuleCollection), Priority: to.Ptr[int32](65000), Action: &n.FirewallPolicyFilterRuleCollectionAction{Type: to.Ptr(n.FirewallPolicyFilterRuleCollectionActionTypeDeny)}, Rules: []n.FirewallPolicyRuleClassification{applicationRule("deny-other-applications", "*", []string{"*"}, true)}}
	return n.FirewallPolicyRuleCollectionGroup{Properties: &n.FirewallPolicyRuleCollectionGroupProperties{Priority: to.Ptr[int32](400), RuleCollections: []n.FirewallPolicyRuleCollectionClassification{allow, deny}}}, nil
}

func (s FirewallSettings) PolicyDefinition(c Config) (n.FirewallPolicy, error) {
	if _, err := s.canonical(c); err != nil {
		return n.FirewallPolicy{}, err
	}
	return n.FirewallPolicy{Location: to.Ptr(c.Region), Properties: &n.FirewallPolicyPropertiesFormat{SKU: &n.FirewallPolicySKU{Tier: to.Ptr(n.FirewallPolicySKUTierStandard)}, ThreatIntelMode: to.Ptr(n.AzureFirewallThreatIntelModeDeny), DNSSettings: &n.DNSSettings{EnableProxy: to.Ptr(false), RequireProxyForNetworkRules: to.Ptr(false)}, Snat: &n.FirewallPolicySNAT{PrivateRanges: []*string{to.Ptr("IANAPrivateRanges")}, AutoLearnPrivateRanges: to.Ptr(n.AutoLearnPrivateRangesModeDisabled)}}}, nil
}

func (s FirewallSettings) FirewallDefinition(c Config) (n.AzureFirewall, error) {
	if _, err := s.canonical(c); err != nil {
		return n.AzureFirewall{}, err
	}
	return n.AzureFirewall{Location: to.Ptr(c.Region), Properties: &n.AzureFirewallPropertiesFormat{SKU: &n.AzureFirewallSKU{Name: to.Ptr(n.AzureFirewallSKUNameAZFWVnet), Tier: to.Ptr(n.AzureFirewallSKUTierStandard)}, ThreatIntelMode: to.Ptr(n.AzureFirewallThreatIntelModeDeny), FirewallPolicy: &n.SubResource{ID: to.Ptr(s.Policy)}, IPConfigurations: []*n.AzureFirewallIPConfiguration{{Name: to.Ptr("egress"), Properties: &n.AzureFirewallIPConfigurationPropertiesFormat{Subnet: &n.SubResource{ID: to.Ptr(s.subnet())}, PublicIPAddress: &n.SubResource{ID: to.Ptr(s.PublicIP)}}}}}}, nil
}
