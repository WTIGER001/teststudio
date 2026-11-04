package azureworker

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type backendEndpointFixture struct {
	t           *testing.T
	mu          sync.Mutex
	settings    BackendEndpointSettings
	endpoint    n.PrivateEndpoint
	nic         n.Interface
	subnet      n.Subnet
	relay       n.Subnet
	security    n.SecurityGroup
	connections []*a.PrivateEndpointConnection
	links       []*a.PrivateLinkResource
	backend     *backendFixture
	reads       int
	pages       int
	mode        string
}

func backendEndpointTest(t *testing.T) (Config, BackendEndpointSettings, *backendEndpointFixture, *BackendEndpointGuard) {
	t.Helper()
	c, backend, previous, prior := backendServiceTest(t)
	s := BackendEndpointSettings{Backend: backend, Endpoint: c.resourceID("Microsoft.Network", "privateEndpoints", "relay-backend"), Interface: c.resourceID("Microsoft.Network", "networkInterfaces", "relay-backend-nic"), Subnet: backend.Exchange.VirtualNetwork + "/subnets/backend-endpoint", SubnetPrefix: "10.40.2.0/24", RelaySubnetPrefix: "10.40.0.0/27", SecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "relay-backend-endpoint"), Address: "10.40.2.4"}
	f := &backendEndpointFixture{t: t, settings: s, backend: previous}
	f.subnet, _ = s.SubnetDefinition(c)
	f.subnet.ID, f.subnet.Name, f.subnet.Type = to.Ptr(s.Subnet), to.Ptr("backend-endpoint"), to.Ptr("Microsoft.Network/virtualNetworks/subnets")
	f.subnet.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.subnet.Properties.PrivateEndpoints = []*n.PrivateEndpoint{{ID: to.Ptr(s.Endpoint)}}
	f.security, _ = s.SecurityGroupDefinition(c)
	f.security.ID, f.security.Name, f.security.Type = to.Ptr(s.SecurityGroup), to.Ptr("relay-backend-endpoint"), to.Ptr("Microsoft.Network/networkSecurityGroups")
	f.security.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.security.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(s.Subnet)}}
	for _, rule := range f.security.Properties.SecurityRules {
		rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	}
	f.endpoint = n.PrivateEndpoint{ID: to.Ptr(s.Endpoint), Name: to.Ptr("relay-backend"), Type: to.Ptr("Microsoft.Network/privateEndpoints"), Location: to.Ptr(c.Region), Properties: &n.PrivateEndpointProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, NetworkInterfaces: []*n.Interface{{ID: to.Ptr(s.Interface)}}, PrivateLinkServiceConnections: []*n.PrivateLinkServiceConnection{{Name: to.Ptr("backend-connection"), Properties: &n.PrivateLinkServiceConnectionProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateLinkServiceID: to.Ptr(s.Backend.Environment), GroupIDs: []*string{to.Ptr("managedEnvironment")}, PrivateLinkServiceConnectionState: &n.PrivateLinkServiceConnectionState{Status: to.Ptr("Approved"), ActionsRequired: to.Ptr("None")}}}}}}
	configuration := s.Interface + "/ipConfigurations/backend-address"
	f.nic = n.Interface{ID: to.Ptr(s.Interface), Name: to.Ptr("relay-backend-nic"), Type: to.Ptr("Microsoft.Network/networkInterfaces"), Location: to.Ptr(c.Region), Properties: &n.InterfacePropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateEndpoint: &n.PrivateEndpoint{ID: to.Ptr(s.Endpoint)}, IPConfigurations: []*n.InterfaceIPConfiguration{{ID: to.Ptr(configuration), Name: to.Ptr("backend-address"), Properties: &n.InterfaceIPConfigurationPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateIPAddress: to.Ptr(s.Address), PrivateIPAddressVersion: to.Ptr(n.IPVersionIPv4), PrivateIPAllocationMethod: to.Ptr(n.IPAllocationMethodDynamic), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, PrivateLinkConnectionProperties: &n.InterfaceIPConfigurationPrivateLinkConnectionProperties{GroupID: to.Ptr("managedEnvironment"), RequiredMemberName: to.Ptr("backend-member")}}}}}}
	f.subnet.Properties.IPConfigurations = []*n.IPConfiguration{{ID: to.Ptr(configuration)}}
	f.relay = n.Subnet{ID: to.Ptr(backend.Exchange.Subnet), Name: to.Ptr("infrastructure"), Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"), Properties: &n.SubnetPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), AddressPrefix: to.Ptr(s.RelaySubnetPrefix), Delegations: []*n.Delegation{{Name: to.Ptr("apps"), Properties: &n.ServiceDelegationPropertiesFormat{ServiceName: to.Ptr("Microsoft.App/environments"), ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded)}}}}}
	f.connections = []*a.PrivateEndpointConnection{{ID: to.Ptr(backend.Environment + "/privateEndpointConnections/backend.approved"), Name: to.Ptr("backend.approved"), Type: to.Ptr("Microsoft.App/managedEnvironments/privateEndpointConnections"), Properties: &a.PrivateEndpointConnectionProperties{PrivateEndpoint: &a.PrivateEndpoint{ID: to.Ptr(s.Endpoint)}, GroupIDs: []*string{to.Ptr("managedEnvironment")}, ProvisioningState: to.Ptr(a.PrivateEndpointConnectionProvisioningStateSucceeded), PrivateLinkServiceConnectionState: &a.PrivateLinkServiceConnectionState{Status: to.Ptr(a.PrivateEndpointServiceConnectionStatusApproved), ActionsRequired: to.Ptr("None")}}}}
	f.links = []*a.PrivateLinkResource{{ID: to.Ptr(backend.Environment + "/privateLinkResources/managedEnvironment"), Name: to.Ptr("managedEnvironment"), Type: to.Ptr("Microsoft.App/managedEnvironments/privateLinkResources"), Properties: &a.PrivateLinkResourceProperties{GroupID: to.Ptr("managedEnvironment"), RequiredMembers: []*string{to.Ptr("backend-member")}, RequiredZoneNames: []*string{to.Ptr("privatelink." + c.Region + ".azurecontainerapps.io")}}}}
	previous.environment.Properties.PrivateEndpointConnections = clone(f.connections)
	reader, err := NewSDKBackendEndpointReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewBackendEndpointGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *backendEndpointFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	version := "2025-03-01"
	if strings.Contains(r.URL.Path, "/providers/Microsoft.App/") {
		version = exchangeVersion
	}
	if r.Method != "GET" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != version || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("backend endpoint reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-backend-endpoint-canary"}}), nil
	}
	var body any
	s := f.settings
	switch r.URL.Path {
	case s.Endpoint:
		body = f.endpoint
	case s.Interface:
		body = f.nic
	case s.Subnet:
		body = f.subnet
	case s.SecurityGroup:
		body = f.security
	case s.Backend.Exchange.Subnet:
		body = f.relay
	case s.Backend.Environment + "/privateEndpointConnections":
		body = map[string]any{"value": f.connections}
		if f.mode == "unfinished" {
			f.pages++
			body = map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + r.URL.Path + "?api-version=" + exchangeVersion + "&%24skipToken=" + strconv.Itoa(f.pages)}
		}
		if f.mode == "foreign-page" {
			body = map[string]any{"value": f.connections, "nextLink": "https://foreign.invalid/private-canary"}
		}
	case s.Backend.Environment + "/privateLinkResources":
		body = map[string]any{"value": f.links}
	default:
		f.t.Error("unselected backend endpoint resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	return deadlineResponse(r, 200, body), nil
}
