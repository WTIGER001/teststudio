package azureworker

import (
	"net/http"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func relayFirewallTest(t *testing.T, relayName ...string) (Config, RelayFirewallSettings, *firewallFixture, *relayPerimeterFixture, *RelayFirewallGuard) {
	t.Helper()
	c, perimeter, previous, prior := relayPerimeterTest(t, relayName...)
	s := RelayFirewallSettings{Perimeter: perimeter, Firewall: c.resourceID("Microsoft.Network", "azureFirewalls", "relay-firewall"), Policy: c.resourceID("Microsoft.Network", "firewallPolicies", "relay-policy"), PublicIP: c.resourceID("Microsoft.Network", "publicIPAddresses", "relay-firewall-ip"), SubnetPrefix: "10.40.1.0/26"}
	f := firewallBindingFixture(t, c, s.binding())
	previous.topology.network.Properties.Subnets = append(previous.topology.network.Properties.Subnets, clone(&f.subnet))
	reader, err := NewSDKRelayFirewallReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewRelayFirewallGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, previous, g
}

func TestAzureRelayFirewallBindsFreshOwnershipAndReusesNetwork(t *testing.T) {
	c, s, f, previous, g := relayFirewallTest(t)
	s.Perimeter.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*5 || previous.reads != i*2 || previous.topology.reads != i || previous.topology.endpoint.reads != i*7 {
			t.Fatal("relay firewall rejected or parent observations repeated", err, f.reads, previous.reads, previous.topology.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("firewall observation became complete network admission")
	}
	f.allow().SourceAddresses[0] = to.Ptr("10.20.0.0/27")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 15 {
		t.Fatal("changed source policy reused cached success")
	}
	previous.routes.Properties.Routes[0].Properties.NextHopType = to.Ptr(n.RouteNextHopTypeInternet)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 15 {
		t.Fatal("unverified source perimeter reached firewall reads")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 15 {
		t.Fatal("foreign profile reached firewall reads")
	}
}

func TestAzureRelayFirewallRejectsForeignBindingsAndPolicyBypasses(t *testing.T) {
	for name, mutate := range map[string]func(*firewallFixture){
		"worker-address": func(f *firewallFixture) {
			f.firewall.Properties.IPConfigurations[0].Properties.PrivateIPAddress = to.Ptr("10.20.1.4")
		},
		"foreign-policy": func(f *firewallFixture) { f.firewall.Properties.FirewallPolicy.ID = to.Ptr("foreign-policy-canary") },
		"classic-network": func(f *firewallFixture) {
			f.firewall.Properties.NetworkRuleCollections = []*n.AzureFirewallNetworkRuleCollection{{}}
		},
		"foreign-subnet": func(f *firewallFixture) {
			f.firewall.Properties.IPConfigurations[0].Properties.Subnet.ID = to.Ptr("foreign-subnet-canary")
		},
		"public-owner":  func(f *firewallFixture) { f.public.Properties.IPConfiguration.ID = to.Ptr("foreign-owner-canary") },
		"subnet-route":  func(f *firewallFixture) { f.subnet.Properties.RouteTable = &n.RouteTable{} },
		"policy-parent": func(f *firewallFixture) { f.policy.Properties.BasePolicy = &n.SubResource{} },
		"policy-shared": func(f *firewallFixture) {
			f.policy.Properties.Firewalls = append(f.policy.Properties.Firewalls, &n.SubResource{})
		},
		"wildcard": func(f *firewallFixture) { f.allow().TargetFqdns = []*string{to.Ptr("*")} },
		"registry-wildcard": func(f *firewallFixture) {
			f.allow().TargetFqdns = append(f.allow().TargetFqdns, to.Ptr("*.azurecr.io"))
		},
		"source":           func(f *firewallFixture) { f.allow().SourceAddresses[0] = to.Ptr("*") },
		"extra-port":       func(f *firewallFixture) { f.allow().Protocols[0].Port = to.Ptr[int32](8443) },
		"tls-interception": func(f *firewallFixture) { f.allow().TerminateTLS = to.Ptr(true) },
		"dns-proxy":        func(f *firewallFixture) { f.policy.Properties.DNSSettings.EnableProxy = to.Ptr(true) },
		"populated-default": func(f *firewallFixture) {
			f.groups[1].Properties.RuleCollections = clone(f.groups[0].Properties.RuleCollections)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, _, g := relayFirewallTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, _, _, previous, g := relayFirewallTest(t)
	previous.topology.network.Properties.Subnets[2].Properties.DefaultOutboundAccess = to.Ptr(true)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	previous.topology.network.Properties.Subnets = previous.topology.network.Properties.Subnets[:2]
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayFirewallSDKPreservesPagingUncertaintyAndReadScope(t *testing.T) {
	c, s, f, _, g := relayFirewallTest(t)
	reader := g.reader.(*SDKFirewallReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) {
			r.URL.Path = s.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Firewall
		},
		func(r *http.Request) { r.URL.Path = s.Perimeter.SecurityGroup },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$skipToken=unselected" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Firewall+"?api-version=2025-03-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Firewall(t.Context(), s.Policy)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected request reached transport")
	}
	f.mode = "paged"
	if err := g.Check(t.Context(), c); err != nil || f.pages != 2 || f.reads != 6 {
		t.Fatal("bounded relay inventory lost paging", err, f.pages, f.reads)
	}
	for _, mode := range []string{"unavailable", "foreign-page", "endless", "groups-hidden"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "foreign-page" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), c), want)
	}
	f.mode = ""
	f.firewall.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayFirewallSettingsFreezeDedicatedResources(t *testing.T) {
	c, s, _, _, g := relayFirewallTest(t)
	for _, mutate := range []func(*RelayFirewallSettings){
		func(s *RelayFirewallSettings) {
			s.Firewall = s.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Firewall
		},
		func(s *RelayFirewallSettings) {
			s.Policy = s.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Policy
		},
		func(s *RelayFirewallSettings) {
			s.PublicIP = s.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.PublicIP
		},
		func(s *RelayFirewallSettings) { s.SubnetPrefix = "10.40.1.0/27" },
		func(s *RelayFirewallSettings) { s.SubnetPrefix = "10.40.1.1/26" },
		func(s *RelayFirewallSettings) { s.SubnetPrefix = "10.40.0.0/23" },
		func(s *RelayFirewallSettings) { s.Perimeter.FirewallIP = "10.40.1.3" },
		func(s *RelayFirewallSettings) { s.Perimeter.FirewallIP = "10.40.1.63" },
		func(s *RelayFirewallSettings) { s.Perimeter.RegistryEndpointIPs[0] = "10.40.1.6" },
	} {
		changed := s
		changed.Perimeter.RegistryEndpointIPs = slices.Clone(s.Perimeter.RegistryEndpointIPs)
		mutate(&changed)
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.Perimeter.SecurityGroup = c.resourceID("Microsoft.Network", "networkSecurityGroups", "other-relay")
	_, err := NewRelayFirewallGuard(c, changed, g.perimeter, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKRelayFirewallReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	first, _ := s.RuleGroupDefinition(c)
	first.Properties.RuleCollections[0].(*n.FirewallPolicyFilterRuleCollection).Rules[0].(*n.ApplicationRule).TargetFqdns[0] = to.Ptr("*")
	second, _ := s.RuleGroupDefinition(c)
	allow := second.Properties.RuleCollections[0].(*n.FirewallPolicyFilterRuleCollection).Rules[0].(*n.ApplicationRule)
	if len(allow.SourceAddresses) != 1 || value(allow.SourceAddresses[0]) != "10.40.0.0/27" || value(allow.TargetFqdns[0]) == "*" {
		t.Fatal("relay definitions changed their source or retained caller mutation")
	}
}
