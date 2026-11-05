package main

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type registrationProvider struct{ hostedexecution.Provider }

func TestHostedRegistrationKeepsExecutionAndDiscoveryProvidersSeparate(t *testing.T) {
	execution, discovery := &registrationProvider{}, &registrationProvider{}
	for _, mode := range []string{"separate", "execution-only", "discovery-only", "recovery-only"} {
		t.Run(mode, func(t *testing.T) {
			var execute, inspect hostedexecution.Provider = execution, discovery
			if mode == "execution-only" {
				inspect = nil
			}
			if mode == "discovery-only" {
				execute = nil
			}
			run, source, exchange := &runs.Service{}, &sources.Service{}, &workerexchange.Service{}
			configureHostedWorkers(run, source, exchange, execute, inspect, "owned", "https://exchange.studio.example", mode != "recovery-only")
			if execute == nil {
				if run.Go != nil || run.Playwright != nil || len(run.RecoveryDrivers) != 0 {
					t.Fatal("discovery host gained coded execution")
				}
			} else {
				driver, ok := run.Go.(*hostedexecution.Executor)
				if !ok || driver.Provider != execution || len(run.RecoveryDrivers) != 2 {
					t.Fatal("execution registration uses discovery authority")
				}
			}
			if inspect == nil {
				if len(source.Inspectors)+len(source.RecoveryInspectors) != 0 {
					t.Fatal("execution host gained discovery authority")
				}
			} else {
				for _, runner := range []string{"go", "playwright"} {
					inspector, ok := source.RecoveryInspectors[runner].(*hostedexecution.DiscoveryInspector)
					if !ok || inspector.Provider != discovery {
						t.Fatal("discovery recovery uses execution authority")
					}
				}
				if mode == "recovery-only" && len(source.Inspectors) != 0 || mode != "recovery-only" && len(source.Inspectors) != 2 {
					t.Fatal("new discovery admission differs from configured mode")
				}
			}
		})
	}
}
