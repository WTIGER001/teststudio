package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RegistrySettings struct {
	Endpoints           EndpointSettings `json:"endpoints"`
	PreparationEndpoint string           `json:"preparationEndpoint"`
}

func (s RegistrySettings) canonical(c Config) (RegistrySettings, error) {
	var err error
	s.Endpoints, err = s.Endpoints.canonical(c)
	if err != nil || !c.ownsResource(s.PreparationEndpoint, "Microsoft.Network", "privateEndpoints") || strings.EqualFold(s.PreparationEndpoint, s.Endpoints.RegistryEndpoint) || strings.EqualFold(s.PreparationEndpoint, s.Endpoints.ExchangeEndpoint) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s RegistrySettings) registry() string    { return s.Endpoints.Registry }
func (s RegistrySettings) region() string      { return s.Endpoints.PrivateDNS.DNS.RegistryDataRegion }
func (s RegistrySettings) connections() string { return s.registry() + "/privateEndpointConnections" }
func (s RegistrySettings) links() string       { return s.registry() + "/privateLinkResources" }
func (s RegistrySettings) replicas() string    { return s.registry() + "/replications" }
func (s RegistrySettings) dataHost(c Config) string {
	return strings.TrimSuffix(c.Registry, ".azurecr.io") + "." + s.region() + ".data.azurecr.io"
}
