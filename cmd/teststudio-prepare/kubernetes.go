package main

import (
	"github.com/wtiger001/teststudio/adapters/execution/kubernetes"
	"github.com/wtiger001/teststudio/adapters/preparation/registryimage"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func configureRegistryPreparation(service *sources.Service, settings kubernetes.Settings, directory string, runtimes map[string]string, lookup func(string) (string, bool)) (func(), error) {
	registry, err := settings.OpenRegistry(lookup)
	if err != nil {
		return nil, err
	}
	for _, image := range runtimes {
		if image != "" && registry.ValidateReference(image) != nil {
			registry.Close()
			return nil, hostedexecution.ErrConfiguration
		}
	}
	provider, err := settings.Open(service.Store, lookup, service.AuthorizeDiscoveryCleanup, nil)
	if err != nil {
		registry.Close()
		return nil, err
	}
	if err := attachRegistryPreparation(service, provider, registry, "kubernetes", settings.Worker.ExchangeOrigin, directory, runtimes); err != nil {
		provider.Close()
		registry.Close()
		return nil, err
	}
	return func() { provider.Close(); registry.Close() }, nil
}

func attachRegistryPreparation(service *sources.Service, provider hostedexecution.Provider, registry *oci.Client, name, origin, directory string, runtimes map[string]string) error {
	for _, image := range runtimes {
		if image != "" && registry.ValidateReference(image) != nil {
			return hostedexecution.ErrConfiguration
		}
	}

	exchange := &workerexchange.Service{Store: service.Store, Authorize: service.AuthorizeDiscovery, AuthorizeDiscoveryCleanup: service.AuthorizeDiscoveryCleanup}
	for _, runner := range []string{"go", "playwright"} {
		inspector := &hostedexecution.DiscoveryInspector{RequireGatewayAddress: name == "ecs", Provider: provider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin}
		builder := &registryimage.Builder{RequireGatewayAddress: name == "ecs", Registry: registry, RuntimeImage: runtimes[runner], Directory: directory}
		service.RecoveryInspectors[runner] = inspector
		service.RecoveryBuilders[runner] = builder
		if runtimes[runner] != "" {
			service.Inspectors[runner], service.Builders[runner] = inspector, builder
		}
	}
	return nil
}
