package azureworker

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationNetworkSettings struct {
	SourceSecurityGroup string `json:"sourceSecurityGroup"`
	NATGateway          string `json:"natGateway"`
	NATPublicIP         string `json:"natPublicIp"`
	DatabaseIP          string `json:"databaseIp"`
	StorageAccount      string `json:"storageAccount"`
	FileEndpoint        string `json:"fileEndpoint"`
	FileInterface       string `json:"fileInterface"`
	FileSubnet          string `json:"fileSubnet"`
	FileSubnetPrefix    string `json:"fileSubnetPrefix"`
	FileSecurityGroup   string `json:"fileSecurityGroup"`
	FileIP              string `json:"fileIp"`
	FileZone            string `json:"fileZone"`
}

func (n PreparationNetworkSettings) canonical(s PreparationSettings) (PreparationNetworkSettings, error) {
	c, r := s.Worker, s.Writer.Registry
	if s.Host == nil || !c.ownsResource(n.SourceSecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !c.ownsResource(n.FileSecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !c.ownsResource(n.NATGateway, "Microsoft.Network", "natGateways") || !c.ownsResource(n.NATPublicIP, "Microsoft.Network", "publicIPAddresses") || !c.ownsResource(n.FileEndpoint, "Microsoft.Network", "privateEndpoints") || !c.ownsResource(n.FileInterface, "Microsoft.Network", "networkInterfaces") || !subnetResource(c, n.FileSubnet, r.network()) || !zoneResource(c, n.FileZone, "privatelink.file.core.windows.net") {
		return n, hostedexecution.ErrConfiguration
	}
	account := s.Host.JournalAccount
	if !strings.HasSuffix(n.StorageAccount, "/"+account) || !c.ownsResource(strings.TrimSuffix(n.StorageAccount, "/"+account)+"/journal-account", "Microsoft.Storage", "storageAccounts") {
		return n, hostedexecution.ErrConfiguration
	}
	t := r.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology
	for _, id := range []string{t.ControllerSubnet, r.SourceSubnet, r.Subnet} {
		if strings.EqualFold(n.FileSubnet, id) {
			return n, hostedexecution.ErrConfiguration
		}
	}
	seen := map[string]bool{}
	for _, id := range []string{n.SourceSecurityGroup, n.FileSecurityGroup, r.SecurityGroup, r.Registry.Endpoints.SecurityGroup, t.SecurityGroup} {
		key := strings.ToLower(id)
		if seen[key] {
			return n, hostedexecution.ErrConfiguration
		}
		seen[key] = true
	}
	for _, id := range []string{r.Registry.PreparationEndpoint, r.Registry.RelayEndpoint, r.Registry.Endpoints.RegistryEndpoint, r.Registry.Endpoints.ExchangeEndpoint} {
		if strings.EqualFold(n.FileEndpoint, id) {
			return n, hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{r.Interface, r.Registry.Endpoints.RegistryNIC, r.Registry.Endpoints.ExchangeNIC} {
		if strings.EqualFold(n.FileInterface, id) {
			return n, hostedexecution.ErrConfiguration
		}
	}
	file, ok := privatePrefix(n.FileSubnetPrefix)
	source, _ := netip.ParsePrefix(r.SourceSubnetPrefix)
	registry, _ := netip.ParsePrefix(r.SubnetPrefix)
	worker, _ := netip.ParsePrefix(t.WorkerVNetPrefix)
	database, valid := privateHost(n.DatabaseIP)
	if !ok || file.Bits() > 29 || file.Overlaps(source) || file.Overlaps(registry) || file.Overlaps(worker) || !usableEndpointAddress(file, n.FileIP) || !valid || source.Contains(database) || registry.Contains(database) || file.Contains(database) || worker.Contains(database) {
		return n, hostedexecution.ErrConfiguration
	}
	containedFile, containedDatabase := false, false
	for _, raw := range r.ControllerPrefixes {
		space, _ := netip.ParsePrefix(raw)
		containedFile = containedFile || containsPrefix(space, file)
		containedDatabase = containedDatabase || usableEndpointAddress(space, n.DatabaseIP)
	}
	if !containedFile || !containedDatabase {
		return n, hostedexecution.ErrConfiguration
	}
	return n, nil
}

func (n PreparationNetworkSettings) role() endpointRole {
	return endpointRole{n.FileEndpoint, n.FileInterface, n.StorageAccount, "file", map[string]string{"file": n.FileIP}}
}

func (n PreparationNetworkSettings) share(s PreparationSettings) string {
	return n.StorageAccount + "/fileServices/default/shares/" + s.Host.JournalShare
}

func (n PreparationNetworkSettings) binding(s PreparationSettings) privateZoneBinding {
	return privateZoneBinding{ID: n.FileZone, VirtualNetwork: s.Writer.Registry.network(), LinkName: "preparation-journal", Records: map[string]string{s.Host.JournalAccount: n.FileIP}, ResolutionPolicy: true}
}
