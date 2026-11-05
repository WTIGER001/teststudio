package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

type preparationNetworkFixture struct {
	t        *testing.T
	mu       sync.Mutex
	settings PreparationSettings
	o        preparationNetworkObservation
	host     *preparationHostFixture
	dns      *privateDNSFixture
	reads    int
	mode     string
}

func preparationNetworkDefaults(s PreparationSettings) PreparationNetworkSettings {
	c, r := s.Worker, s.Writer.Registry
	return PreparationNetworkSettings{SourceSecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "preparation-source"), NATGateway: c.resourceID("Microsoft.Network", "natGateways", "preparation-source"), NATPublicIP: c.resourceID("Microsoft.Network", "publicIPAddresses", "preparation-source"), DatabaseIP: "10.21.5.4", StorageAccount: c.resourceID("Microsoft.Storage", "storageAccounts", s.Host.JournalAccount), FileEndpoint: c.resourceID("Microsoft.Network", "privateEndpoints", "preparation-journal"), FileInterface: c.resourceID("Microsoft.Network", "networkInterfaces", "preparation-journal"), FileSubnet: r.network() + "/subnets/journal-endpoints", FileSubnetPrefix: "10.21.4.0/24", FileSecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "preparation-journal"), FileIP: "10.21.4.4", FileZone: c.resourceID("Microsoft.Network", "privateDnsZones", "privatelink.file.core.windows.net")}
}

