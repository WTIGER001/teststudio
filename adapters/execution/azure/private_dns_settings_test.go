package azureworker

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePrivateDNSSettingsAndDefinitionsFreezeRoleAddressBindings(t *testing.T) {
	c, s, _, g := privateDNSTest(t)
	cases := map[string]func(*PrivateDNSSettings){
		"parent-zone": func(s *PrivateDNSSettings) {
			s.ExchangeZone = strings.Replace(s.ExchangeZone, "workers.studio.internal.example.test", "studio.internal.example.test", 1)
		},
		"foreign-registry-zone": func(s *PrivateDNSSettings) {
			s.RegistryZone = strings.Replace(s.RegistryZone, "privatelink.azurecr.io", "azurecr.io", 1)
		},
		"foreign-subscription": func(s *PrivateDNSSettings) {
			s.RegistryZone = strings.Replace(s.RegistryZone, "11111111-2222-3333-4444-555555555555", "99999999-2222-3333-4444-555555555555", 1)
		},
		"public-address":         func(s *PrivateDNSSettings) { s.ExchangeIP = "8.8.8.8" },
		"unselected-address":     func(s *PrivateDNSSettings) { s.DataIP = "10.20.2.7" },
		"duplicate-role-address": func(s *PrivateDNSSettings) { s.RegistryIP = s.ExchangeIP },
		"query-in-resource":      func(s *PrivateDNSSettings) { s.RegistryZone += "?secret=private" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := s
			mutate(&changed)
			_, err := NewPrivateDNSGuard(c, changed, g.dns, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	records, err := s.RecordDefinitions(c, s.ExchangeZone)
	if err != nil {
		t.Fatal(err)
	}
	record := records["@"]
	record.Properties.ARecords[0].IPv4Address = to.Ptr("8.8.8.8")
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("caller mutation changed the guard", err)
	}
	origin, _ := url.Parse(c.ExchangeOrigin)
	if !strings.HasSuffix(s.ExchangeZone, "/"+origin.Hostname()) {
		t.Fatal("exchange authority is not at its exact hostname")
	}
	login := strings.TrimSuffix(c.Registry, ".azurecr.io")
	definitions, err := s.RecordDefinitions(c, s.RegistryZone)
	if err != nil || len(definitions) != 2 || value(definitions[login].Properties.ARecords[0].IPv4Address) != s.RegistryIP || value(definitions[login+"."+s.DNS.RegistryDataRegion+".data"].Properties.ARecords[0].IPv4Address) != s.DataIP {
		t.Fatal("registry role, data region or address declaration changed", err)
	}
	exchangeLink, _ := s.LinkDefinition(c, s.ExchangeZone)
	registryLink, _ := s.LinkDefinition(c, s.RegistryZone)
	if exchangeLink.Properties.ResolutionPolicy != nil || value(registryLink.Properties.ResolutionPolicy) != p.ResolutionPolicyDefault || value(registryLink.Properties.RegistrationEnabled) {
		t.Fatal("link definition enabled fallback or registration, or configured an unsupported custom-zone policy")
	}
}
