package azureworker

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type relayTopologyFixture struct {
	t        *testing.T
	settings RelayTopologySettings
	network  n.VirtualNetwork
	endpoint *backendEndpointFixture
	reads    int
	mode     string
}

func relayTopologyTest(t *testing.T) (Config, RelayTopologySettings, *relayTopologyFixture, *RelayTopologyGuard) {
	t.Helper()
	c, endpoint, previous, prior := backendEndpointTest(t)
	s := RelayTopologySettings{BackendEndpoint: endpoint, VirtualNetworkPrefix: "10.40.0.0/16"}
	id := endpoint.Backend.Exchange.VirtualNetwork
	_, name := resourceParts(id)
	f := &relayTopologyFixture{t: t, settings: s, endpoint: previous, network: n.VirtualNetwork{ID: to.Ptr(id), Name: to.Ptr(name), Type: to.Ptr("Microsoft.Network/virtualNetworks"), Location: to.Ptr(c.Region), Properties: &n.VirtualNetworkPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), AddressSpace: &n.AddressSpace{AddressPrefixes: []*string{to.Ptr(s.VirtualNetworkPrefix)}}, Subnets: []*n.Subnet{clone(&previous.relay), clone(&previous.subnet)}}}}
	reader, err := NewSDKRelayTopologyReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewRelayTopologyGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *relayTopologyFixture) Do(r *http.Request) (*http.Response, error) {
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Path != f.settings.BackendEndpoint.Backend.Exchange.VirtualNetwork || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("relay topology reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-relay-network-canary"}}), nil
	}
	if f.mode == "redirect" {
		r := deadlineResponse(r, 302, nil)
		r.Header.Set("Location", "https://foreign.invalid/private-relay-network-canary")
		return r, nil
	}
	return deadlineResponse(r, 200, f.network), nil
}
