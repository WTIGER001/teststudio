package azuresandbox

import (
	"net/netip"
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const postgresPrivateZone = "privatelink.postgres.database.azure.com"

type HostDatabaseSettings struct {
	Server        string   `json:"server"`
	Endpoint      string   `json:"endpoint"`
	Interface     string   `json:"interface"`
	Subnet        string   `json:"subnet"`
	SubnetPrefix  string   `json:"subnetPrefix"`
	SecurityGroup string   `json:"securityGroup"`
	Address       string   `json:"address"`
	Zone          string   `json:"zone"`
	Consumers     []string `json:"consumers,omitempty"`
}

func (d HostDatabaseSettings) name() string { return d.Server[strings.LastIndex(d.Server, "/")+1:] }

func (h HostSettings) validateDatabase(s Settings) error {
	d, c := h.Database, s.Worker
	prefix, valid := privateIPv4Prefix(d.SubnetPrefix)
	parent, _ := netip.ParsePrefix(h.Network.Topology.Prefix)
	if !resourceID(c, d.Server, "Microsoft.DBforPostgreSQL", "flexibleServers") || len(d.name()) < 3 || len(d.name()) > 63 || !resourceName.MatchString(d.name()) || h.DatabaseHost != d.name()+".postgres.database.azure.com" || !valid || prefix.Bits() > 29 || !usablePrivateAddress(prefix, d.Address) || prefix.Bits() < parent.Bits() || !parent.Contains(prefix.Addr()) || !serviceSubnet(c, d.Subnet) || !strings.HasPrefix(d.Subnet, h.virtualNetwork()+"/subnets/") || !zoneID(c, d.Zone, postgresPrivateZone) || len(d.Consumers) > 7 {
		return hostedexecution.ErrConfiguration
	}
	for _, raw := range []string{h.SubnetPrefix, h.Registry.SubnetPrefix, h.Network.Topology.FirewallPrefix} {
		other, _ := netip.ParsePrefix(raw)
		if prefix.Overlaps(other) {
			return hostedexecution.ErrConfiguration
		}
	}
	seen := map[string]bool{}
	p, r := s.Placement, s.Placement.Services.RelayNetwork
	for _, id := range []string{h.Registry.Endpoint, h.Registry.Interface, h.Registry.SecurityGroup, h.Registry.Zone, p.Private.RegistryZone, p.Private.ExchangeZone, r.Registry.Zone, h.Subnet, h.Registry.Subnet, h.firewallBinding().Subnet, h.Network.SecurityGroup, p.Private.RegistryEndpoint, p.Private.RegistryInterface, p.Private.ExchangeEndpoint, p.Private.ExchangeInterface, p.Private.SecurityGroup, p.SecurityGroup, p.Services.BackendEndpoint.Endpoint, p.Services.BackendEndpoint.Interface, p.Services.BackendEndpoint.SecurityGroup, r.Registry.Endpoint, r.Registry.Interface, r.Registry.SecurityGroup, r.SecurityGroup} {
		seen[strings.ToLower(id)] = true
	}
	for _, id := range p.RegistryConsumers {
		seen[strings.ToLower(id)] = true
	}
	for _, entry := range [][2]string{{d.Endpoint, "privateEndpoints"}, {d.Interface, "networkInterfaces"}, {d.SecurityGroup, "networkSecurityGroups"}} {
		if !resourceID(c, entry[0], "Microsoft.Network", entry[1]) {
			return hostedexecution.ErrConfiguration
		}
	}
	for _, id := range append([]string{d.Endpoint, d.Interface, d.Subnet, d.SecurityGroup, d.Zone}, d.Consumers...) {
		key := strings.ToLower(id)
		if seen[key] {
			return hostedexecution.ErrConfiguration
		}
		seen[key] = true
	}
	for _, id := range d.Consumers {
		if !resourceID(c, id, "Microsoft.Network", "privateEndpoints") {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}

func (d HostDatabaseSettings) consumers() []string {
	result := append([]string{d.Endpoint}, d.Consumers...)
	slices.Sort(result)
	return result
}

func (h HostSettings) databaseZone() azureworker.PrivateZoneBinding {
	return azureworker.PrivateZoneBinding{ID: h.Database.Zone, VirtualNetwork: h.virtualNetwork(), LinkName: "controller-database", ResolutionPolicy: true, Records: map[string]string{h.Database.name(): h.Database.Address}}
}

func (h HostSettings) databaseRules() []any {
	return []any{networkRule("controller-database-postgres", "Inbound", "Allow", "Tcp", h.SubnetPrefix, h.Database.Address, "5432", 100), networkRule("endpoint-deny-in", "Inbound", "Deny", "*", "*", "*", "*", 4096), networkRule("endpoint-deny-out", "Outbound", "Deny", "*", "*", "*", "*", 4096)}
}

func (s Settings) HostDatabaseDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	h, d := s.Host, s.Host.Database
	zone := h.databaseZone()
	result := map[string]any{
		d.SecurityGroup: map[string]any{"location": s.Worker.Region, "properties": map[string]any{"flushConnection": true, "securityRules": h.databaseRules()}},
		d.Subnet:        map[string]any{"properties": map[string]any{"addressPrefix": d.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": d.SecurityGroup}, "privateEndpointNetworkPolicies": "Enabled", "privateLinkServiceNetworkPolicies": "Enabled"}},
		zone.ID:         map[string]any{"location": "global"},
		zone.ID + "/virtualNetworkLinks/" + zone.LinkName: zone.LinkDefinition(),
	}
	for name, record := range zone.RecordDefinitions() {
		result[zone.ID+"/A/"+name] = record
	}
	return result, nil
}
