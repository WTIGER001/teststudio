package azureworker

import (
	"net/http"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureRelayPerimeterReusesSubnetsAndRefreshesRules(t *testing.T) {
	c, s, f, g := relayPerimeterTest(t)
	s.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*2 || f.topology.reads != i || f.topology.endpoint.reads != i*7 {
			t.Fatal("relay perimeter rejected or repeated parent reads", err, f.reads, f.topology.reads, f.topology.endpoint.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("relay perimeter became complete network admission")
	}
	f.routes.Properties.Routes[0].Properties.NextHopType = to.Ptr(n.RouteNextHopTypeInternet)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 6 {
		t.Fatal("changed routes reused cached success")
	}
	f.topology.network.Properties.VirtualNetworkPeerings = []*n.VirtualNetworkPeering{{}}
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 6 {
		t.Fatal("unverified topology reached perimeter reads")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 6 {
		t.Fatal("foreign profile reached perimeter reads")
	}
}

func TestAzureRelayPerimeterRejectsSubnetRuleAndRouteBypasses(t *testing.T) {
	for name, mutate := range map[string]func(*relayPerimeterFixture){
		"default-outbound": func(f *relayPerimeterFixture) {
			f.topology.endpoint.relay.Properties.DefaultOutboundAccess = to.Ptr(true)
		},
		"missing-outbound": func(f *relayPerimeterFixture) { f.topology.endpoint.relay.Properties.DefaultOutboundAccess = nil },
		"other-group": func(f *relayPerimeterFixture) {
			f.topology.endpoint.relay.Properties.NetworkSecurityGroup.ID = to.Ptr(f.settings.Topology.BackendEndpoint.SecurityGroup)
		},
		"other-table": func(f *relayPerimeterFixture) {
			f.topology.endpoint.relay.Properties.RouteTable.ID = to.Ptr(f.settings.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.RouteTable)
		},
		"nat": func(f *relayPerimeterFixture) {
			f.topology.endpoint.relay.Properties.NatGateway = &n.SubResource{ID: to.Ptr("private-nat-canary")}
		},
		"service-endpoint": func(f *relayPerimeterFixture) {
			f.topology.endpoint.relay.Properties.ServiceEndpoints = []*n.ServiceEndpointPropertiesFormat{{Service: to.Ptr("Microsoft.Storage")}}
		},
		"shared-nsg": func(f *relayPerimeterFixture) {
			f.security.Properties.Subnets = append(f.security.Properties.Subnets, &n.Subnet{ID: to.Ptr(f.settings.Topology.BackendEndpoint.Subnet)})
		},
		"nsg-interface": func(f *relayPerimeterFixture) { f.security.Properties.NetworkInterfaces = []*n.Interface{{}} },
		"no-flush":      func(f *relayPerimeterFixture) { f.security.Properties.FlushConnection = to.Ptr(false) },
		"broad-destination": func(f *relayPerimeterFixture) {
			f.security.Properties.SecurityRules[11].Properties.DestinationAddressPrefixes = []*string{to.Ptr("10.40.0.0/16")}
		},
		"wrong-port": func(f *relayPerimeterFixture) {
			f.security.Properties.SecurityRules[11].Properties.DestinationPortRange = to.Ptr("*")
		},
		"metadata-allow": func(f *relayPerimeterFixture) {
			f.security.Properties.SecurityRules[3].Properties.Access = to.Ptr(n.SecurityRuleAccessAllow)
		},
		"dns-bypass": func(f *relayPerimeterFixture) {
			f.security.Properties.SecurityRules[5].Properties.DestinationAddressPrefix = to.Ptr("*")
		},
		"extra-rule": func(f *relayPerimeterFixture) {
			f.security.Properties.SecurityRules = append(f.security.Properties.SecurityRules, &n.SecurityRule{Name: to.Ptr("escape")})
		},
		"internet-route": func(f *relayPerimeterFixture) {
			f.routes.Properties.Routes[0].Properties.NextHopType = to.Ptr(n.RouteNextHopTypeInternet)
		},
		"other-firewall": func(f *relayPerimeterFixture) {
			f.routes.Properties.Routes[0].Properties.NextHopIPAddress = to.Ptr("10.40.1.5")
		},
		"extra-route": func(f *relayPerimeterFixture) {
			f.routes.Properties.Routes = append(f.routes.Properties.Routes, clone(f.routes.Properties.Routes[0]))
		},
		"bgp": func(f *relayPerimeterFixture) { f.routes.Properties.DisableBgpRoutePropagation = to.Ptr(false) },
		"shared-route": func(f *relayPerimeterFixture) {
			f.routes.Properties.Subnets = append(f.routes.Properties.Subnets, &n.Subnet{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := relayPerimeterTest(t)
			mutate(f)
			f.topology.network.Properties.Subnets[0] = clone(&f.topology.endpoint.relay)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureRelayPerimeterPreservesUncertaintyAndExactSDKScope(t *testing.T) {
	c, s, f, g := relayPerimeterTest(t)
	reader := g.reader.(*SDKPerimeterReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path = s.Topology.BackendEndpoint.Subnet },
		func(r *http.Request) { r.URL.Path = s.Topology.BackendEndpoint.SecurityGroup },
		func(r *http.Request) { r.URL.Path += "/securityRules" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=rules" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.SecurityGroup+"?api-version=2025-03-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.SecurityGroup(t.Context(), s.RouteTable)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	_, err = reader.RouteTable(t.Context(), s.SecurityGroup)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected request reached transport")
	}
	f.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.mode = ""
	f.routes.Properties.Routes[0].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.security.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayPerimeterSettingsAndDefinitionsFreezeAuthority(t *testing.T) {
	c, s, _, g := relayPerimeterTest(t)
	for _, mutate := range []func(*RelayPerimeterSettings){
		func(s *RelayPerimeterSettings) { s.SecurityGroup = s.Topology.BackendEndpoint.SecurityGroup },
		func(s *RelayPerimeterSettings) {
			s.RouteTable = s.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.RouteTable
		},
		func(s *RelayPerimeterSettings) { s.FirewallIP = "8.8.8.8" },
		func(s *RelayPerimeterSettings) { s.FirewallIP = "10.40.0.4" },
		func(s *RelayPerimeterSettings) { s.FirewallIP = "10.40.2.4" },
		func(s *RelayPerimeterSettings) { s.RegistryEndpointIPs[0] = s.FirewallIP },
		func(s *RelayPerimeterSettings) { s.RegistryEndpointIPs[0] = "10.20.2.4" },
		func(s *RelayPerimeterSettings) { s.RegistryEndpointIPs[0] = s.RegistryEndpointIPs[1] },
		func(s *RelayPerimeterSettings) { s.RegistryEndpointIPs = s.RegistryEndpointIPs[:1] },
	} {
		changed := s
		changed.RegistryEndpointIPs = slices.Clone(s.RegistryEndpointIPs)
		mutate(&changed)
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.Topology.VirtualNetworkPrefix = "10.40.0.0/20"
	_, err := NewRelayPerimeterGuard(c, changed, g.topology, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKRelayPerimeterReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	first, err := s.SecurityGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	first.Properties.SecurityRules[11].Properties.DestinationAddressPrefixes[0] = to.Ptr("*")
	second, _ := s.SecurityGroupDefinition(c)
	addresses := second.Properties.SecurityRules[11].Properties.DestinationAddressPrefixes
	if len(addresses) != 3 || value(addresses[0]) != "10.40.2.4" || value(addresses[1]) != "10.40.3.4" || value(addresses[2]) != "10.40.3.5" {
		t.Fatal("private HTTPS destinations changed or definition retained mutation")
	}
}
