package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type relayRegistryFixture struct {
	t        *testing.T
	mu       sync.Mutex
	settings RelayRegistryEndpointSettings
	dns      *relayDNSFixture
	registry *registryFixture
	network  *relayTopologyFixture
	endpoint n.PrivateEndpoint
	nic      n.Interface
	subnet   n.Subnet
	security n.SecurityGroup
	reads    int
	mode     string
}

func relayRegistryTest(t *testing.T) (Config, RelayRegistryEndpointSettings, *relayRegistryFixture, *RelayRegistryEndpointGuard) {
	t.Helper()
	c, dns, previous, prior := relayDNSTest(t, "relay-registry")
	s := RelayRegistryEndpointSettings{DNS: dns, Endpoint: c.resourceID("Microsoft.Network", "privateEndpoints", "relay-registry"), Interface: c.resourceID("Microsoft.Network", "networkInterfaces", "relay-registry-nic"), Subnet: dns.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork + "/subnets/registry-endpoint", SubnetPrefix: "10.40.3.0/24", SecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "relay-registry-endpoint"), RegistryIP: "10.40.3.4", DataIP: "10.40.3.5"}
	topology := prior.firewall.perimeter.topology
	registry := topology.endpoint.backend.inventory.exchange.registry
	f := &relayRegistryFixture{t: t, settings: s, dns: previous, registry: registry.reader.(*SDKRegistryReader).http.base.(*registryFixture), network: topology.reader.(*SDKRelayTopologyReader).http.base.(*relayTopologyFixture)}
	f.subnet, _ = s.SubnetDefinition(c)
	f.subnet.ID, f.subnet.Name, f.subnet.Type = to.Ptr(s.Subnet), to.Ptr("registry-endpoint"), to.Ptr("Microsoft.Network/virtualNetworks/subnets")
	f.subnet.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.subnet.Properties.PrivateEndpoints = []*n.PrivateEndpoint{{ID: to.Ptr(s.Endpoint)}}
	f.security, _ = s.SecurityGroupDefinition(c)
	f.security.ID, f.security.Name, f.security.Type = to.Ptr(s.SecurityGroup), to.Ptr("relay-registry-endpoint"), to.Ptr("Microsoft.Network/networkSecurityGroups")
	f.security.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.security.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(s.Subnet)}}
	for _, rule := range f.security.Properties.SecurityRules {
		rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	}
	role := s.role()
	f.endpoint = n.PrivateEndpoint{ID: to.Ptr(s.Endpoint), Name: to.Ptr("relay-registry"), Type: to.Ptr("Microsoft.Network/privateEndpoints"), Location: to.Ptr(c.Region), Properties: &n.PrivateEndpointProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, NetworkInterfaces: []*n.Interface{{ID: to.Ptr(s.Interface)}}, PrivateLinkServiceConnections: []*n.PrivateLinkServiceConnection{{Name: to.Ptr("registry-connection"), Properties: &n.PrivateLinkServiceConnectionProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateLinkServiceID: to.Ptr(role.service), GroupIDs: []*string{to.Ptr("registry")}, PrivateLinkServiceConnectionState: &n.PrivateLinkServiceConnectionState{Status: to.Ptr("Approved"), ActionsRequired: to.Ptr("None")}}}}}}
	f.nic = n.Interface{ID: to.Ptr(s.Interface), Name: to.Ptr("relay-registry-nic"), Type: to.Ptr("Microsoft.Network/networkInterfaces"), Location: to.Ptr(c.Region), Properties: &n.InterfacePropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateEndpoint: &n.PrivateEndpoint{ID: to.Ptr(s.Endpoint)}}}
	for _, member := range []string{"registry", "registry_data_" + dns.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.region()} {
		name := strings.ReplaceAll(member, "_", "-")
		id := s.Interface + "/ipConfigurations/" + name
		f.nic.Properties.IPConfigurations = append(f.nic.Properties.IPConfigurations, &n.InterfaceIPConfiguration{ID: to.Ptr(id), Name: to.Ptr(name), Properties: &n.InterfaceIPConfigurationPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateIPAddress: to.Ptr(role.members[member]), PrivateIPAddressVersion: to.Ptr(n.IPVersionIPv4), PrivateIPAllocationMethod: to.Ptr(n.IPAllocationMethodDynamic), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, PrivateLinkConnectionProperties: &n.InterfaceIPConfigurationPrivateLinkConnectionProperties{GroupID: to.Ptr("registry"), RequiredMemberName: to.Ptr(member)}}})
		f.subnet.Properties.IPConfigurations = append(f.subnet.Properties.IPConfigurations, &n.IPConfiguration{ID: to.Ptr(id)})
	}
	f.network.network.Properties.Subnets = append(f.network.network.Properties.Subnets, clone(&f.subnet))
	reader, err := NewSDKRelayRegistryEndpointReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewRelayRegistryEndpointGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *relayRegistryFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("relay registry endpoint reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-relay-registry-canary"}}), nil
	}
	var body any
	switch r.URL.Path {
	case f.settings.Endpoint:
		body = f.endpoint
	case f.settings.Interface:
		body = f.nic
	case f.settings.Subnet:
		body = f.subnet
	case f.settings.SecurityGroup:
		body = f.security
	default:
		f.t.Error("unselected registry endpoint resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	return deadlineResponse(r, 200, body), nil
}
