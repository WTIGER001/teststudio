package azureworker

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureBackendEndpointBindsFreshApprovalWithoutAnotherEnvironmentRead(t *testing.T) {
	c, s, f, g := backendEndpointTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 7 || f.backend.reads != 5 {
		t.Fatal("private backend endpoint rejected or environment read repeated", err, f.reads, f.backend.reads)
	}
	s.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	if err := g.Check(t.Context(), c); err != nil || f.reads != 14 || f.backend.reads != 10 {
		t.Fatal("fresh approval reused stale state", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("endpoint observation became complete network admission")
	}
	f.backend.application.Properties.Template.Containers[0].Args[0] = to.Ptr("--serve-role=api")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 14 {
		t.Fatal("unverified backend permitted endpoint reads")
	}
	c.NetworkProfile = "changed"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 14 {
		t.Fatal("different profile reached endpoint observation")
	}
}

func TestAzureBackendEndpointRejectsForeignBindingsAndPolicyBypass(t *testing.T) {
	for name, mutate := range map[string]func(*backendEndpointFixture){
		"other-service": func(f *backendEndpointFixture) {
			f.endpoint.Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceID = to.Ptr(f.settings.Backend.Exchange.environment())
		},
		"foreign-consumer": func(f *backendEndpointFixture) {
			f.connections[0].Properties.PrivateEndpoint.ID = to.Ptr(f.settings.Backend.Exchange.Registry.Endpoints.ExchangeEndpoint)
		},
		"extra-approval":        func(f *backendEndpointFixture) { f.connections = append(f.connections, clone(f.connections[0])) },
		"wrong-required-member": func(f *backendEndpointFixture) { f.links[0].Properties.RequiredMembers[0] = to.Ptr("other-member") },
		"wrong-zone": func(f *backendEndpointFixture) {
			f.links[0].Properties.RequiredZoneNames[0] = to.Ptr("foreign.invalid")
		},
		"other-nic": func(f *backendEndpointFixture) {
			f.endpoint.Properties.NetworkInterfaces[0].ID = to.Ptr(f.settings.Backend.Exchange.Registry.Endpoints.ExchangeNIC)
		},
		"wrong-address": func(f *backendEndpointFixture) {
			f.nic.Properties.IPConfigurations[0].Properties.PrivateIPAddress = to.Ptr("10.40.2.9")
		},
		"other-source-prefix": func(f *backendEndpointFixture) { f.relay.Properties.AddressPrefix = to.Ptr("10.41.0.0/27") },
		"wrong-delegation": func(f *backendEndpointFixture) {
			f.relay.Properties.Delegations[0].Properties.ServiceName = to.Ptr("Microsoft.Web/serverFarms")
		},
		"network-policy-disabled": func(f *backendEndpointFixture) {
			f.subnet.Properties.PrivateEndpointNetworkPolicies = to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled)
		},
		"shared-subnet": func(f *backendEndpointFixture) {
			f.subnet.Properties.PrivateEndpoints = append(f.subnet.Properties.PrivateEndpoints, &n.PrivateEndpoint{ID: to.Ptr(f.settings.Backend.Exchange.Registry.Endpoints.ExchangeEndpoint)})
		},
		"unexpected-route": func(f *backendEndpointFixture) {
			f.subnet.Properties.RouteTable = &n.RouteTable{ID: to.Ptr("private-route-canary")}
		},
		"broad-source": func(f *backendEndpointFixture) {
			f.security.Properties.SecurityRules[0].Properties.SourceAddressPrefix = to.Ptr("VirtualNetwork")
		},
		"wrong-port": func(f *backendEndpointFixture) {
			f.security.Properties.SecurityRules[0].Properties.DestinationPortRange = to.Ptr("*")
		},
		"shared-security-group": func(f *backendEndpointFixture) {
			f.security.Properties.Subnets = append(f.security.Properties.Subnets, &n.Subnet{ID: to.Ptr(f.settings.Backend.Exchange.Subnet)})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := backendEndpointTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureBackendEndpointSDKPreservesUncertaintyAndReadScope(t *testing.T) {
	c, s, f, g := backendEndpointTest(t)
	reader := g.reader.(*SDKBackendEndpointReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.Backend.Application + "/listSecrets" },
		func(r *http.Request) { r.URL.Path = s.Backend.Environment },
		func(r *http.Request) { r.URL.Path = s.Backend.Exchange.Registry.Endpoints.RegistryEndpoint },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=ipConfigurations" },
		func(r *http.Request) { r.URL.RawQuery += "&$skipToken=other" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Endpoint+"?api-version=2025-03-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected request reached transport")
	}
	for _, mode := range []string{"unavailable", "unfinished", "foreign-page"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "foreign-page" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), c), want)
	}
	f.mode = ""
	f.connections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.PrivateEndpointServiceConnectionStatusPending)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.connections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.PrivateEndpointServiceConnectionStatusApproved)
	f.backend.environment.Properties.PrivateEndpointConnections[0].Name = to.Ptr("other.approval")
	f.backend.environment.Properties.PrivateEndpointConnections[0].ID = to.Ptr(s.Backend.Environment + "/privateEndpointConnections/other.approval")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureBackendEndpointSettingsRejectAliasesAndReservedAddresses(t *testing.T) {
	c, s, _, g := backendEndpointTest(t)
	for _, mutate := range []func(*BackendEndpointSettings){
		func(s *BackendEndpointSettings) { s.Subnet = s.Backend.Exchange.Subnet },
		func(s *BackendEndpointSettings) { s.SubnetPrefix = s.RelaySubnetPrefix },
		func(s *BackendEndpointSettings) { s.RelaySubnetPrefix = "10.20.0.0/27" },
		func(s *BackendEndpointSettings) { s.Address = "10.40.2.0" },
		func(s *BackendEndpointSettings) { s.Address = "10.40.2.3" },
		func(s *BackendEndpointSettings) { s.Address = "10.40.2.255" },
		func(s *BackendEndpointSettings) { s.Address = "::ffff:10.40.2.4" },
		func(s *BackendEndpointSettings) {
			s.SecurityGroup = s.Backend.Exchange.Registry.Endpoints.SecurityGroup
		},
		func(s *BackendEndpointSettings) { s.Endpoint += "/foreign" },
	} {
		changed := s
		mutate(&changed)
		_, err := NewBackendEndpointGuard(c, changed, g.backend, g.reader)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	if _, err := NewSDKBackendEndpointReader(c, s, nil, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("implicit endpoint credential discovery accepted")
	}
	if _, err := NewBackendEndpointGuard(c, s, nil, g.reader); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing backend observation accepted")
	}
	first, err := s.SecurityGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	first.Properties.SecurityRules[0].Properties.SourceAddressPrefix = to.Ptr("*")
	second, _ := s.SecurityGroupDefinition(c)
	if value(second.Properties.SecurityRules[0].Properties.SourceAddressPrefix) != s.RelaySubnetPrefix || value(second.Properties.SecurityRules[0].Properties.DestinationAddressPrefix) != s.Address {
		t.Fatal("caller mutation changed a later endpoint definition")
	}
}
