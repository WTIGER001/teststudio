package main

import (
	"context"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type registrationProvider struct{ hostedexecution.Provider }

type gatewayRegistrationProvider struct{ registrationProvider }

func (p *gatewayRegistrationProvider) SupportsGateway() bool { return true }
func (p *gatewayRegistrationProvider) GatewayProfile(context.Context) (string, error) {
	return "gateway-v1-fixture", nil
}

func TestHostedRegistrationEnablesEnvironmentProfilesOnlyWithGatewaySupport(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var provider hostedexecution.Provider = &registrationProvider{}
		if enabled {
			provider = &gatewayRegistrationProvider{}
		}
		run, source, exchange := &runs.Service{}, &sources.Service{}, &workerexchange.Service{}
		configureHostedWorkers(run, source, exchange, provider, nil, "kubernetes", "https://exchange.test", false)
		if !enabled {
			if run.GoAPI != nil || run.Browser != nil || len(run.RecoveryDrivers) != 2 {
				t.Fatal("offline provider enabled gateway execution")
			}
			continue
		}
		goAPI, ok := run.GoAPI.(*hostedexecution.Executor)
		if !ok || !goAPI.Environment || goAPI.Profile() != "go-kubernetes-environment-v1" || len(run.RecoveryDrivers) != 4 {
			t.Fatal("gateway provider did not register separate execution and recovery profiles")
		}
		browser, ok := run.Browser.(*hostedexecution.Executor)
		if !ok || !browser.Environment || browser.Provider != provider {
			t.Fatal("browser registration lost its trusted gateway provider")
		}
	}
}

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

func TestHostedRegistrationKeepsECSOfflineAndEnvironmentPoolsSeparate(t *testing.T) {
	offline, environment := &registrationProvider{}, &gatewayRegistrationProvider{}
	run, source, exchange := &runs.Service{}, &sources.Service{}, &workerexchange.Service{}
	configureHostedWorkers(run, source, exchange, offline, offline, "ecs", "https://exchange.test", true)
	configureHostedEnvironmentWorkers(run, source, exchange, environment, "ecs", "https://exchange.test")
	if run.Go.(*hostedexecution.Executor).Provider != offline || run.Playwright.(*hostedexecution.Executor).Provider != offline || len(run.RecoveryDrivers) != 4 {
		t.Fatal("environment registration replaced offline execution or recovery")
	}
	for _, driver := range []any{run.GoAPI, run.Browser} {
		registered := driver.(*hostedexecution.Executor)
		if registered.Provider != environment || !registered.Environment {
			t.Fatal("application execution uses the offline pool")
		}
	}
	for _, runner := range []string{"go", "playwright"} {
		inspector := source.Inspectors[runner].(*hostedexecution.DiscoveryInspector)
		if inspector.Provider != offline || !inspector.RequireGatewayAddress {
			t.Fatal("ECS source discovery lost its isolated pool or address-capability requirement")
		}
	}
}
