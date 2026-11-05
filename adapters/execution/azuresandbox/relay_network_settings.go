package azuresandbox

import (
	"encoding/binary"
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayRegistrySettings struct {
	Endpoint       string `json:"endpoint"`
	Interface      string `json:"interface"`
	Subnet         string `json:"subnet"`
	SubnetPrefix   string `json:"subnetPrefix"`
	SecurityGroup  string `json:"securityGroup"`
	LoginIP        string `json:"loginIp"`
	DataIP         string `json:"dataIp"`
	Zone           string `json:"zone"`
	PullAssignment string `json:"pullAssignment"`
	Connection     string `json:"connection"`
}

type RelayNetworkSettings struct {
	Topology      TopologySettings      `json:"topology"`
	FirewallIP    string                `json:"firewallIp"`
	SecurityGroup string                `json:"securityGroup"`
	RouteTable    string                `json:"routeTable"`
	Registry      RelayRegistrySettings `json:"registry"`
	Resolver      ResolverSettings      `json:"resolver"`
}

func usablePrivateAddress(prefix netip.Prefix, raw string) bool {
	ip, err := netip.ParseAddr(raw)
	if err != nil || !ip.Is4() || ip.String() != raw || !prefix.Contains(ip) {
		return false
	}
	start, host := prefix.Addr().As4(), ip.As4()
	offset := binary.BigEndian.Uint32(host[:]) - binary.BigEndian.Uint32(start[:])
	return offset >= 4 && offset != (uint32(1)<<(32-prefix.Bits()))-1
}

func (r RelayNetworkSettings) virtualNetwork(s Settings) string {
	return strings.Join(strings.Split(s.Placement.Services.Relay.Subnet, "/")[:9], "/")
}

func (r RelayNetworkSettings) validate(c Config, p PlacementSettings, v ServicesSettings) error {
	parent, valid := privateIPv4Prefix(r.Topology.Prefix)
	firewall, firewallValid := privateIPv4Prefix(r.Topology.FirewallPrefix)
	registry, registryValid := privateIPv4Prefix(r.Registry.SubnetPrefix)
	source, _ := netip.ParsePrefix(v.BackendEndpoint.RelaySubnetPrefix)
	backend, _ := netip.ParsePrefix(v.BackendEndpoint.SubnetPrefix)
	worker, _ := netip.ParsePrefix(p.Network.Topology.Prefix)
	if !valid || !firewallValid || !registryValid || parent.Overlaps(worker) || firewall.Bits() > 26 || registry.Bits() > 29 || !usablePrivateAddress(firewall, r.FirewallIP) || !usablePrivateAddress(registry, r.Registry.LoginIP) || !usablePrivateAddress(registry, r.Registry.DataIP) || r.Registry.LoginIP == r.Registry.DataIP || r.Resolver.validate(c) != nil {
		return hostedexecution.ErrConfiguration
	}
	prefixes := []netip.Prefix{source, backend, firewall, registry}
	for i, child := range prefixes {
		if child.Bits() < parent.Bits() || !parent.Contains(child.Addr()) {
			return hostedexecution.ErrConfiguration
		}
		for _, other := range prefixes[:i] {
			if child.Overlaps(other) {
				return hostedexecution.ErrConfiguration
			}
		}
	}
	for _, entry := range [][2]string{{r.Topology.Firewall, "azureFirewalls"}, {r.Topology.Policy, "firewallPolicies"}, {r.Topology.PublicIP, "publicIPAddresses"}, {r.SecurityGroup, "networkSecurityGroups"}, {r.RouteTable, "routeTables"}, {r.Registry.Endpoint, "privateEndpoints"}, {r.Registry.Interface, "networkInterfaces"}, {r.Registry.SecurityGroup, "networkSecurityGroups"}} {
		if !resourceID(c, entry[0], "Microsoft.Network", entry[1]) {
			return hostedexecution.ErrConfiguration
		}
	}
	parentID := strings.Join(strings.Split(v.Relay.Subnet, "/")[:9], "/")
	if !serviceSubnet(c, r.Registry.Subnet) || !strings.HasPrefix(r.Registry.Subnet, parentID+"/subnets/") || !zoneID(c, r.Registry.Zone, "privatelink.azurecr.io") || strings.EqualFold(r.Registry.Zone, p.Private.RegistryZone) {
		return hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	ids := []string{parentID + "/subnets/AzureFirewallSubnet", p.firewallSubnet(), p.Network.Topology.Firewall, p.Network.Topology.Policy, p.Network.Topology.PublicIP, p.SecurityGroup, p.RouteTable, p.Private.SecurityGroup, p.Private.RegistryEndpoint, p.Private.RegistryInterface, p.Private.ExchangeEndpoint, p.Private.ExchangeInterface, p.Subnet, p.Private.Subnet, v.BackendEndpoint.Endpoint, v.BackendEndpoint.Interface, v.BackendEndpoint.SecurityGroup, v.BackendEndpoint.Subnet, v.Relay.Subnet, p.Network.Resolver.Policy, p.Network.Resolver.AllowList, p.Network.Resolver.BlockList}
	for _, id := range ids {
		seen[strings.ToLower(id)] = true
	}
	for _, id := range []string{r.Topology.Firewall, r.Topology.Policy, r.Topology.PublicIP, r.SecurityGroup, r.RouteTable, r.Registry.Endpoint, r.Registry.Interface, r.Registry.SecurityGroup, r.Registry.Subnet, r.Resolver.Policy, r.Resolver.AllowList, r.Resolver.BlockList} {
		if seen[strings.ToLower(id)] {
			return hostedexecution.ErrConfiguration
		}
		seen[strings.ToLower(id)] = true
	}
	for _, entry := range [][2]string{{r.Registry.PullAssignment, "roleAssignments"}, {r.Registry.Connection, "privateEndpointConnections"}} {
		prefix := p.Registry + "/"
		if entry[1] == "roleAssignments" {
			prefix += "providers/Microsoft.Authorization/"
		}
		prefix += entry[1] + "/"
		name := strings.TrimPrefix(entry[0], prefix)
		if !strings.HasPrefix(entry[0], prefix) || !identifier.MatchString(name) || entry[1] == "roleAssignments" && (!nonzeroGUID(name) || strings.EqualFold(entry[0], p.PullAssignment)) {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}
