package azureworker

import (
	"net/netip"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationRegistrySettings struct {
	Registry           RegistrySettings `json:"registry"`
	ControllerPrefixes []string         `json:"controllerPrefixes"`
	Interface          string           `json:"interface"`
	Subnet             string           `json:"subnet"`
	SubnetPrefix       string           `json:"subnetPrefix"`
	SecurityGroup      string           `json:"securityGroup"`
	SourceSubnet       string           `json:"sourceSubnet"`
	SourceSubnetPrefix string           `json:"sourceSubnetPrefix"`
	RegistryIP         string           `json:"registryIp"`
	DataIP             string           `json:"dataIp"`
	RegistryZone       string           `json:"registryZone"`
}

func (s PreparationRegistrySettings) canonical(c Config) (PreparationRegistrySettings, error) {
	var err error
	s.Registry, err = s.Registry.canonical(c)
	if err != nil {
		return s, err
	}
	t := s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology
	if !c.ownsResource(s.Interface, "Microsoft.Network", "networkInterfaces") || !c.ownsResource(s.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !subnetResource(c, s.Subnet, t.ControllerVNet) || !subnetResource(c, s.SourceSubnet, t.ControllerVNet) || strings.EqualFold(s.Subnet, s.SourceSubnet) || strings.EqualFold(s.Subnet, t.ControllerSubnet) || strings.EqualFold(s.SourceSubnet, t.ControllerSubnet) || !zoneResource(c, s.RegistryZone, "privatelink.azurecr.io") || strings.EqualFold(s.RegistryZone, s.Registry.Endpoints.PrivateDNS.RegistryZone) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, id := range []string{s.Registry.Endpoints.RegistryNIC, s.Registry.Endpoints.ExchangeNIC} {
		if strings.EqualFold(id, s.Interface) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{t.SecurityGroup, s.Registry.Endpoints.SecurityGroup} {
		if strings.EqualFold(id, s.SecurityGroup) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	endpoint, ok := privatePrefix(s.SubnetPrefix)
	source, valid := privatePrefix(s.SourceSubnetPrefix)
	if !ok || !valid || endpoint.Bits() > 29 || source.Bits() > 27 || endpoint.Overlaps(source) || s.RegistryIP == s.DataIP || !usableEndpointAddress(endpoint, s.RegistryIP) || !usableEndpointAddress(endpoint, s.DataIP) {
		return s, hostedexecution.ErrConfiguration
	}
	if len(s.ControllerPrefixes) == 0 || len(s.ControllerPrefixes) > 8 {
		return s, hostedexecution.ErrConfiguration
	}
	s.ControllerPrefixes = slices.Clone(s.ControllerPrefixes)
	slices.Sort(s.ControllerPrefixes)
	spaces := []netip.Prefix{}
	worker, _ := netip.ParsePrefix(t.WorkerVNetPrefix)
	for _, raw := range s.ControllerPrefixes {
		space, valid := privatePrefix(raw)
		if !valid || space.Overlaps(worker) {
			return s, hostedexecution.ErrConfiguration
		}
		for _, existing := range spaces {
			if existing.Overlaps(space) {
				return s, hostedexecution.ErrConfiguration
			}
		}
		spaces = append(spaces, space)
	}
	for _, prefix := range []netip.Prefix{endpoint, source} {
		found := false
		for _, space := range spaces {
			if containsPrefix(space, prefix) {
				found = true
			}
		}
		if !found {
			return s, hostedexecution.ErrConfiguration
		}
	}
	if endpoint.Overlaps(worker) || source.Overlaps(worker) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s PreparationRegistrySettings) network() string {
	return s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerVNet
}
func (s PreparationRegistrySettings) role() endpointRole {
	return endpointRole{s.Registry.PreparationEndpoint, s.Interface, s.Registry.registry(), "registry", map[string]string{"registry": s.RegistryIP, "registry_data_" + s.Registry.region(): s.DataIP}}
}
func (s PreparationRegistrySettings) binding(c Config) privateZoneBinding {
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	return privateZoneBinding{ID: s.RegistryZone, VirtualNetwork: s.network(), LinkName: "preparation-network", ResolutionPolicy: true, Records: map[string]string{name: s.RegistryIP, name + "." + s.Registry.region() + ".data": s.DataIP}}
}
