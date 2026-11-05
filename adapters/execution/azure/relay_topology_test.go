package azureworker

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureRelayTopologyReusesFreshSubnetObservations(t *testing.T) {
	c, _, f, g := relayTopologyTest(t)
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i || f.endpoint.reads != i*7 || f.endpoint.backend.reads != i*5 {
			t.Fatal("relay topology rejected or repeated parent reads", err, f.reads, f.endpoint.reads, f.endpoint.backend.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("relay topology became complete network admission")
	}
	f.network.Properties.DhcpOptions = &n.DhcpOptions{DNSServers: []*string{to.Ptr("10.40.3.4")}}
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 3 {
		t.Fatal("changed relay topology reused successful observation")
	}
	f.endpoint.backend.application.Properties.Template.Containers[0].Args[0] = to.Ptr("--serve-role=api")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 3 {
		t.Fatal("unverified backend reached relay topology")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 3 {
		t.Fatal("foreign profile reached relay topology")
	}
}

func TestAzureRelayTopologyRejectsAlternatePathsAndSubnetConflicts(t *testing.T) {
	for name, mutate := range map[string]func(*relayTopologyFixture){
		"other-network": func(f *relayTopologyFixture) {
			f.network.ID = to.Ptr(f.settings.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerVNet)
		},
		"other-region": func(f *relayTopologyFixture) { f.network.Location = to.Ptr("westus") },
		"other-space": func(f *relayTopologyFixture) {
			f.network.Properties.AddressSpace.AddressPrefixes[0] = to.Ptr("10.41.0.0/16")
		},
		"extra-space": func(f *relayTopologyFixture) {
			f.network.Properties.AddressSpace.AddressPrefixes = append(f.network.Properties.AddressSpace.AddressPrefixes, to.Ptr("10.41.0.0/16"))
		},
		"peering": func(f *relayTopologyFixture) {
			f.network.Properties.VirtualNetworkPeerings = []*n.VirtualNetworkPeering{{ID: to.Ptr("private-peering-canary")}}
		},
		"public-nat": func(f *relayTopologyFixture) {
			f.network.Properties.DefaultPublicNatGateway = &n.SubResource{ID: to.Ptr("private-nat-canary")}
		},
		"bgp": func(f *relayTopologyFixture) { f.network.Properties.BgpCommunities = &n.VirtualNetworkBgpCommunities{} },
		"allocation": func(f *relayTopologyFixture) {
			f.network.Properties.IPAllocations = []*n.SubResource{{ID: to.Ptr("private-allocation-canary")}}
		},
		"nil-subnet": func(f *relayTopologyFixture) { f.network.Properties.Subnets[0] = nil },
		"duplicate-subnet": func(f *relayTopologyFixture) {
			f.network.Properties.Subnets = append(f.network.Properties.Subnets, clone(f.network.Properties.Subnets[0]))
		},
		"foreign-subnet": func(f *relayTopologyFixture) {
			f.network.Properties.Subnets[0].ID = to.Ptr(f.settings.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerSubnet)
		},
		"unknown-platform-name": func(f *relayTopologyFixture) {
			f.network.Properties.Subnets[0].ID = to.Ptr(f.settings.BackendEndpoint.Backend.Exchange.VirtualNetwork + "/subnets/ForeignSubnet")
			f.network.Properties.Subnets[0].Name = to.Ptr("ForeignSubnet")
		},
		"outside-prefix": func(f *relayTopologyFixture) {
			f.network.Properties.Subnets[0].Properties.AddressPrefix = to.Ptr("10.41.0.0/27")
		},
		"overlapping-prefix": func(f *relayTopologyFixture) {
			f.network.Properties.Subnets[0].Properties.AddressPrefix = to.Ptr(f.settings.BackendEndpoint.SubnetPrefix)
		},
		"subnet-limit": func(f *relayTopologyFixture) { f.network.Properties.Subnets = make([]*n.Subnet, 257) },
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := relayTopologyTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, s, f, g := relayTopologyTest(t)
	extra := clone(f.network.Properties.Subnets[0])
	extra.ID, extra.Name = to.Ptr(s.BackendEndpoint.Backend.Exchange.VirtualNetwork+"/subnets/preparation"), to.Ptr("preparation")
	extra.Properties.AddressPrefix = to.Ptr("10.40.3.0/24")
	f.network.Properties.Subnets = append(f.network.Properties.Subnets, extra)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("bounded additional subnet rejected", err)
	}
}

func TestAzureRelayTopologyPreservesUncertaintyAndExactSDKScope(t *testing.T) {
	c, s, f, g := relayTopologyTest(t)
	reader := g.reader.(*SDKRelayTopologyReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path += "/subnets" },
		func(r *http.Request) { r.URL.Path = s.BackendEndpoint.Subnet },
		func(r *http.Request) { r.URL.Path = s.BackendEndpoint.Backend.Environment },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery = "api-version=2024-01-01" },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=subnets" },
		func(r *http.Request) { r.URL.RawQuery += "&$skipToken=other" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.BackendEndpoint.Backend.Exchange.VirtualNetwork+"?api-version=2025-03-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.VirtualNetwork(t.Context(), s.BackendEndpoint.Subnet)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected request reached transport")
	}
	for _, mode := range []string{"unavailable", "redirect"} {
		f.mode = mode
		requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	}
	f.mode = ""
	f.network.Properties.Subnets[0].Properties.RouteTable = &n.RouteTable{ID: to.Ptr("changed-route-canary")}
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.network.Properties.Subnets = nil
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.network.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayTopologySettingsBindTheCompleteParent(t *testing.T) {
	c, s, _, g := relayTopologyTest(t)
	for _, prefix := range []string{"8.0.0.0/8", "10.40.0.1/16", "10.40.0.0/27", "10.0.0.0/8", "::ffff:10.40.0.0/112"} {
		changed := s
		changed.VirtualNetworkPrefix = prefix
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.BackendEndpoint.SecurityGroup = c.resourceID("Microsoft.Network", "networkSecurityGroups", "other-endpoint")
	_, err := NewRelayTopologyGuard(c, changed, g.endpoint, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewRelayTopologyGuard(c, s, nil, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKRelayTopologyReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
}
