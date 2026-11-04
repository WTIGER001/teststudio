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
		ip, _ := netip.ParseAddr(raw)
		if !prefix.Contains(ip) || ip == lastIPv4(prefix) {
			return s, hostedexecution.ErrConfiguration
		}
		offset := uint32(ip.As4()[0])<<24 | uint32(ip.As4()[1])<<16 | uint32(ip.As4()[2])<<8 | uint32(ip.As4()[3])
		start := prefix.Addr().As4()
		base := uint32(start[0])<<24 | uint32(start[1])<<16 | uint32(start[2])<<8 | uint32(start[3])
		if offset-base < 4 {
			return s, hostedexecution.ErrConfiguration
		}
	}
	return s, nil
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
