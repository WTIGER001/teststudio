package main

import (
	"errors"
	"flag"

	"github.com/wtiger001/teststudio/adapters/execution/kubernetes"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func validateDiscoveryOnlyFlags(flags *flag.FlagSet, enabled, assembly bool, location string, ecs, kubernetes bool) error {
	if !enabled {
		return nil
	}
	invalid := assembly || ecs == kubernetes || ecs && location != "ecs" || kubernetes && location != "kubernetes" || flags.NArg() != 0
	flags.Visit(func(value *flag.Flag) {
		switch value.Name {
		case "discovery-only", "database", "database-env", "require-postgres", "execution-location", "ecs-workers-config", "ecs-workers-env", "kubernetes-workers-config", "kubernetes-workers-env":
		default:
			invalid = true
		}
	})
	if invalid {
		return errors.New("discovery-only preparation requires one matching hosted execution profile, database settings, and no assembly, source runtimes, build directory, repository sync, or secret binding flags")
	}
	return nil
}

func attachHostedDiscovery(service *sources.Service, provider hostedexecution.Provider, name, origin string) error {
	if service == nil || provider == nil || service.AssemblyOnly || len(service.Builders)+len(service.RecoveryBuilders) != 0 {
		return hostedexecution.ErrConfiguration
	}
	exchange := &workerexchange.Service{Store: service.Store, Authorize: service.AuthorizeDiscovery, AuthorizeDiscoveryCleanup: service.AuthorizeDiscoveryCleanup}
	inspectors, recovery := map[string]preparation.ArtifactInspector{}, map[string]preparation.ArtifactInspector{}
	for _, runner := range []string{"go", "playwright"} {
		inspector := &hostedexecution.DiscoveryInspector{Provider: provider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin}
		inspectors[runner], recovery[runner] = inspector, inspector
	}
	service.Inspectors, service.RecoveryInspectors = inspectors, recovery
	return nil
}

func configureKubernetesDiscovery(service *sources.Service, settings kubernetes.Settings, lookup func(string) (string, bool)) (func(), error) {
	provider, err := settings.Open(service.Store, lookup, service.AuthorizeDiscoveryCleanup)
	if err != nil {
		return nil, err
	}
	if err := attachHostedDiscovery(service, provider, "kubernetes", settings.Worker.ExchangeOrigin); err != nil {
		provider.Close()
		return nil, err
	}
	return provider.Close, nil
}
