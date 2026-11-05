package azureworker

import (
	"net/netip"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayPerimeterSettings struct {
	Topology            RelayTopologySettings `json:"topology"`
	SecurityGroup       string                `json:"securityGroup"`
	RouteTable          string                `json:"routeTable"`
	FirewallIP          string                `json:"firewallIp"`
	RegistryEndpointIPs []string              `json:"registryEndpointIps"`
}

func (s RelayPerimeterSettings) canonical(c Config) (RelayPerimeterSettings, error) {
	var err error
	s.Topology, err = s.Topology.canonical(c)
	if err != nil || !c.ownsResource(s.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !c.ownsResource(s.RouteTable, "Microsoft.Network", "routeTables") || len(s.RegistryEndpointIPs) < 2 || len(s.RegistryEndpointIPs) > 8 {
		return s, hostedexecution.ErrConfiguration
	}
	endpoint := s.Topology.BackendEndpoint
	worker := endpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter
	for _, id := range []string{endpoint.SecurityGroup, endpoint.Backend.Exchange.Registry.Endpoints.SecurityGroup, worker.Topology.SecurityGroup} {
		if strings.EqualFold(id, s.SecurityGroup) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	if strings.EqualFold(s.RouteTable, worker.Topology.RouteTable) {
		return s, hostedexecution.ErrConfiguration
	}
	s.RegistryEndpointIPs = slices.Clone(s.RegistryEndpointIPs)
	slices.Sort(s.RegistryEndpointIPs)
	vnet, _ := netip.ParsePrefix(s.Topology.VirtualNetworkPrefix)
	source, _ := netip.ParsePrefix(endpoint.RelaySubnetPrefix)
	backend, _ := netip.ParsePrefix(endpoint.SubnetPrefix)
	seen := map[string]bool{}
	for _, raw := range append([]string{s.FirewallIP}, s.RegistryEndpointIPs...) {
		ip, ok := privateHost(raw)
		if !ok || !vnet.Contains(ip) || source.Contains(ip) || backend.Contains(ip) || seen[raw] {
			return s, hostedexecution.ErrConfiguration
		}
		seen[raw] = true
	}
	return s, nil
}
