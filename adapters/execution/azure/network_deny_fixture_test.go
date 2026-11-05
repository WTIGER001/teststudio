package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

type networkDenyFixture struct {
	t           *testing.T
	mu          sync.Mutex
	settings    NetworkDenySettings
	inventory   *privateDNSInventoryFixture
	assignments map[string]a.DenyAssignment
	reads       int
	mode        string
}

func networkDenyTest(t *testing.T) (Config, NetworkDenySettings, *networkDenyFixture, *NetworkDenyGuard) {
	t.Helper()
	c, selected, previous, prior := privateDNSInventoryTest(t)
	exchange := selected.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange
	zones := exchange.Registry.Endpoints.PrivateDNS
	suffix := "/providers/Microsoft.Authorization/denyAssignments/"
	s := NetworkDenySettings{
		PrivateDNS:                   selected,
		WorkerAssignment:             zones.DNS.Firewall.Perimeter.Topology.WorkerVNet + suffix + "aaaaaaaa-1111-2222-3333-444444444444",
		RelayAssignment:              exchange.VirtualNetwork + suffix + "bbbbbbbb-1111-2222-3333-444444444444",
		WorkerExchangeZoneAssignment: zones.ExchangeZone + suffix + "cccccccc-1111-2222-3333-444444444444",
		WorkerRegistryZoneAssignment: zones.RegistryZone + suffix + "dddddddd-1111-2222-3333-444444444444",
		RelayRegistryZoneAssignment:  selected.RegistryZone + suffix + "eeeeeeee-1111-2222-3333-444444444444",
	}
	f := &networkDenyFixture{t: t, settings: s, inventory: previous, assignments: map[string]a.DenyAssignment{}}
	for _, binding := range s.bindings() {
		parts := strings.Split(binding.assignment, "/")
		f.assignments[binding.assignment] = a.DenyAssignment{ID: to.Ptr(binding.assignment), Name: to.Ptr(parts[len(parts)-1]), Type: to.Ptr("Microsoft.Authorization/denyAssignments"), Properties: &a.DenyAssignmentProperties{Scope: to.Ptr(binding.scope), DenyAssignmentEffect: to.Ptr(a.DenyAssignmentEffectEnforced), IsSystemProtected: to.Ptr(true), DoNotApplyToChildScopes: to.Ptr(false), Principals: []*a.DenyAssignmentPrincipal{{ID: to.Ptr(allDenyPrincipals), Type: to.Ptr("SystemDefined")}}, Permissions: []*a.DenyAssignmentPermission{{Actions: []*string{to.Ptr("*")}, NotActions: []*string{to.Ptr("*/read"), to.Ptr("Microsoft.Authorization/locks/delete"), to.Ptr(networkReadAssociationAction)}}}}}
	}
	reader, err := NewSDKNetworkDenyReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewNetworkDenyGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *networkDenyFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version="+networkDenyVersion || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("network deny reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-network-deny-canary"}}), nil
	}
	if f.mode == "missing" {
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	assignment, ok := f.assignments[r.URL.Path]
	if !ok {
		f.t.Error("unselected deny assignment reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	return deadlineResponse(r, 200, assignment), nil
}
