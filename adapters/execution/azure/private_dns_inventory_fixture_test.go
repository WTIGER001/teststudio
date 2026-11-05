package azureworker

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

type privateDNSInventoryFixture struct {
	t        *testing.T
	mu       sync.Mutex
	worker   Config
	zones    []*p.PrivateZone
	links    map[string][]*p.VirtualNetworkLink
	selected *relayPrivateDNSFixture
	root     string
	reads    int
	pages    int
	mode     string
}

func privateDNSInventoryTest(t *testing.T) (Config, RelayPrivateDNSSettings, *privateDNSInventoryFixture, *PrivateDNSInventoryGuard) {
	t.Helper()
	c, s, previous, prior := relayPrivateDNSTest(t)
	f := &privateDNSInventoryFixture{t: t, worker: c, selected: previous, links: map[string][]*p.VirtualNetworkLink{}, root: "/subscriptions/" + c.Subscription + "/providers/Microsoft.Network/privateDnsZones"}
	worker := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS
	for _, binding := range []privateZoneBinding{worker.binding(c, worker.ExchangeZone), worker.binding(c, worker.RegistryZone), s.binding(c)} {
		fixture := privateZoneFixture(t, []privateZoneBinding{binding})
		f.zones = append(f.zones, to.Ptr(fixture.zones[binding.ID]))
		for _, link := range fixture.links[binding.ID] {
			f.links[binding.ID] = append(f.links[binding.ID], to.Ptr(link))
		}
	}
	reader, err := NewSDKPrivateDNSInventoryReader(c, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewPrivateDNSInventoryGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *privateDNSInventoryFixture) addZone(id, network string) {
	_, name := resourceParts(id)
	count := int64(0)
	if network != "" {
		count = 1
		f.links[id] = []*p.VirtualNetworkLink{{ID: to.Ptr(id + "/virtualNetworkLinks/other-network"), Name: to.Ptr("other-network"), Type: to.Ptr("Microsoft.Network/privateDnsZones/virtualNetworkLinks"), Location: to.Ptr("global"), Properties: &p.VirtualNetworkLinkProperties{VirtualNetwork: &p.SubResource{ID: to.Ptr(network)}, RegistrationEnabled: to.Ptr(false), ProvisioningState: to.Ptr(p.ProvisioningStateSucceeded), VirtualNetworkLinkState: to.Ptr(p.VirtualNetworkLinkStateCompleted)}}}
	} else {
		f.links[id] = []*p.VirtualNetworkLink{}
	}
	f.zones = append(f.zones, &p.PrivateZone{ID: to.Ptr(id), Name: to.Ptr(name), Type: to.Ptr("Microsoft.Network/privateDnsZones"), Location: to.Ptr("global"), Properties: &p.PrivateZoneProperties{ProvisioningState: to.Ptr(p.ProvisioningStateSucceeded), NumberOfVirtualNetworkLinks: to.Ptr(count), NumberOfVirtualNetworkLinksWithRegistration: to.Ptr[int64](0)}})
}

func (f *privateDNSInventoryFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2024-06-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("private DNS inventory widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-dns-inventory-canary"}}), nil
	}
	if r.URL.Path == f.root {
		f.pages++
		page := map[string]any{"value": f.zones}
		switch f.mode {
		case "paged":
			if r.URL.Query().Get("$skipToken") == "" {
				page["value"] = f.zones[:1]
				page["nextLink"] = "https://management.azure.com" + f.root + "?api-version=2024-06-01&%24skipToken=one"
			} else {
				page["value"] = f.zones[1:]
			}
		case "foreign-page":
			page["nextLink"] = "https://foreign.invalid/private"
		case "missing-value":
			delete(page, "value")
		case "endless":
			page["value"] = []any{}
			page["nextLink"] = "https://management.azure.com" + f.root + "?api-version=2024-06-01&%24skipToken=" + strconv.Itoa(f.pages)
		}
		return deadlineResponse(r, 200, page), nil
	}
	id := strings.TrimSuffix(r.URL.Path, "/virtualNetworkLinks")
	links, ok := f.links[id]
	if !ok || id == r.URL.Path {
		f.t.Error("unlisted private zone or unrelated resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	page := map[string]any{"value": links}
	if f.mode == "missing-links" {
		delete(page, "value")
	}
	if f.mode == "foreign-link-page" {
		page["nextLink"] = "https://management.azure.com" + f.root + "?api-version=2024-06-01&%24skipToken=one"
	}
	return deadlineResponse(r, 200, page), nil
}
