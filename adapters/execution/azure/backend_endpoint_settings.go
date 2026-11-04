package azureworker

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type BackendEndpointSettings struct {
	Backend           BackendSettings `json:"backend"`
	Endpoint          string          `json:"endpoint"`
	Interface         string          `json:"interface"`
	Subnet            string          `json:"subnet"`
	SubnetPrefix      string          `json:"subnetPrefix"`
	RelaySubnetPrefix string          `json:"relaySubnetPrefix"`
	SecurityGroup     string          `json:"securityGroup"`
	Address           string          `json:"address"`
}

func (s BackendEndpointSettings) canonical(c Config) (BackendEndpointSettings, error) {
	var err error
	s.Backend, err = s.Backend.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	prefix, ok := privatePrefix(s.SubnetPrefix)
	source, sourceOK := privatePrefix(s.RelaySubnetPrefix)
	exchange := s.Backend.Exchange
	worker, _ := netip.ParsePrefix(exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNetPrefix)
	if !ok || !sourceOK || prefix.Bits() > 29 || source.Bits() > 27 || prefix.Overlaps(source) || prefix.Overlaps(worker) || source.Overlaps(worker) ||
		!usableEndpointAddress(prefix, s.Address) || !subnetResource(c, s.Subnet, exchange.VirtualNetwork) || strings.EqualFold(s.Subnet, exchange.Subnet) ||
		!c.ownsResource(s.Endpoint, "Microsoft.Network", "privateEndpoints") || !c.ownsResource(s.Interface, "Microsoft.Network", "networkInterfaces") ||
		!c.ownsResource(s.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") ||
		strings.EqualFold(s.SecurityGroup, exchange.Registry.Endpoints.SecurityGroup) || strings.EqualFold(s.SecurityGroup, exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.SecurityGroup) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s BackendEndpointSettings) role() endpointRole {
	return endpointRole{s.Endpoint, s.Interface, s.Backend.Environment, "managedEnvironment", map[string]string{"managedEnvironment": s.Address}}
}
