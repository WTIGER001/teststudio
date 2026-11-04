package main

import (
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func configureKubernetesWorkers(run *runs.Service, source *sources.Service, exchange *workerexchange.Service, provider hostedexecution.Provider, origin string) {
	makeDriver := func(runner string) *hostedexecution.Executor {
		return &hostedexecution.Executor{Provider: provider, Exchange: exchange, Name: "kubernetes", Runner: runner, ExchangeOrigin: origin, ApproveArtifact: source.ApproveArtifact}
	}
	goDriver, playwright := makeDriver("go"), makeDriver("playwright")
	run.Go, run.Playwright = goDriver, playwright
	run.GoAPI, run.Browser = nil, nil
	run.RecoveryDriver = nil
	run.RecoveryDrivers = []execution.ManagedExecutor{goDriver, playwright}
	source.RecoveryInspectors = map[string]preparation.ArtifactInspector{}
}
