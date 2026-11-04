package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type endpointFixture struct {
	t          *testing.T
	mu         sync.Mutex
	settings   EndpointSettings
	endpoints  map[string]*n.PrivateEndpoint
	interfaces map[string]*n.Interface
	subnet     n.Subnet
	security   n.SecurityGroup
	reads      int
	mode       string
}

func endpointTest(t *testing.T) (Config, EndpointSettings, *endpointFixture, *EndpointGuard) {
	t.Helper()
	c, dns, _, guard := privateDNSTest(t)
	s := EndpointSettings{PrivateDNS: dns, Subnet: dns.DNS.Firewall.Perimeter.Topology.WorkerVNet + "/subnets/endpoints", SubnetPrefix: "10.20.2.0/24", SecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "worker-endpoints"), Registry: c.resourceID("Microsoft.ContainerRegistry", "registries", strings.TrimSuffix(c.Registry, ".azurecr.io")), RegistryEndpoint: c.resourceID("Microsoft.Network", "privateEndpoints", "worker-registry"), RegistryNIC: c.resourceID("Microsoft.Network", "networkInterfaces", "registry-endpoint-nic"), ExchangeEnvironment: c.resourceID("Microsoft.App", "managedEnvironments", "studio-exchange"), ExchangeEndpoint: c.resourceID("Microsoft.Network", "privateEndpoints", "worker-exchange"), ExchangeNIC: c.resourceID("Microsoft.Network", "networkInterfaces", "exchange-endpoint-nic")}
	f := &endpointFixture{t: t, settings: s, endpoints: map[string]*n.PrivateEndpoint{}, interfaces: map[string]*n.Interface{}}
	f.subnet, _ = s.SubnetDefinition(c)
	f.subnet.ID, f.subnet.Name, f.subnet.Type = to.Ptr(s.Subnet), to.Ptr("endpoints"), to.Ptr("Microsoft.Network/virtualNetworks/subnets")
	f.subnet.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.security, _ = s.SecurityGroupDefinition(c)
	_, securityName := resourceParts(s.SecurityGroup)
	f.security.ID, f.security.Name, f.security.Type = to.Ptr(s.SecurityGroup), to.Ptr(securityName), to.Ptr("Microsoft.Network/networkSecurityGroups")
	f.security.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.security.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(s.Subnet)}}
	for _, rule := range f.security.Properties.SecurityRules {
		rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	}
	for _, row := range []struct {
		endpoint, nic, service, group string
		members, ips                  []string
	}{
		{s.RegistryEndpoint, s.RegistryNIC, s.Registry, "registry", []string{"registry", "registry_data_westus2"}, []string{"10.20.2.5", "10.20.2.6"}},
		{s.ExchangeEndpoint, s.ExchangeNIC, s.ExchangeEnvironment, "managedEnvironment", []string{"managedEnvironment"}, []string{"10.20.2.4"}},
	} {
		_, endpointName := resourceParts(row.endpoint)
		_, nicName := resourceParts(row.nic)
		endpoint := &n.PrivateEndpoint{ID: to.Ptr(row.endpoint), Name: to.Ptr(endpointName), Type: to.Ptr("Microsoft.Network/privateEndpoints"), Location: to.Ptr(c.Region), Properties: &n.PrivateEndpointProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, NetworkInterfaces: []*n.Interface{{ID: to.Ptr(row.nic)}}, PrivateLinkServiceConnections: []*n.PrivateLinkServiceConnection{{Name: to.Ptr("service-connection"), Properties: &n.PrivateLinkServiceConnectionProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateLinkServiceID: to.Ptr(row.service), GroupIDs: []*string{to.Ptr(row.group)}, PrivateLinkServiceConnectionState: &n.PrivateLinkServiceConnectionState{Status: to.Ptr("Approved"), ActionsRequired: to.Ptr("None")}}}}}}
		nic := &n.Interface{ID: to.Ptr(row.nic), Name: to.Ptr(nicName), Type: to.Ptr("Microsoft.Network/networkInterfaces"), Location: to.Ptr(c.Region), Properties: &n.InterfacePropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateEndpoint: &n.PrivateEndpoint{ID: to.Ptr(row.endpoint)}, EnableIPForwarding: to.Ptr(false), NicType: to.Ptr(n.NetworkInterfaceNicTypeStandard)}}
		for i, member := range row.members {
			name := "privateEndpointIpConfig." + member
			id := row.nic + "/ipConfigurations/" + name
			nic.Properties.IPConfigurations = append(nic.Properties.IPConfigurations, &n.InterfaceIPConfiguration{ID: to.Ptr(id), Name: to.Ptr(name), Properties: &n.InterfaceIPConfigurationPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateIPAddress: to.Ptr(row.ips[i]), PrivateIPAddressVersion: to.Ptr(n.IPVersionIPv4), PrivateIPAllocationMethod: to.Ptr(n.IPAllocationMethodDynamic), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, PrivateLinkConnectionProperties: &n.InterfaceIPConfigurationPrivateLinkConnectionProperties{GroupID: to.Ptr(row.group), RequiredMemberName: to.Ptr(member)}}})
			f.subnet.Properties.IPConfigurations = append(f.subnet.Properties.IPConfigurations, &n.IPConfiguration{ID: to.Ptr(id)})
		}
		f.endpoints[row.endpoint], f.interfaces[row.nic] = endpoint, nic
		f.subnet.Properties.PrivateEndpoints = append(f.subnet.Properties.PrivateEndpoints, &n.PrivateEndpoint{ID: to.Ptr(row.endpoint)})
	}
	reader, err := NewSDKEndpointReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	result, err := NewEndpointGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, result
}

func (f *endpointFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("endpoint SDK widened read authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	switch f.mode {
	case "unavailable":
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-provider-canary"}}), nil
	case "missing":
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	case "redirect":
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var body any
	switch {
	case f.endpoints[r.URL.Path] != nil:
		body = f.endpoints[r.URL.Path]
	case f.interfaces[r.URL.Path] != nil:
		body = f.interfaces[r.URL.Path]
	case r.URL.Path == f.settings.Subnet:
		body = f.subnet
	case r.URL.Path == f.settings.SecurityGroup:
		body = f.security
	default:
		f.t.Error("unowned endpoint resource requested")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, body)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}
