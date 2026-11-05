package azuresandbox

import (
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p PlacementSettings) validateRegistryConsumers(c Config) error {
	if len(p.RegistryConsumers) > 6 {
		return hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	for _, id := range []string{p.Private.RegistryEndpoint, p.Services.RelayNetwork.Registry.Endpoint, p.Private.ExchangeEndpoint, p.Services.BackendEndpoint.Endpoint} {
		seen[strings.ToLower(id)] = true
	}
	for _, id := range p.RegistryConsumers {
		if !resourceID(c, id, "Microsoft.Network", "privateEndpoints") || seen[strings.ToLower(id)] {
			return hostedexecution.ErrConfiguration
		}
		seen[strings.ToLower(id)] = true
	}
	return nil
}

func (s Settings) registryBinding() azureworker.RegistryServiceBinding {
	p := s.Placement
	endpoints := append([]string{p.Private.RegistryEndpoint, p.Services.RelayNetwork.Registry.Endpoint}, p.RegistryConsumers...)
	slices.Sort(endpoints)
	return azureworker.RegistryServiceBinding{Registry: p.Registry, Login: s.Worker.Registry, Region: s.Worker.Region, Endpoints: endpoints}
}
