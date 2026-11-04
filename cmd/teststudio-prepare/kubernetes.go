package main

import (
	"github.com/wtiger001/teststudio/adapters/execution/kubernetes"
	"github.com/wtiger001/teststudio/adapters/preparation/registryimage"
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
	provider, err := settings.Open(service.Store, lookup, service.AuthorizeDiscoveryCleanup)
	if err != nil {
		registry.Close()
		return nil, err
	}
	exchange := &workerexchange.Service{Store: service.Store, Authorize: service.AuthorizeDiscovery, AuthorizeDiscoveryCleanup: service.AuthorizeDiscoveryCleanup}
	for _, runner := range []string{"go", "playwright"} {
		inspector := &hostedexecution.DiscoveryInspector{Provider: provider, Exchange: exchange, Name: "kubernetes", Runner: runner, ExchangeOrigin: settings.Worker.ExchangeOrigin}
		builder := &registryimage.Builder{Registry: registry, RuntimeImage: runtimes[runner], Directory: directory}
		service.RecoveryInspectors[runner] = inspector
		service.RecoveryBuilders[runner] = builder
		if runtimes[runner] != "" {
			service.Inspectors[runner], service.Builders[runner] = inspector, builder
		}
	}
	return func() { provider.Close(); registry.Close() }, nil
}
