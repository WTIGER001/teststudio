package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayPrivateDNSSettings struct {
	Endpoint     RelayRegistryEndpointSettings `json:"endpoint"`
	RegistryZone string                        `json:"registryZone"`
}

func (s RelayPrivateDNSSettings) canonical(c Config) (RelayPrivateDNSSettings, error) {
	var err error
	s.Endpoint, err = s.Endpoint.canonical(c)
	if err != nil || !zoneResource(c, s.RegistryZone, "privatelink.azurecr.io") {
		return s, hostedexecution.ErrConfiguration
	}
	worker := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS
	if strings.EqualFold(s.RegistryZone, worker.RegistryZone) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s RelayPrivateDNSSettings) binding(c Config) privateZoneBinding {
	registry := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	return privateZoneBinding{ID: s.RegistryZone, VirtualNetwork: s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork, LinkName: "relay-network", ResolutionPolicy: true, Records: map[string]string{name: s.Endpoint.RegistryIP, name + "." + registry.region() + ".data": s.Endpoint.DataIP}}
}
