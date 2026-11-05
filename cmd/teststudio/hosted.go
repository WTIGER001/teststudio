package main

import (
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func configureHostedWorkers(run *runs.Service, source *sources.Service, exchange *workerexchange.Service, executionProvider, discoveryProvider hostedexecution.Provider, name, origin string, preparationEnabled bool) {
	makeDriver := func(runner string) *hostedexecution.Executor {
		return &hostedexecution.Executor{Provider: executionProvider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin, ApproveArtifact: source.ApproveArtifact}
	}
	run.Go, run.Playwright = nil, nil
	run.RecoveryDrivers = nil
	if executionProvider != nil {
		goDriver, playwright := makeDriver("go"), makeDriver("playwright")
		run.Go, run.Playwright = goDriver, playwright
		run.RecoveryDrivers = []execution.ManagedExecutor{goDriver, playwright}
	}
	run.GoAPI, run.Browser = nil, nil
	configureHostedEnvironmentWorkers(run, source, exchange, executionProvider, name, origin)
	run.RecoveryDriver = nil
	source.RecoveryInspectors = map[string]preparation.ArtifactInspector{}
	source.Inspectors = map[string]preparation.ArtifactInspector{}
	if discoveryProvider != nil {
		for _, runner := range []string{"go", "playwright"} {
			inspector := &hostedexecution.DiscoveryInspector{RequireGatewayAddress: name == "ecs", Provider: discoveryProvider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin}
			source.RecoveryInspectors[runner] = inspector
			if preparationEnabled {
				source.Inspectors[runner] = inspector
			}
		}
	}
	configureWorkerAuthorization(run, source, exchange)
}

func configureHostedEnvironmentWorkers(run *runs.Service, source *sources.Service, exchange *workerexchange.Service, provider hostedexecution.Provider, name, origin string) {
	gateway, ok := provider.(hostedexecution.GatewayProvider)
	if !ok || !gateway.SupportsGateway() {
		return
	}
	driver := func(runner string) *hostedexecution.Executor {
		return &hostedexecution.Executor{Environment: true, Provider: provider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin, ApproveArtifact: source.ApproveArtifact}
	}
	goAPI, browser := driver("go"), driver("playwright")
	run.GoAPI, run.Browser = goAPI, browser
	run.RecoveryDrivers = append(run.RecoveryDrivers, goAPI, browser)
}
