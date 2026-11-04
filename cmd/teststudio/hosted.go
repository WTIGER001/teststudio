package main

import (
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func configureHostedWorkers(run *runs.Service, source *sources.Service, exchange *workerexchange.Service, provider hostedexecution.Provider, name, origin string, preparationEnabled bool) {
	makeDriver := func(runner string) *hostedexecution.Executor {
		return &hostedexecution.Executor{Provider: provider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin, ApproveArtifact: source.ApproveArtifact}
	}
	goDriver, playwright := makeDriver("go"), makeDriver("playwright")
	run.Go, run.Playwright = goDriver, playwright
	run.GoAPI, run.Browser = nil, nil
	run.RecoveryDriver = nil
	run.RecoveryDrivers = []execution.ManagedExecutor{goDriver, playwright}
	source.RecoveryInspectors = map[string]preparation.ArtifactInspector{}
	source.Inspectors = map[string]preparation.ArtifactInspector{}
	for _, runner := range []string{"go", "playwright"} {
		inspector := &hostedexecution.DiscoveryInspector{Provider: provider, Exchange: exchange, Name: name, Runner: runner, ExchangeOrigin: origin}
		source.RecoveryInspectors[runner] = inspector
		if preparationEnabled {
			source.Inspectors[runner] = inspector
		}
	}
	configureWorkerAuthorization(run, source, exchange)
}
