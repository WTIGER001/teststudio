package azureworker

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureRelayRegistryEndpointBindsFreshApprovalWithoutRegistryRereads(t *testing.T) {
	c, _, f, g := relayRegistryTest(t)
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*4 || f.registry.reads != i*4 || f.network.reads != i || f.dns.reads != i*6 {
			t.Fatal("registry endpoint rejected or parent read repeated", err, f.reads, f.registry.reads, f.network.reads, f.dns.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("registry endpoint observation became complete network admission")
	}
	f.registry.connections[2].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusPending)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 8 {
		t.Fatal("pending service approval reached endpoint reads")
	}
	f.registry.connections[2].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusApproved)
	f.nic.Properties.IPConfigurations[1].Properties.PrivateIPAddress = to.Ptr("10.40.3.6")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 12 {
		t.Fatal("changed data IP reused successful observation")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 12 {
		t.Fatal("foreign profile reached endpoint reads")
	}
}

func TestAzureRelayRegistryEndpointRejectsOwnershipAndPolicyBypasses(t *testing.T) {
	for name, mutate := range map[string]func(*relayRegistryFixture){
		"foreign-service": func(f *relayRegistryFixture) {
			f.endpoint.Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceID = to.Ptr(f.settings.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Environment)
		},
		"foreign-nic": func(f *relayRegistryFixture) {
			f.endpoint.Properties.NetworkInterfaces[0].ID = to.Ptr(f.settings.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Interface)
		},
		"wrong-member": func(f *relayRegistryFixture) {
			f.nic.Properties.IPConfigurations[1].Properties.PrivateLinkConnectionProperties.RequiredMemberName = to.Ptr("registry_data_otherregion")
		},
		"wrong-address": func(f *relayRegistryFixture) {
			f.nic.Properties.IPConfigurations[0].Properties.PrivateIPAddress = to.Ptr("10.40.3.6")
		},
		"public-ip": func(f *relayRegistryFixture) {
			f.nic.Properties.IPConfigurations[0].Properties.PublicIPAddress = &n.PublicIPAddress{}
		},
		"disabled-policy": func(f *relayRegistryFixture) {
			f.subnet.Properties.PrivateEndpointNetworkPolicies = to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled)
		},
		"shared-subnet": func(f *relayRegistryFixture) {
			f.subnet.Properties.PrivateEndpoints = append(f.subnet.Properties.PrivateEndpoints, &n.PrivateEndpoint{ID: to.Ptr(f.settings.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Endpoint)})
		},
		"route-bypass": func(f *relayRegistryFixture) { f.subnet.Properties.RouteTable = &n.RouteTable{} },
		"broad-source": func(f *relayRegistryFixture) {
			f.security.Properties.SecurityRules[0].Properties.SourceAddressPrefix = to.Ptr("VirtualNetwork")
		},
		"wrong-port": func(f *relayRegistryFixture) {
			f.security.Properties.SecurityRules[0].Properties.DestinationPortRange = to.Ptr("*")
		},
		"extra-destination": func(f *relayRegistryFixture) {
			f.security.Properties.SecurityRules[0].Properties.DestinationAddressPrefixes = append(f.security.Properties.SecurityRules[0].Properties.DestinationAddressPrefixes, to.Ptr("10.40.3.6"))
		},
		"shared-group": func(f *relayRegistryFixture) {
			f.security.Properties.Subnets = append(f.security.Properties.Subnets, &n.Subnet{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := relayRegistryTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, _, f, g := relayRegistryTest(t)
	f.network.network.Properties.Subnets[3].Properties.DefaultOutboundAccess = to.Ptr(true)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.network.network.Properties.Subnets = f.network.network.Properties.Subnets[:3]
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayRegistryEndpointSDKPreservesUncertaintyAndResourceScope(t *testing.T) {
	c, s, f, g := relayRegistryTest(t)
	reader := g.reader.(*SDKRelayRegistryEndpointReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) {
			r.URL.Path = s.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.connections()
		},
		func(r *http.Request) { r.URL.Path = s.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Endpoint },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=all" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Endpoint+"?api-version=2025-03-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Endpoint(t.Context(), s.Interface)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected request reached transport")
	}
	f.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.mode = ""
	f.endpoint.Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr("Pending")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayRegistryEndpointSettingsBindSelectedConsumerAndAddresses(t *testing.T) {
	c, s, _, g := relayRegistryTest(t)
	for _, mutate := range []func(*RelayRegistryEndpointSettings){
		func(s *RelayRegistryEndpointSettings) {
			s.Endpoint = s.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.PreparationEndpoint
		},
		func(s *RelayRegistryEndpointSettings) {
			s.Interface = s.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Interface
		},
		func(s *RelayRegistryEndpointSettings) {
			s.Subnet = s.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Subnet
		},
		func(s *RelayRegistryEndpointSettings) { s.SubnetPrefix = "10.40.2.0/24" },
		func(s *RelayRegistryEndpointSettings) { s.RegistryIP = "10.40.3.3" },
		func(s *RelayRegistryEndpointSettings) { s.DataIP = "10.40.3.255" },
		func(s *RelayRegistryEndpointSettings) { s.DataIP = "10.40.3.6" },
		func(s *RelayRegistryEndpointSettings) { s.SecurityGroup = s.DNS.Firewall.Perimeter.SecurityGroup },
	} {
		changed := s
		mutate(&changed)
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	for _, reserved := range []string{"10.40.3.0", "10.40.3.3", "10.40.3.255"} {
		changed := s
		changed.RegistryIP = reserved
		changed.DNS.Firewall.Perimeter.RegistryEndpointIPs = []string{reserved, changed.DataIP}
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.DNS.Policy = c.resourceID("Microsoft.Network", "dnsResolverPolicies", "other-relay-dns")
	_, err := NewRelayRegistryEndpointGuard(c, changed, g.dns, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKRelayRegistryEndpointReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	first, _ := s.SecurityGroupDefinition(c)
	first.Properties.SecurityRules[0].Properties.DestinationAddressPrefixes[0] = to.Ptr("*")
	second, _ := s.SecurityGroupDefinition(c)
	addresses := second.Properties.SecurityRules[0].Properties.DestinationAddressPrefixes
	if len(addresses) != 2 || value(addresses[0]) != s.RegistryIP || value(addresses[1]) != s.DataIP {
		t.Fatal("registry endpoint destinations widened or retained mutation")
	}
}

func TestAzureRegistryServiceSupportsOnlyExplicitRelayConsumer(t *testing.T) {
	c, s, f, g := registryServiceTest(t, "relay-registry")
	f.mode = "paged"
	if err := g.Check(t.Context(), c); err != nil || f.reads != 5 {
		t.Fatal("explicit three-consumer registry profile rejected", err, f.reads)
	}
	before := f.pages[s.connections()]
	f.mode = "endless"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.pages[s.connections()]-before != 4 {
		t.Fatal("explicit relay consumer inventory exceeded its four-page bound")
	}
	f.mode = ""
	f.connections = append(f.connections, clone(f.connections[2]))
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	f.connections = f.connections[:3]
	f.connections[2].Properties.PrivateEndpoint.ID = to.Ptr(c.resourceID("Microsoft.Network", "privateEndpoints", "unknown-consumer"))
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	f.connections[2].Properties.PrivateEndpoint.ID = to.Ptr(s.RelayEndpoint)
	f.connections = f.connections[:2]
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	c, _, legacy, old := registryServiceTest(t)
	legacy.connections = append(legacy.connections, clone(f.registry.Properties.PrivateEndpointConnections[2]))
	requireDeadlineError(t, old.Check(t.Context(), c), hostedexecution.ErrOwnership)
}

func TestAzureRegistryRelayConsumerSettingsRejectAliases(t *testing.T) {
	c, s, _, _ := registryServiceTest(t, "relay-registry")
	for _, id := range []string{s.PreparationEndpoint, s.Endpoints.RegistryEndpoint, s.Endpoints.ExchangeEndpoint, s.Endpoints.RegistryNIC, s.RelayEndpoint + "/foreign"} {
		changed := s
		changed.RelayEndpoint = id
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
}
