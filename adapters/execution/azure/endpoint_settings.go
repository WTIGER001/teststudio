package azureworker

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type EndpointSettings struct {
	PrivateDNS          PrivateDNSSettings `json:"privateDns"`
	Subnet              string             `json:"subnet"`
	SubnetPrefix        string             `json:"subnetPrefix"`
	SecurityGroup       string             `json:"securityGroup"`
	Registry            string             `json:"registry"`
	RegistryEndpoint    string             `json:"registryEndpoint"`
	RegistryNIC         string             `json:"registryNic"`
	ExchangeEnvironment string             `json:"exchangeEnvironment"`
	ExchangeEndpoint    string             `json:"exchangeEndpoint"`
	ExchangeNIC         string             `json:"exchangeNic"`
}

func (s EndpointSettings) canonical(c Config) (EndpointSettings, error) {
	var err error
	s.PrivateDNS, err = s.PrivateDNS.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	topology := s.PrivateDNS.DNS.Firewall.Perimeter.Topology
	prefix, valid := privatePrefix(s.SubnetPrefix)
	vnet, _ := netip.ParsePrefix(topology.WorkerVNetPrefix)
	worker, _ := netip.ParsePrefix(topology.WorkerSubnetPrefix)
	firewall, _ := netip.ParsePrefix(s.PrivateDNS.DNS.Firewall.SubnetPrefix)
	if !valid ||
		prefix.Bits() > 29 ||
		prefix.Bits() < vnet.Bits() ||
		!vnet.Contains(prefix.Addr()) ||
		prefix.Overlaps(worker) ||
		prefix.Overlaps(firewall) ||
		!subnetResource(c, s.Subnet, topology.WorkerVNet) ||
		!c.ownsResource(s.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") ||
		strings.EqualFold(s.SecurityGroup, topology.SecurityGroup) ||
		!c.ownsResource(s.Registry, "Microsoft.ContainerRegistry", "registries") ||
		!strings.HasSuffix(s.Registry, "/"+strings.TrimSuffix(c.Registry, ".azurecr.io")) ||
		!c.ownsResource(s.ExchangeEnvironment, "Microsoft.App", "managedEnvironments") ||
		strings.EqualFold(s.ExchangeEnvironment, c.WorkerEnvironment) ||
		strings.EqualFold(s.ExchangeEnvironment, c.ControllerEnvironment) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, pair := range [][3]string{{s.RegistryEndpoint, s.ExchangeEndpoint, "privateEndpoints"}, {s.RegistryNIC, s.ExchangeNIC, "networkInterfaces"}} {
		kind := pair[2]
		if !c.ownsResource(pair[0], "Microsoft.Network", kind) || !c.ownsResource(pair[1], "Microsoft.Network", kind) || strings.EqualFold(pair[0], pair[1]) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	for _, raw := range []string{s.PrivateDNS.ExchangeIP, s.PrivateDNS.RegistryIP, s.PrivateDNS.DataIP} {
		if !usableEndpointAddress(prefix, raw) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	return s, nil
}

func usableEndpointAddress(prefix netip.Prefix, raw string) bool {
	ip, valid := privateHost(raw)
	if !valid || !prefix.IsValid() || !prefix.Addr().Is4() || !prefix.Contains(ip) || ip == lastIPv4(prefix) {
		return false
	}
	address, start := ip.As4(), prefix.Addr().As4()
	offset := uint32(address[0])<<24 | uint32(address[1])<<16 | uint32(address[2])<<8 | uint32(address[3])
	base := uint32(start[0])<<24 | uint32(start[1])<<16 | uint32(start[2])<<8 | uint32(start[3])
	return offset-base >= 4
}

func lastIPv4(p netip.Prefix) netip.Addr {
	a := p.Addr().As4()
	v := uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
	v |= uint32(0xffffffff) >> p.Bits()
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

type endpointRole struct {
	endpoint, nic, service, group string
	members                       map[string]string
}

func (s EndpointSettings) roles() []endpointRole {
	return []endpointRole{
		{s.RegistryEndpoint, s.RegistryNIC, s.Registry, "registry", map[string]string{"registry": s.PrivateDNS.RegistryIP, "registry_data_" + s.PrivateDNS.DNS.RegistryDataRegion: s.PrivateDNS.DataIP}},
		{s.ExchangeEndpoint, s.ExchangeNIC, s.ExchangeEnvironment, "managedEnvironment", map[string]string{"managedEnvironment": s.PrivateDNS.ExchangeIP}},
	}
}
