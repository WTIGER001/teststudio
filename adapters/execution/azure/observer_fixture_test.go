package azureworker

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

type observerFixture struct {
	t                *testing.T
	mu               sync.Mutex
	routes           map[string]policy.Transporter
	bootstrapReads   int
	reads            int
	closed           int
	unavailable      bool
	credential       *observerCredential
	factoryClients   []string
	identity         *readIdentityFixture
	bootstrapReaders int
	denyPath         string
	denies           []a.DenyAssignment
	workloads        *workloadInventoryFixture
}

type observerCredential struct {
	reads atomic.Int64
	fail  atomic.Bool
}

func (c *observerCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.reads.Add(1)
	if ctx.Err() != nil {
		return azcore.AccessToken{}, ctx.Err()
	}
	if c.fail.Load() {
		return azcore.AccessToken{}, errors.New("private-observer-token-canary")
	}
	token, err := (fixtureCredential{}).GetToken(ctx, options)
	token.Token = "observer-credential-canary"
	return token, err
}

func observerTest(t *testing.T) (Config, ReadIdentitySettings, *observerFixture, *ReadIdentityGuard) {
	t.Helper()
	c, s, identity, g := readIdentityTest(t)
	f := &observerFixture{t: t, routes: map[string]policy.Transporter{}, identity: identity, credential: &observerCredential{}}
	f.denyPath = observerScope(c) + "/providers/Microsoft.Authorization/denyAssignments"
	for _, assignment := range identity.protection.assignments {
		f.denies = append(f.denies, assignment)
	}
	protection := g.protection
	inventory := protection.inventory
	relayZones := inventory.selected
	relayRegistry := relayZones.endpoint
	relayDNS := relayRegistry.dns
	relayFirewall := relayDNS.firewall
	relayPerimeter := relayFirewall.perimeter
	relayTopology := relayPerimeter.topology
	backendEndpoint := relayTopology.endpoint
	backend := backendEndpoint.backend
	workloads := backend.inventory
	f.workloads = workloads.reader.(*SDKWorkloadInventoryReader).http.base.(*workloadInventoryFixture)
	exchange := workloads.exchange
	registry := exchange.registry
	endpoints := registry.endpoints
	zones := endpoints.dns
	dns := zones.dns
	firewall := dns.firewall
	perimeter := firewall.perimeter
	topology := perimeter.topology
	for _, h := range []*armReadHTTP{
		g.reader.(*SDKReadIdentityReader).http,
		protection.reader.(*SDKNetworkDenyReader).http,
		relayZones.reader.(*SDKPrivateDNSReader).http,
		relayRegistry.reader.(*SDKRelayRegistryEndpointReader).http,
		relayDNS.reader.(*SDKDNSReader).http,
		relayFirewall.reader.(*SDKFirewallReader).http,
		relayPerimeter.reader.(*SDKPerimeterReader).http,
		relayTopology.reader.(*SDKRelayTopologyReader).http,
		backendEndpoint.reader.(*SDKBackendEndpointReader).http,
		backend.reader.(*SDKBackendReader).http,
		workloads.reader.(*SDKWorkloadInventoryReader).http,
		exchange.reader.(*SDKExchangeReader).http,
		registry.reader.(*SDKRegistryReader).http,
		endpoints.reader.(*SDKEndpointReader).http,
		zones.reader.(*SDKPrivateDNSReader).http,
		dns.reader.(*SDKDNSReader).http,
		firewall.reader.(*SDKFirewallReader).http,
		perimeter.reader.(*SDKPerimeterReader).http,
		topology.reader.(*SDKTopologyReader).http,
	} {
		h.base = f.capture(h.base)
		f.bootstrapReaders++
	}
	reader := inventory.reader.(*SDKPrivateDNSInventoryReader)
	reader.base = f.capture(reader.base)
	f.bootstrapReaders++
	g.credential = func(id string) (azcore.TokenCredential, error) {
		f.factoryClients = append(f.factoryClients, id)
		return f.credential, nil
	}
	return c, s, f, g
}

func observerRoute(r *http.Request) string {
	return r.Method + " " + r.URL.Path + "?api-version=" + r.URL.Query().Get("api-version")
}

func (f *observerFixture) capture(base policy.Transporter) policy.Transporter {
	return transportFunc(func(r *http.Request) (*http.Response, error) {
		f.mu.Lock()
		f.bootstrapReads++
		f.routes[observerRoute(r)] = base
		f.mu.Unlock()
		return base.Do(r)
	})
}

func (f *observerFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.reads++
	base, unavailable := f.routes[observerRoute(r)], f.unavailable
	f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer observer-credential-canary" {
		f.t.Error("bound inspection used another credential")
	}
	if base == nil && r.URL.Path != f.denyPath {
		f.t.Error("bound inspection reached an unobserved bootstrap path")
		return deadlineResponse(r, 404, nil), nil
	}
	if unavailable {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-observer-transport-canary"}}), nil
	}
	if r.URL.Path == f.denyPath {
		if r.Method != "GET" || r.URL.RawQuery != "api-version="+networkDenyVersion {
			f.t.Error("observer deny inventory widened its request")
		}
		return deadlineResponse(r, 200, map[string]any{"value": f.denies}), nil
	}
	copy := r.Clone(r.Context())
	copy.Header.Set("Authorization", "Bearer local-credential-canary")
	return base.Do(copy)
}

func (f *observerFixture) CloseIdleConnections() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
}
