package azureworker

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type relayDNSFixture struct {
	*dnsFixture
	firewall *firewallFixture
}

func relayDNSTest(t *testing.T) (Config, RelayDNSSettings, *relayDNSFixture, *RelayDNSGuard) {
	t.Helper()
	c, firewall, previous, _, prior := relayFirewallTest(t)
	s := RelayDNSSettings{Firewall: firewall, Policy: c.resourceID("Microsoft.Network", "dnsResolverPolicies", "relay-dns"), AllowList: c.resourceID("Microsoft.Network", "dnsResolverDomainLists", "relay-dns-allow"), BlockList: c.resourceID("Microsoft.Network", "dnsResolverDomainLists", "relay-dns-block")}
	f := &relayDNSFixture{dnsFixture: dnsBindingFixture(t, c, s.binding(c)), firewall: previous}
	f.controllerVNet = firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerVNet
	reader, err := NewSDKRelayDNSReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewRelayDNSGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func TestAzureRelayDNSBindsFreshVNetAssociationWithoutParentRereads(t *testing.T) {
	c, s, f, g := relayDNSTest(t)
	s.Firewall.Perimeter.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*6 || f.firewall.reads != i*5 {
			t.Fatal("relay DNS rejected or repeated parent reads", err, f.reads, f.firewall.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("DNS observation became complete network admission")
	}
	f.attached[0].ID = to.Ptr("other-policy-canary")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 18 {
		t.Fatal("changed association reused cached success")
	}
	f.firewall.allow().SourceAddresses[0] = to.Ptr("*")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 18 {
		t.Fatal("unverified firewall reached DNS reads")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 18 {
		t.Fatal("foreign profile reached DNS reads")
	}
}

func TestAzureRelayDNSRejectsDomainRuleAndAssociationBypasses(t *testing.T) {
	for name, mutate := range map[string]func(*relayDNSFixture){
		"root-allow": func(f *relayDNSFixture) { f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr(".")) },
		"backend-name": func(f *relayDNSFixture) {
			f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr("backend.azurecontainerapps.io"))
		},
		"registry-wildcard": func(f *relayDNSFixture) {
			f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr("*.azurecr.io"))
		},
		"other-region": func(f *relayDNSFixture) {
			f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr("other.otherregion.data.azurecr.io"))
		},
		"bulk-url": func(f *relayDNSFixture) {
			f.allow.Properties.DomainsURL = to.Ptr("https://foreign.invalid/private-canary")
		},
		"disabled-block": func(f *relayDNSFixture) {
			f.rules[1].Properties.DNSSecurityRuleState = to.Ptr(d.DNSSecurityRuleStateDisabled)
		},
		"wrong-order": func(f *relayDNSFixture) { f.rules[1].Properties.Priority = to.Ptr[int32](99) },
		"worker-rule": func(f *relayDNSFixture) { f.rules[0].Name = to.Ptr("worker-allow-approved") },
		"other-list": func(f *relayDNSFixture) {
			f.rules[0].Properties.DNSResolverDomainLists[0].ID = to.Ptr("other-list-canary")
		},
		"extra-rule":      func(f *relayDNSFixture) { f.rules = append(f.rules, clone(f.rules[0])) },
		"controller-link": func(f *relayDNSFixture) { f.links[0].Properties.VirtualNetwork.ID = to.Ptr(f.controllerVNet) },
		"shared-link":     func(f *relayDNSFixture) { f.links = append(f.links, clone(f.links[0])) },
		"shared-policy": func(f *relayDNSFixture) {
			f.attached = append(f.attached, d.SubResource{ID: to.Ptr("other-policy-canary")})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := relayDNSTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureRelayDNSSDKPreservesUncertaintyAndReadActionScope(t *testing.T) {
	c, s, f, g := relayDNSTest(t)
	reader := g.reader.(*SDKDNSReader)
	binding := s.binding(c)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) {
			r.URL.Path = s.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Policy
		},
		func(r *http.Request) { r.Method = "POST"; r.URL.Path = f.controllerVNet + "/listDnsResolverPolicies" },
		func(r *http.Request) { r.Method = "GET"; r.URL.Path = binding.vnetPolicyPath() },
		func(r *http.Request) { r.Method = "POST"; r.URL.Path = binding.vnetPolicyPath(); r.ContentLength = 1 },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$skipToken=unselected" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Policy+"?api-version=2025-05-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.VNetPolicies(t.Context(), f.controllerVNet)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected DNS request reached transport")
	}
	f.mode = "paged"
	if err := g.Check(t.Context(), c); err != nil || f.reads != 7 || f.rulePages != 2 {
		t.Fatal("bounded complete DNS paging rejected", err, f.reads, f.rulePages)
	}
	for _, mode := range []string{"unavailable", "foreign-page", "endless", "incomplete-vnet-inventory"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "foreign-page" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), c), want)
	}
	f.mode = ""
	f.attached = nil
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayDNSSettingsFreezeAuthorityAndOmitPinnedBackendLookup(t *testing.T) {
	c, s, _, g := relayDNSTest(t)
	worker := s.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS
	for _, mutate := range []func(*RelayDNSSettings){
		func(s *RelayDNSSettings) { s.Policy = worker.Policy },
		func(s *RelayDNSSettings) { s.AllowList = worker.AllowList },
		func(s *RelayDNSSettings) { s.BlockList = worker.BlockList },
		func(s *RelayDNSSettings) { s.AllowList = worker.BlockList },
		func(s *RelayDNSSettings) { s.BlockList = s.AllowList },
		func(s *RelayDNSSettings) { s.Policy = s.Firewall.Policy },
	} {
		changed := s
		mutate(&changed)
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.Firewall.Policy = c.resourceID("Microsoft.Network", "firewallPolicies", "other-relay-policy")
	_, err := NewRelayDNSGuard(c, changed, g.firewall, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKRelayDNSReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	domains := s.ApprovedDomains(c)
	exchange, _ := url.Parse(c.ExchangeOrigin)
	backend, _ := url.Parse(s.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.BackendOrigin)
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	if len(domains) != 13 || slices.Contains(domains, exchange.Hostname()) || slices.Contains(domains, backend.Hostname()) || !slices.Contains(domains, name+"."+worker.RegistryDataRegion+".data.azurecr.io") {
		t.Fatal("relay DNS acquired backend, exchange or unrelated registry names", domains)
	}
	first, _ := s.DomainListDefinition(c, true)
	first.Properties.Domains[0] = to.Ptr(".")
	domains[0] = "."
	second, _ := s.DomainListDefinition(c, true)
	if value(second.Properties.Domains[0]) == "." {
		t.Fatal("caller mutation changed a fresh DNS definition")
	}
	block, _ := s.DomainListDefinition(c, false)
	defaultRule, _ := s.RuleDefinition(c, false)
	if len(block.Properties.Domains) != 1 || value(block.Properties.Domains[0]) != "." || value(defaultRule.Properties.Action.ActionType) != d.ActionTypeBlock || value(defaultRule.Properties.Priority) != 65000 {
		t.Fatal("relay default block is missing")
	}
}
