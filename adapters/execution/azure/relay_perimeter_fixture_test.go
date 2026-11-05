package azureworker

import (
	"net/http"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type relayPerimeterFixture struct {
	t        *testing.T
	mu       sync.Mutex
	settings RelayPerimeterSettings
	topology *relayTopologyFixture
	security n.SecurityGroup
	routes   n.RouteTable
	reads    int
	mode     string
}

func relayPerimeterTest(t *testing.T) (Config, RelayPerimeterSettings, *relayPerimeterFixture, *RelayPerimeterGuard) {
	t.Helper()
	c, topology, previous, prior := relayTopologyTest(t)
	s := RelayPerimeterSettings{Topology: topology, SecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "relay-source"), RouteTable: c.resourceID("Microsoft.Network", "routeTables", "relay-source"), FirewallIP: "10.40.1.4", RegistryEndpointIPs: []string{"10.40.3.5", "10.40.3.4"}}
	subnet := topology.BackendEndpoint.Backend.Exchange.Subnet
	p := previous.endpoint.relay.Properties
	p.DefaultOutboundAccess = to.Ptr(false)
	p.NetworkSecurityGroup = &n.SecurityGroup{ID: to.Ptr(s.SecurityGroup)}
	p.RouteTable = &n.RouteTable{ID: to.Ptr(s.RouteTable)}
	previous.network.Properties.Subnets[0] = clone(&previous.endpoint.relay)
	f := &relayPerimeterFixture{t: t, settings: s, topology: previous}
	f.security, _ = s.SecurityGroupDefinition(c)
	f.security.ID, f.security.Name, f.security.Type = to.Ptr(s.SecurityGroup), to.Ptr("relay-source"), to.Ptr("Microsoft.Network/networkSecurityGroups")
	f.security.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.security.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(subnet)}}
	for _, rule := range f.security.Properties.SecurityRules {
		rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	}
	f.routes, _ = s.RouteTableDefinition(c)
	f.routes.ID, f.routes.Name, f.routes.Type = to.Ptr(s.RouteTable), to.Ptr("relay-source"), to.Ptr("Microsoft.Network/routeTables")
	f.routes.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.routes.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(subnet)}}
	f.routes.Properties.Routes[0].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	reader, err := NewSDKRelayPerimeterReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewRelayPerimeterGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *relayPerimeterFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("relay perimeter reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	var body any
	switch r.URL.Path {
	case f.settings.SecurityGroup:
		body = f.security
	case f.settings.RouteTable:
		body = f.routes
	default:
		f.t.Error("unselected relay perimeter resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-relay-perimeter-canary"}}), nil
	}
	return deadlineResponse(r, 200, body), nil
}
