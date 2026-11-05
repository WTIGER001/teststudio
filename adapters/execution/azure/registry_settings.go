package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RegistrySettings struct {
	Endpoints           EndpointSettings `json:"endpoints"`
	PreparationEndpoint string           `json:"preparationEndpoint"`
	RelayEndpoint       string           `json:"relayEndpoint,omitempty"`
}

func (s RegistrySettings) canonical(c Config) (RegistrySettings, error) {
	var err error
	s.Endpoints, err = s.Endpoints.canonical(c)
	if err != nil || !c.ownsResource(s.PreparationEndpoint, "Microsoft.Network", "privateEndpoints") || strings.EqualFold(s.PreparationEndpoint, s.Endpoints.RegistryEndpoint) || strings.EqualFold(s.PreparationEndpoint, s.Endpoints.ExchangeEndpoint) {
		return s, hostedexecution.ErrConfiguration
	}
	if s.RelayEndpoint != "" {
		if !c.ownsResource(s.RelayEndpoint, "Microsoft.Network", "privateEndpoints") {
			return s, hostedexecution.ErrConfiguration
		}
		for _, id := range []string{s.Endpoints.RegistryEndpoint, s.Endpoints.ExchangeEndpoint, s.PreparationEndpoint} {
			if strings.EqualFold(s.RelayEndpoint, id) {
				return s, hostedexecution.ErrConfiguration
			}
		}
	}
	return s, nil
}

func (s RegistrySettings) consumers() []string {
	ids := []string{s.Endpoints.RegistryEndpoint, s.PreparationEndpoint}
	if s.RelayEndpoint != "" {
		ids = append(ids, s.RelayEndpoint)
	}
	return ids
}

func (s RegistrySettings) registry() string    { return s.Endpoints.Registry }
func (s RegistrySettings) region() string      { return s.Endpoints.PrivateDNS.DNS.RegistryDataRegion }
func (s RegistrySettings) connections() string { return s.registry() + "/privateEndpointConnections" }
func (s RegistrySettings) links() string       { return s.registry() + "/privateLinkResources" }
func (s RegistrySettings) replicas() string    { return s.registry() + "/replications" }
func (s RegistrySettings) dataHost(c Config) string {
	return strings.TrimSuffix(c.Registry, ".azurecr.io") + "." + s.region() + ".data.azurecr.io"
}
