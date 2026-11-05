package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type preparationRegistryFixture struct {
	t              *testing.T
	mu             sync.Mutex
	settings       PreparationRegistrySettings
	registry       *registryFixture
	dns            *privateDNSFixture
	endpoint       n.PrivateEndpoint
	nic            n.Interface
	subnet, source n.Subnet
	security       n.SecurityGroup
	network        n.VirtualNetwork
	reads          int
	mode           string
}

func preparationRegistryTest(t *testing.T) (Config, PreparationRegistrySettings, *preparationRegistryFixture, *PreparationRegistryGuard) {
	t.Helper()
	c, registry, parent, prior := registryServiceTest(t)
	topology := registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology
	zone := strings.Replace(c.resourceID("Microsoft.Network", "privateDnsZones", "privatelink.azurecr.io"), "/resourceGroups/"+c.ResourceGroup+"/", "/resourceGroups/preparation-private-dns/", 1)
	s := PreparationRegistrySettings{ControllerPrefixes: []string{"10.21.0.0/16"}, Registry: registry, Interface: c.resourceID("Microsoft.Network", "networkInterfaces", "preparation-registry-nic"), Subnet: topology.ControllerVNet + "/subnets/preparation-endpoints", SubnetPrefix: "10.21.3.0/24", SourceSubnet: topology.ControllerVNet + "/subnets/preparation", SourceSubnetPrefix: "10.21.2.0/24", SecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "preparation-registry"), RegistryIP: "10.21.3.4", DataIP: "10.21.3.5", RegistryZone: zone}
	f := &preparationRegistryFixture{t: t, settings: s, registry: parent, dns: privateZoneFixture(t, []privateZoneBinding{s.binding(c)})}
	f.subnet, _ = s.SubnetDefinition(c)
	f.subnet.ID, f.subnet.Name, f.subnet.Type = to.Ptr(s.Subnet), to.Ptr("preparation-endpoints"), to.Ptr("Microsoft.Network/virtualNetworks/subnets")
	f.subnet.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.subnet.Properties.PrivateEndpoints = []*n.PrivateEndpoint{{ID: to.Ptr(s.Registry.PreparationEndpoint)}}
	f.source = n.Subnet{ID: to.Ptr(s.SourceSubnet), Name: to.Ptr("preparation"), Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"), Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.SourceSubnetPrefix), ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded)}}
	f.security, _ = s.SecurityGroupDefinition(c)
	f.security.ID, f.security.Name, f.security.Type = to.Ptr(s.SecurityGroup), to.Ptr("preparation-registry"), to.Ptr("Microsoft.Network/networkSecurityGroups")
	f.security.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.security.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(s.Subnet)}}
	for _, rule := range f.security.Properties.SecurityRules {
		rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	}
	role := s.role()
	f.endpoint = n.PrivateEndpoint{ID: to.Ptr(role.endpoint), Name: to.Ptr("preparation-registry"), Type: to.Ptr("Microsoft.Network/privateEndpoints"), Location: to.Ptr(c.Region), Properties: &n.PrivateEndpointProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, NetworkInterfaces: []*n.Interface{{ID: to.Ptr(s.Interface)}}, PrivateLinkServiceConnections: []*n.PrivateLinkServiceConnection{{Name: to.Ptr("registry-connection"), Properties: &n.PrivateLinkServiceConnectionProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateLinkServiceID: to.Ptr(role.service), GroupIDs: []*string{to.Ptr("registry")}, PrivateLinkServiceConnectionState: &n.PrivateLinkServiceConnectionState{Status: to.Ptr("Approved"), ActionsRequired: to.Ptr("None")}}}}}}
	f.nic = n.Interface{ID: to.Ptr(s.Interface), Name: to.Ptr("preparation-registry-nic"), Type: to.Ptr("Microsoft.Network/networkInterfaces"), Location: to.Ptr(c.Region), Properties: &n.InterfacePropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateEndpoint: &n.PrivateEndpoint{ID: to.Ptr(role.endpoint)}}}
	for _, member := range []string{"registry", "registry_data_" + registry.region()} {
		name := strings.ReplaceAll(member, "_", "-")
		id := s.Interface + "/ipConfigurations/" + name
		f.nic.Properties.IPConfigurations = append(f.nic.Properties.IPConfigurations, &n.InterfaceIPConfiguration{ID: to.Ptr(id), Name: to.Ptr(name), Properties: &n.InterfaceIPConfigurationPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateIPAddress: to.Ptr(role.members[member]), PrivateIPAddressVersion: to.Ptr(n.IPVersionIPv4), PrivateIPAllocationMethod: to.Ptr(n.IPAllocationMethodDynamic), Subnet: &n.Subnet{ID: to.Ptr(s.Subnet)}, PrivateLinkConnectionProperties: &n.InterfaceIPConfigurationPrivateLinkConnectionProperties{GroupID: to.Ptr("registry"), RequiredMemberName: to.Ptr(member)}}})
		f.subnet.Properties.IPConfigurations = append(f.subnet.Properties.IPConfigurations, &n.IPConfiguration{ID: to.Ptr(id)})
	}
	base := prior.endpoints.dns.dns.firewall.perimeter.topology.reader.(*SDKTopologyReader).http.base.(*topologyFixture)
	f.network = clone(base.cnet)
	f.network.Properties.Subnets = append(f.network.Properties.Subnets, to.Ptr(clone(f.subnet)), to.Ptr(clone(f.source)))
	reader, err := NewSDKPreparationRegistryReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	dns, err := NewSDKPreparationRegistryDNSReader(c, s, fixtureCredential{}, f.dns)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dns.Close)
	g, err := NewPreparationRegistryGuard(c, s, prior, reader, dns)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *preparationRegistryFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("preparation registry network read widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"message": "private-preparation-registry-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private-preparation-canary")
		return response, nil
	}
	resources := map[string]any{f.settings.Registry.PreparationEndpoint: f.endpoint, f.settings.Interface: f.nic, f.settings.Subnet: f.subnet, f.settings.SourceSubnet: f.source, f.settings.SecurityGroup: f.security, f.settings.network(): f.network}
	body, ok := resources[r.URL.Path]
	if !ok {
		f.t.Error("unselected preparation resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	return deadlineResponse(r, 200, body), nil
}
