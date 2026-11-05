package azureworker

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePrivateDNSInventoryRefreshesReportedSubscriptionLinks(t *testing.T) {
	c, s, f, g := privateDNSInventoryTest(t)
	for i := 1; i <= 2; i++ {
		if i == 2 {
			f.zones[0].ID = to.Ptr(strings.Replace(value(f.zones[0].ID), "/providers/Microsoft.Network/", "/providers/microsoft.network/", 1))
		}
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*4 || f.selected.reads != i*3 || f.selected.endpoint.network.reads != i {
			t.Fatal("reported zone inventory rejected or parent repeated", err, f.reads, f.selected.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("reported subscription inventory became complete network authority")
	}
	id := c.resourceID("Microsoft.Network", "privateDnsZones", "shadow.internal.example.test")
	relay := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork
	f.addZone(id, relay)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	reads := f.reads
	f.selected.endpoint.nic.Properties.PrivateEndpoint.ID = to.Ptr("other-endpoint-canary")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != reads {
		t.Fatal("unverified parent reached reported inventory")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
}

func TestAzurePrivateDNSInventoryRejectsUnexpectedLinksAndPreservesUnrelatedZones(t *testing.T) {
	for name, mutate := range map[string]func(Config, RelayPrivateDNSSettings, *privateDNSInventoryFixture){
		"worker-shadow": func(c Config, s RelayPrivateDNSSettings, f *privateDNSInventoryFixture) {
			f.addZone(c.resourceID("Microsoft.Network", "privateDnsZones", "shadow.internal.example.test"), s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNet)
		},
		"relay-shadow": func(c Config, s RelayPrivateDNSSettings, f *privateDNSInventoryFixture) {
			f.addZone(c.resourceID("Microsoft.Network", "privateDnsZones", "mcr.microsoft.com"), s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork)
		},
		"duplicate-zone": func(c Config, s RelayPrivateDNSSettings, f *privateDNSInventoryFixture) {
			f.zones = append(f.zones, clone(f.zones[0]))
		},
		"foreign-subscription": func(c Config, s RelayPrivateDNSSettings, f *privateDNSInventoryFixture) {
			f.zones[0].ID = to.Ptr(strings.Replace(value(f.zones[0].ID), c.Subscription, "99999999-2222-3333-4444-555555555555", 1))
		},
		"wrong-parent": func(c Config, s RelayPrivateDNSSettings, f *privateDNSInventoryFixture) {
			id := value(f.zones[0].ID)
			f.links[id][0].ID = to.Ptr(value(f.zones[1].ID) + "/virtualNetworkLinks/worker-network")
		},
		"malformed-network": func(c Config, s RelayPrivateDNSSettings, f *privateDNSInventoryFixture) {
			f.links[value(f.zones[0].ID)][0].Properties.VirtualNetwork.ID = to.Ptr("foreign-network-canary")
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, s, f, g := privateDNSInventoryTest(t)
			mutate(c, s, f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, _, f, g := privateDNSInventoryTest(t)
	f.addZone(c.resourceID("Microsoft.Network", "privateDnsZones", "unrelated.example.test"), c.resourceID("Microsoft.Network", "virtualNetworks", "unrelated-network"))
	f.addZone(c.resourceID("Microsoft.Network", "privateDnsZones", "empty.example.test"), "")
	if err := g.Check(t.Context(), c); err != nil || f.reads != 6 {
		t.Fatal("unrelated or zero-link zones rejected", err, f.reads)
	}
}

func TestAzurePrivateDNSInventoryPreservesIncompleteCoverageAndPaging(t *testing.T) {
	c, _, f, g := privateDNSInventoryTest(t)
	f.mode = "paged"
	if err := g.Check(t.Context(), c); err != nil || f.reads != 5 || f.pages != 2 {
		t.Fatal("complete subscription paging rejected", err, f.reads)
	}
	for _, mode := range []string{"unavailable", "missing-value", "missing-links", "foreign-page", "foreign-link-page", "endless"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "foreign-page" || mode == "foreign-link-page" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), c), want)
	}
	f.mode = ""
	f.zones[0].Properties.NumberOfVirtualNetworkLinks = to.Ptr[int64](2)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.zones[0].Properties.NumberOfVirtualNetworkLinks = to.Ptr[int64](1)
	f.links[value(f.zones[0].ID)][0].Properties.VirtualNetworkLinkState = to.Ptr(p.VirtualNetworkLinkStateInProgress)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzurePrivateDNSInventoryBoundsSDKAuthorityAndConfiguration(t *testing.T) {
	c, s, f, g := privateDNSInventoryTest(t)
	if _, err := NewSDKPrivateDNSInventoryReader(c, nil, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("implicit inventory credential discovery accepted")
	}
	changed := s
	changed.RegistryZone = strings.Replace(s.RegistryZone, "/resourceGroups/relay-private-dns/", "/resourceGroups/other-private-dns/", 1)
	_, err := NewPrivateDNSInventoryGuard(c, changed, g.selected, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	reader := g.reader.(*SDKPrivateDNSInventoryReader)
	h := newARMReadHTTP(c, map[string]string{f.root: "2024-06-01"}, reader.base)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path = s.RegistryZone + "/ALL" },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$filter=approved" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+f.root+"?api-version=2024-06-01", nil)
		mutate(request)
		_, err := h.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected request reached transport")
	}
	budget := &privateDNSInventoryBudget{base: transportFunc(func(r *http.Request) (*http.Response, error) { return deadlineResponse(r, 200, nil), nil })}
	request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+f.root+"?api-version=2024-06-01", nil)
	_, err = budget.Do(request)
	requireDeadlineError(t, err, hostedexecution.ErrUnavailable)
	if privateDNSInventoryID(c, s.RegistryZone+"?secret=private") {
		t.Fatal("query-bearing private zone became read authority")
	}
}
