package azureworker

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureFirewallSettingsAndDefinitionsPreserveIsolationBindings(t *testing.T) {
	c, s, _, g := firewallTest(t)
	cases := map[string]func(*FirewallSettings){
		"small-subnet":      func(s *FirewallSettings) { s.SubnetPrefix = "10.20.1.0/27" },
		"worker-overlap":    func(s *FirewallSettings) { s.SubnetPrefix = "10.20.0.0/23" },
		"outside-appliance": func(s *FirewallSettings) { s.SubnetPrefix = "10.20.3.0/26" },
		"endpoint-overlap":  func(s *FirewallSettings) { s.SubnetPrefix = "10.20.0.0/16" },
		"foreign-subscription": func(s *FirewallSettings) {
			s.Firewall = strings.Replace(s.Firewall, "11111111-2222-3333-4444-555555555555", "99999999-2222-3333-4444-555555555555", 1)
		},
		"foreign-policy-kind": func(s *FirewallSettings) { s.Policy = s.Firewall },
		"invalid-public-ip":   func(s *FirewallSettings) { s.PublicIP += "?expand=secrets" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := s
			mutate(&changed)
			_, err := NewFirewallGuard(c, changed, g.perimeter, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	definition, err := s.RuleGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	definition.Properties.RuleCollections[0].(*n.FirewallPolicyFilterRuleCollection).Rules[0].(*n.ApplicationRule).TargetFqdns = []*string{to.Ptr("*")}
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("returned definition changed the guard", err)
	}
}

func TestAzureFirewallApplicationDeclarationsBoundPlatformAccessAndFallback(t *testing.T) {
	c, s, _, _ := firewallTest(t)
	definition, err := s.RuleGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	collections := definition.Properties.RuleCollections
	if len(collections) != 2 {
		t.Fatal("application fallback is missing")
	}
	allow := collections[0].(*n.FirewallPolicyFilterRuleCollection)
	deny := collections[1].(*n.FirewallPolicyFilterRuleCollection)
	if value(allow.Action.Type) != n.FirewallPolicyFilterRuleCollectionActionTypeAllow || value(deny.Action.Type) != n.FirewallPolicyFilterRuleCollectionActionTypeDeny || value(allow.Priority) >= value(deny.Priority) {
		t.Fatal("terminal application decision order is wrong")
	}
	approved := map[string]bool{"mcr.microsoft.com": true, "*.data.mcr.microsoft.com": true, "packages.aks.azure.com": true, "acs-mirror.azureedge.net": true, "*.identity.azure.net": true, "login.microsoftonline.com": true, "*.login.microsoftonline.com": true, "login.microsoft.com": true, "*.login.microsoft.com": true}
	rule := allow.Rules[0].(*n.ApplicationRule)
	if len(rule.SourceAddresses) != 1 || value(rule.SourceAddresses[0]) != s.Perimeter.Topology.WorkerSubnetPrefix || len(rule.Protocols) != 1 || value(rule.Protocols[0].ProtocolType) != n.FirewallPolicyRuleApplicationProtocolTypeHTTPS || value(rule.Protocols[0].Port) != 443 {
		t.Fatal("platform access widened beyond worker HTTPS")
	}
	for _, domain := range rule.TargetFqdns {
		if !approved[value(domain)] {
			t.Fatal("unapproved platform destination", value(domain))
		}
		delete(approved, value(domain))
	}
	if len(approved) != 0 {
		t.Fatal("required platform declarations are missing")
	}
	fallback := deny.Rules[0].(*n.ApplicationRule)
	if len(fallback.SourceAddresses) != 1 || value(fallback.SourceAddresses[0]) != "*" || len(fallback.TargetFqdns) != 1 || value(fallback.TargetFqdns[0]) != "*" {
		t.Fatal("fallback does not override infrastructure allowances")
	}
	ports := map[int32]n.FirewallPolicyRuleApplicationProtocolType{80: n.FirewallPolicyRuleApplicationProtocolTypeHTTP, 443: n.FirewallPolicyRuleApplicationProtocolTypeHTTPS}
	for _, protocol := range fallback.Protocols {
		if ports[value(protocol.Port)] != value(protocol.ProtocolType) {
			t.Fatal("fallback gained an unrelated protocol")
		}
		delete(ports, value(protocol.Port))
	}
	if len(ports) != 0 {
		t.Fatal("fallback lacks an application protocol")
	}
}
