package azureworker

import (
	"slices"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePerimeterSettingsAndDefinitionsFreezePrivateAddresses(t *testing.T) {
	c, s, _, f, g := perimeterTest(t)
	cases := map[string]func(*PerimeterSettings){
		"public-firewall":      func(s *PerimeterSettings) { s.FirewallIP = "8.8.8.8" },
		"worker-firewall":      func(s *PerimeterSettings) { s.FirewallIP = "10.20.0.4" },
		"foreign-firewall":     func(s *PerimeterSettings) { s.FirewallIP = "10.21.1.4" },
		"public-endpoint":      func(s *PerimeterSettings) { s.PrivateEndpointIPs[0] = "8.8.8.8" },
		"worker-endpoint":      func(s *PerimeterSettings) { s.PrivateEndpointIPs[0] = "10.20.0.4" },
		"duplicate-endpoint":   func(s *PerimeterSettings) { s.PrivateEndpointIPs[0] = s.PrivateEndpointIPs[1] },
		"firewall-as-endpoint": func(s *PerimeterSettings) { s.PrivateEndpointIPs[0] = s.FirewallIP },
		"too-few-endpoints":    func(s *PerimeterSettings) { s.PrivateEndpointIPs = s.PrivateEndpointIPs[:1] },
		"too-many-endpoints":   func(s *PerimeterSettings) { s.PrivateEndpointIPs = make([]string, 9) },
		"ipv6":                 func(s *PerimeterSettings) { s.FirewallIP = "fd00::4" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := s
			changed.PrivateEndpointIPs = slices.Clone(s.PrivateEndpointIPs)
			mutate(&changed)
			_, err := NewPerimeterGuard(c, changed, g.topology, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	group, err := s.SecurityGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	group.Properties.SecurityRules[0].Properties.SourceAddressPrefix = nil
	s.PrivateEndpointIPs[0] = "8.8.8.8"
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("caller mutation changed the saved guard", err)
	}
	addresses := f.rule("worker-private-endpoints").Properties.DestinationAddressPrefixes
	slices.Reverse(addresses)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("address order changed set semantics", err)
	}
	addresses[0] = addresses[1]
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
}