func preparationNetworkTest(t *testing.T) (PreparationSettings, *preparationNetworkFixture, *PreparationNetworkGuard) {
	t.Helper()
	s, host, parent := preparationHostTest(t)
	p := preparationNetworkDefaults(s)
	s.Network = &p
	f := &preparationNetworkFixture{t: t, settings: s, host: host, dns: privateZoneFixture(t, []privateZoneBinding{p.binding(s)})}
	c, r, o := s.Worker, s.Writer.Registry, &f.o
	o.source, _ = s.PreparationSourceSubnetDefinition()
	o.source.ID, o.source.Name, o.source.Type = to.Ptr(r.SourceSubnet), to.Ptr("preparation"), to.Ptr("Microsoft.Network/virtualNetworks/subnets")
	o.source.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	o.source.Properties.Delegations[0].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	o.file, _ = s.PreparationFileSubnetDefinition()
	o.file.ID, o.file.Name, o.file.Type = to.Ptr(p.FileSubnet), to.Ptr("journal-endpoints"), to.Ptr("Microsoft.Network/virtualNetworks/subnets")
	o.file.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	o.file.Properties.PrivateEndpoints = []*n.PrivateEndpoint{{ID: to.Ptr(p.FileEndpoint)}}
	o.sourceNSG, _ = s.PreparationSourceSecurityGroupDefinition()
	o.fileNSG, _ = s.PreparationFileSecurityGroupDefinition()
	for _, entry := range []struct {
		group      *n.SecurityGroup
		id, subnet string
	}{{&o.sourceNSG, p.SourceSecurityGroup, r.SourceSubnet}, {&o.fileNSG, p.FileSecurityGroup, p.FileSubnet}} {
		_, name := resourceParts(entry.id)
		entry.group.ID, entry.group.Name, entry.group.Type = to.Ptr(entry.id), to.Ptr(name), to.Ptr("Microsoft.Network/networkSecurityGroups")
		entry.group.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
		entry.group.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(entry.subnet)}}
		for _, rule := range entry.group.Properties.SecurityRules {
			rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
		}
	}
	o.endpoint = n.PrivateEndpoint{ID: to.Ptr(p.FileEndpoint), Name: to.Ptr("preparation-journal"), Type: to.Ptr("Microsoft.Network/privateEndpoints"), Location: to.Ptr(c.Region), Properties: &n.PrivateEndpointProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), Subnet: &n.Subnet{ID: to.Ptr(p.FileSubnet)}, NetworkInterfaces: []*n.Interface{{ID: to.Ptr(p.FileInterface)}}, PrivateLinkServiceConnections: []*n.PrivateLinkServiceConnection{{Name: to.Ptr("journal-file"), Properties: &n.PrivateLinkServiceConnectionProperties{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateLinkServiceID: to.Ptr(p.StorageAccount), GroupIDs: []*string{to.Ptr("file")}, PrivateLinkServiceConnectionState: &n.PrivateLinkServiceConnectionState{Status: to.Ptr("Approved"), ActionsRequired: to.Ptr("None")}}}}}}
	ipID := p.FileInterface + "/ipConfigurations/file"
	o.nic = n.Interface{ID: to.Ptr(p.FileInterface), Name: to.Ptr("preparation-journal"), Type: to.Ptr("Microsoft.Network/networkInterfaces"), Location: to.Ptr(c.Region), Properties: &n.InterfacePropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateEndpoint: &n.PrivateEndpoint{ID: to.Ptr(p.FileEndpoint)}, IPConfigurations: []*n.InterfaceIPConfiguration{{ID: to.Ptr(ipID), Name: to.Ptr("file"), Properties: &n.InterfaceIPConfigurationPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), PrivateIPAddress: to.Ptr(p.FileIP), PrivateIPAddressVersion: to.Ptr(n.IPVersionIPv4), PrivateIPAllocationMethod: to.Ptr(n.IPAllocationMethodStatic), Subnet: &n.Subnet{ID: to.Ptr(p.FileSubnet)}, PrivateLinkConnectionProperties: &n.InterfaceIPConfigurationPrivateLinkConnectionProperties{GroupID: to.Ptr("file"), RequiredMemberName: to.Ptr("file")}}}}}}
	o.file.Properties.IPConfigurations = []*n.IPConfiguration{{ID: to.Ptr(ipID)}}
	o.nat, _ = s.PreparationNATDefinition()
	o.nat.ID, o.nat.Name, o.nat.Type = to.Ptr(p.NATGateway), to.Ptr("preparation-source"), to.Ptr("Microsoft.Network/natGateways")
	o.nat.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	o.nat.Properties.Subnets = []*n.SubResource{{ID: to.Ptr(r.SourceSubnet)}}
	o.ip, _ = s.PreparationPublicIPDefinition()
	o.ip.ID, o.ip.Name, o.ip.Type = to.Ptr(p.NATPublicIP), to.Ptr("preparation-source"), to.Ptr("Microsoft.Network/publicIPAddresses")
	o.ip.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	o.ip.Properties.IPAddress = to.Ptr("203.0.113.10")
	o.ip.Properties.NatGateway = &n.NatGateway{ID: to.Ptr(p.NATGateway)}
	previous := parent.writer.registry.reader.(*SDKPreparationRegistryReader).http.base.(*preparationRegistryFixture)
	previous.source = o.source
	for i, subnet := range previous.network.Properties.Subnets {
		if strings.EqualFold(value(subnet.ID), r.SourceSubnet) {
			previous.network.Properties.Subnets[i] = &o.source
		}
	}
	previous.network.Properties.Subnets = append(previous.network.Properties.Subnets, &o.file)
	o.network = previous.network
	connectionName := "journal-file"
	o.account = a.Account{ID: to.Ptr(p.StorageAccount), Name: to.Ptr(s.Host.JournalAccount), Type: to.Ptr("Microsoft.Storage/storageAccounts"), Location: to.Ptr(c.Region), Kind: to.Ptr(a.KindStorageV2), Properties: &a.AccountProperties{ProvisioningState: to.Ptr(a.ProvisioningStateSucceeded), EnableHTTPSTrafficOnly: to.Ptr(true), AllowSharedKeyAccess: to.Ptr(true), AllowBlobPublicAccess: to.Ptr(false), MinimumTLSVersion: to.Ptr(a.MinimumTLSVersionTLS12), PublicNetworkAccess: to.Ptr(a.PublicNetworkAccessDisabled), PrimaryEndpoints: &a.Endpoints{File: to.Ptr("https://" + s.Host.JournalAccount + ".file.core.windows.net/")}, NetworkRuleSet: &a.NetworkRuleSet{DefaultAction: to.Ptr(a.DefaultActionDeny), Bypass: to.Ptr(a.BypassNone)}, PrivateEndpointConnections: []*a.PrivateEndpointConnection{{ID: to.Ptr(p.StorageAccount + "/privateEndpointConnections/" + connectionName), Name: to.Ptr(connectionName), Type: to.Ptr("Microsoft.Storage/storageAccounts/privateEndpointConnections"), Properties: &a.PrivateEndpointConnectionProperties{ProvisioningState: to.Ptr(a.PrivateEndpointConnectionProvisioningStateSucceeded), PrivateEndpoint: &a.PrivateEndpoint{ID: to.Ptr(p.FileEndpoint)}, PrivateLinkServiceConnectionState: &a.PrivateLinkServiceConnectionState{Status: to.Ptr(a.PrivateEndpointServiceConnectionStatusApproved), ActionRequired: to.Ptr("None")}}}}}}
	o.share = a.FileShare{ID: to.Ptr(p.share(s)), Name: to.Ptr(s.Host.JournalShare), Type: to.Ptr("Microsoft.Storage/storageAccounts/fileServices/shares"), FileShareProperties: &a.FileShareProperties{EnabledProtocols: to.Ptr(a.EnabledProtocolsSMB), ShareQuota: to.Ptr[int32](100)}}
	reader, err := NewSDKPreparationNetworkReader(s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	dns, err := NewSDKPreparationFileDNSReader(s, fixtureCredential{}, f.dns)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dns.Close)
	g, err := NewPreparationNetworkGuard(s, parent, reader, dns)
	if err != nil {
		t.Fatal(err)
	}
	return s, f, g
}

func (f *preparationNetworkFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	p, o := f.settings.Network, &f.o
	resources := map[string]any{p.FileEndpoint: o.endpoint, p.FileInterface: o.nic, p.FileSubnet: o.file, p.FileSecurityGroup: o.fileNSG, f.settings.Writer.Registry.SourceSubnet: o.source, p.SourceSecurityGroup: o.sourceNSG, p.NATGateway: o.nat, p.NATPublicIP: o.ip, f.settings.Writer.Registry.network(): o.network, p.StorageAccount: o.account, p.share(f.settings): o.share}
	body, ok := resources[r.URL.Path]
	version := "2025-03-01"
	if r.URL.Path == p.StorageAccount || r.URL.Path == p.share(f.settings) {
		version = preparationStorageVersion
	}
	if !ok || r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version="+version || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("source and journal network reader widened authority")
		return deadlineResponse(r, 404, nil), nil
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"message": "private-source-network-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private-source-network-canary")
		return response, nil
	}
	return deadlineResponse(r, 200, body), nil
}
