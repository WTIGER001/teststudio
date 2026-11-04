package azureworker

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureDNSSettingsAndDefinitionsRetainExplicitAuthority(t *testing.T) {
	c, s, _, g := dnsTest(t)
	cases := map[string]func(*DNSSettings){
		"missing-data-region": func(s *DNSSettings) { s.RegistryDataRegion = "" },
		"invalid-data-region": func(s *DNSSettings) { s.RegistryDataRegion = "west us" },
		"same-domain-list":    func(s *DNSSettings) { s.BlockList = s.AllowList },
		"wrong-policy-kind":   func(s *DNSSettings) { s.Policy = s.Firewall.Policy },
		"foreign-subscription": func(s *DNSSettings) {
			s.Policy = strings.Replace(s.Policy, "11111111-2222-3333-4444-555555555555", "99999999-2222-3333-4444-555555555555", 1)
		},
		"bulk-list-address": func(s *DNSSettings) { s.AllowList += "/bulk" },
		"credential-query":  func(s *DNSSettings) { s.BlockList += "?secret=private" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := s
			mutate(&changed)
			_, err := NewDNSGuard(c, changed, g.firewall, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	list, err := s.DomainListDefinition(c, true)
	if err != nil {
		t.Fatal(err)
	}
	list.Properties.Domains[0] = to.Ptr(".")
	rule, _ := s.RuleDefinition(c, false)
	rule.Properties.Action.ActionType = to.Ptr(d.ActionTypeAllow)
	domains := s.ApprovedDomains(c)
	domains[0] = "."
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("caller changed saved declarations", err)
	}
	invalid := c
	invalid.ExchangeOrigin = "https://%"
	if s.ApprovedDomains(invalid) != nil {
		t.Fatal("invalid origin produced approved domains")
	}
}

func TestAzureDNSDeclaredDecisionsPreserveHierarchyAndDefaultBlock(t *testing.T) {
	c, s, _, _ := dnsTest(t)
	exchange, _ := url.Parse(c.ExchangeOrigin)
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	cases := []struct {
		domain string
		allow  bool
	}{
		{exchange.Hostname(), true}, {"child." + exchange.Hostname(), true},
		{c.Registry, true}, {name + "." + s.RegistryDataRegion + ".data.azurecr.io", true}, {name + ".privatelink.azurecr.io", true}, {name + "." + s.RegistryDataRegion + ".data.privatelink.azurecr.io", true},
		{"mcr.microsoft.com", true}, {"region.data.mcr.microsoft.com", true}, {"login.microsoftonline.com", true}, {"tenant.identity.azure.net", true},
		{"foreign.invalid", false}, {"github.com", false}, {"registry.npmjs.org", false}, {"foreign.blob.core.windows.net", false}, {"otherregistry.azurecr.io", false}, {"otherregistry.privatelink.azurecr.io", false}, {name + ".otherregion.data.azurecr.io", false}, {"dc.services.visualstudio.com", false}, {name + "." + c.Region + ".data.azurecr.io", false},
	}
	for _, test := range cases {
		t.Run(test.domain, func(t *testing.T) {
			allowed := false
			for _, domain := range s.ApprovedDomains(c) {
				base := strings.TrimPrefix(domain, "*.")
				if test.domain == base || strings.HasSuffix(test.domain, "."+base) {
					allowed = true
				}
			}
			if allowed != test.allow {
				t.Fatal("declaration changed native hierarchy or the default block", allowed, test.allow)
			}
		})
	}
	block, _ := s.DomainListDefinition(c, false)
	defaultRule, _ := s.RuleDefinition(c, false)
	allowRule, _ := s.RuleDefinition(c, true)
	if len(block.Properties.Domains) != 1 || value(block.Properties.Domains[0]) != "." || value(defaultRule.Properties.Action.ActionType) != d.ActionTypeBlock || value(defaultRule.Properties.DNSSecurityRuleState) != d.DNSSecurityRuleStateEnabled || value(defaultRule.Properties.Priority) <= value(allowRule.Properties.Priority) {
		t.Fatal("default block declaration is missing or ordered incorrectly")
	}
}
