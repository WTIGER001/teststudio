package azureworker

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayFirewallSettings struct {
	Perimeter    RelayPerimeterSettings `json:"perimeter"`
	Firewall     string                 `json:"firewall"`
	Policy       string                 `json:"policy"`
	PublicIP     string                 `json:"publicIp"`
	SubnetPrefix string                 `json:"subnetPrefix"`
}

func (s RelayFirewallSettings) binding() firewallBinding {
	return firewallBinding{Firewall: s.Firewall, Policy: s.Policy, PublicIP: s.PublicIP, Subnet: s.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork + "/subnets/AzureFirewallSubnet", SubnetPrefix: s.SubnetPrefix, Address: s.Perimeter.FirewallIP, SourcePrefix: s.Perimeter.Topology.BackendEndpoint.RelaySubnetPrefix, Role: "relay"}
}

func (s RelayFirewallSettings) canonical(c Config) (RelayFirewallSettings, error) {
	var err error
	s.Perimeter, err = s.Perimeter.canonical(c)
	if err != nil || !c.ownsResource(s.Firewall, "Microsoft.Network", "azureFirewalls") || !c.ownsResource(s.Policy, "Microsoft.Network", "firewallPolicies") || !c.ownsResource(s.PublicIP, "Microsoft.Network", "publicIPAddresses") {
		return s, hostedexecution.ErrConfiguration
	}
	topology := s.Perimeter.Topology
	worker := topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall
	if strings.EqualFold(s.Firewall, worker.Firewall) || strings.EqualFold(s.Policy, worker.Policy) || strings.EqualFold(s.PublicIP, worker.PublicIP) {
		return s, hostedexecution.ErrConfiguration
	}
	prefix, valid := privatePrefix(s.SubnetPrefix)
	vnet, _ := netip.ParsePrefix(topology.VirtualNetworkPrefix)
	source, _ := netip.ParsePrefix(topology.BackendEndpoint.RelaySubnetPrefix)
	backend, _ := netip.ParsePrefix(topology.BackendEndpoint.SubnetPrefix)
	if !valid || prefix.Bits() > 26 || !containsPrefix(vnet, prefix) || prefix.Overlaps(source) || prefix.Overlaps(backend) || !usableEndpointAddress(prefix, s.Perimeter.FirewallIP) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, raw := range s.Perimeter.RegistryEndpointIPs {
		ip, _ := netip.ParseAddr(raw)
		if prefix.Contains(ip) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	return s, nil
}
