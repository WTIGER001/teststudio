package azureworker

import (
	"net/netip"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayRegistryEndpointSettings struct {
	DNS           RelayDNSSettings `json:"dns"`
	Endpoint      string           `json:"endpoint"`
	Interface     string           `json:"interface"`
	Subnet        string           `json:"subnet"`
	SubnetPrefix  string           `json:"subnetPrefix"`
	SecurityGroup string           `json:"securityGroup"`
	RegistryIP    string           `json:"registryIp"`
	DataIP        string           `json:"dataIp"`
}

func (s RelayRegistryEndpointSettings) canonical(c Config) (RelayRegistryEndpointSettings, error) {
	var err error
	s.DNS, err = s.DNS.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	firewall := s.DNS.Firewall
	perimeter := firewall.Perimeter
	topology := perimeter.Topology
	backend := topology.BackendEndpoint
	registry := backend.Backend.Exchange.Registry
	if s.Endpoint == "" || s.Endpoint != registry.RelayEndpoint || strings.EqualFold(s.Endpoint, backend.Endpoint) || !c.ownsResource(s.Interface, "Microsoft.Network", "networkInterfaces") || !c.ownsResource(s.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !subnetResource(c, s.Subnet, backend.Backend.Exchange.VirtualNetwork) || strings.EqualFold(s.Subnet, backend.Subnet) || strings.EqualFold(s.Subnet, backend.Backend.Exchange.Subnet) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, id := range []string{backend.Interface, registry.Endpoints.RegistryNIC, registry.Endpoints.ExchangeNIC} {
		if strings.EqualFold(s.Interface, id) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{perimeter.SecurityGroup, backend.SecurityGroup, registry.Endpoints.SecurityGroup, registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.SecurityGroup} {
		if strings.EqualFold(s.SecurityGroup, id) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	prefix, valid := privatePrefix(s.SubnetPrefix)
	vnet, _ := netip.ParsePrefix(topology.VirtualNetworkPrefix)
	if !valid || prefix.Bits() > 29 || !containsPrefix(vnet, prefix) || s.RegistryIP == s.DataIP || !usableEndpointAddress(prefix, s.RegistryIP) || !usableEndpointAddress(prefix, s.DataIP) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, raw := range []string{backend.SubnetPrefix, backend.RelaySubnetPrefix, firewall.SubnetPrefix} {
		other, _ := netip.ParsePrefix(raw)
		if prefix.Overlaps(other) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	addresses := []string{s.RegistryIP, s.DataIP}
	slices.Sort(addresses)
	if !slices.Equal(addresses, perimeter.RegistryEndpointIPs) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s RelayRegistryEndpointSettings) role() endpointRole {
	registry := s.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry
	return endpointRole{s.Endpoint, s.Interface, registry.registry(), "registry", map[string]string{"registry": s.RegistryIP, "registry_data_" + registry.region(): s.DataIP}}
}
