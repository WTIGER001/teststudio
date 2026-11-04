package azureworker

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureJobExecutionObservationsRetainProviderFacts(t *testing.T) {
	_, _, p, now := fixture(t, "playwright", "")
	for state, phase := range map[a.JobExecutionRunningState]string{a.JobExecutionRunningStateProcessing: "pending", a.JobExecutionRunningStateRunning: "running", a.JobExecutionRunningStateSucceeded: "succeeded", a.JobExecutionRunningStateFailed: "failed", a.JobExecutionRunningStateStopped: "stopped"} {
		t.Run(string(state), func(t *testing.T) {
			actual := executionResponse(t, p, now)
			actual.ID, actual.Type = nil, nil
			actual.Properties.Status = to.Ptr(state)
			if phase == "succeeded" || phase == "failed" || phase == "stopped" {
				actual.Properties.EndTime = to.Ptr(now.Add(time.Second))
			}
			observed, err := p.VerifyExecution(executionReceipt(p), actual)
			if err != nil || observed.Phase != phase || observed.ExecutionID != value(executionReceipt(p).ID) {
				t.Fatal("provider response without optional resource metadata rejected", observed, err)
			}
			*observed.StartedAt = now.Add(time.Hour)
			if !actual.Properties.StartTime.Equal(now) {
				t.Fatal("observation changed the SDK response")
			}
		})
	}
}

func TestAzureJobExecutionRejectsChangedReceiptTemplateAndUncertainState(t *testing.T) {
	cases := map[string]func(*a.JobExecution){
		"different-execution": func(v *a.JobExecution) { v.Name = to.Ptr("another-execution") },
		"foreign-id":          func(v *a.JobExecution) { v.ID = to.Ptr("/subscriptions/foreign/executions/foreign") },
		"foreign-type":        func(v *a.JobExecution) { v.Type = to.Ptr("Microsoft.App/containerApps") },
		"image":               func(v *a.JobExecution) { v.Properties.Template.Containers[0].Image = to.Ptr("unapproved:latest") },
		"command":             func(v *a.JobExecution) { v.Properties.Template.Containers[0].Command = []*string{to.Ptr("sh")} },
		"ticket": func(v *a.JobExecution) {
			v.Properties.Template.Containers[0].Env[1].Value = to.Ptr(strings.Repeat("x", 43))
		},
		"execution-binding": func(v *a.JobExecution) { v.Properties.Template.Containers[0].Env[0].Value = to.Ptr("another-attempt") },
		"extra-env": func(v *a.JobExecution) {
			v.Properties.Template.Containers[0].Env = append(v.Properties.Template.Containers[0].Env, &a.EnvironmentVar{Name: to.Ptr("extra"), Value: to.Ptr("private-canary")})
		},
		"secret-reference": func(v *a.JobExecution) { v.Properties.Template.Containers[0].Env[0].SecretRef = to.Ptr("secret") },
		"duplicate-env": func(v *a.JobExecution) {
			v.Properties.Template.Containers[0].Env[1].Name = to.Ptr("TESTSTUDIO_JOB_EXECUTION")
		},
		"sidecar": func(v *a.JobExecution) {
			v.Properties.Template.Containers = append(v.Properties.Template.Containers, &a.JobExecutionContainer{Name: to.Ptr("sidecar")})
		},
		"init": func(v *a.JobExecution) {
			v.Properties.Template.InitContainers = []*a.JobExecutionContainer{{Name: to.Ptr("init")}}
		},
		"unknown":          func(v *a.JobExecution) { v.Properties.Status = to.Ptr(a.JobExecutionRunningStateUnknown) },
		"degraded":         func(v *a.JobExecution) { v.Properties.Status = to.Ptr(a.JobExecutionRunningStateDegraded) },
		"new-state":        func(v *a.JobExecution) { v.Properties.Status = to.Ptr(a.JobExecutionRunningState("new-state")) },
		"missing-template": func(v *a.JobExecution) { v.Properties.Template = nil },
		"running-with-end": func(v *a.JobExecution) { v.Properties.EndTime = to.Ptr(time.Now()) },
		"backwards-times": func(v *a.JobExecution) {
			v.Properties.Status = to.Ptr(a.JobExecutionRunningStateSucceeded)
			v.Properties.EndTime = to.Ptr(v.Properties.StartTime.Add(-time.Second))
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, p, now := fixture(t, "go", "")
			actual := executionResponse(t, p, now)
			mutate(&actual)
			_, err := p.VerifyExecution(executionReceipt(p), actual)
			if err == nil || !errors.Is(err, hostedexecution.ErrOwnership) && !errors.Is(err, hostedexecution.ErrUnavailable) || strings.Contains(err.Error(), "canary") {
				t.Fatal("execution drift or unknown provider state accepted", err)
			}
		})
	}
	_, _, p, now := fixture(t, "go", "")
	for _, state := range []a.JobExecutionRunningState{a.JobExecutionRunningStateRunning, a.JobExecutionRunningStateFailed, a.JobExecutionRunningStateStopped, a.JobExecutionRunningStateSucceeded} {
		actual := executionResponse(t, p, now)
		actual.Properties.Status, actual.Properties.StartTime, actual.Properties.EndTime = to.Ptr(state), nil, nil
		observed, err := p.VerifyExecution(executionReceipt(p), actual)
		if err != nil || observed.StartedAt != nil || observed.FinishedAt != nil {
			t.Fatal("missing optional provider times were invented or prevented observation", err)
		}
	}
	receipt := executionReceipt(p)
	receipt.ID = to.Ptr(strings.Replace(value(receipt.ID), "/jobs/", "/containerApps/", 1))
	if err := p.VerifyReceipt(receipt); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("foreign receipt accepted", err)
	}
}

func TestAzureJobConfigurationScopesImagesOriginsAndTimingPools(t *testing.T) {
	c, job, _, now := fixture(t, "go", "")
	profile, err := ExecutionProfile(c)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"same-environment": func(v *Config) { v.WorkerEnvironment = v.ControllerEnvironment },
		"foreign-subscription": func(v *Config) {
			v.WorkerEnvironment = strings.Replace(v.WorkerEnvironment, v.Subscription, "99999999-2222-3333-4444-555555555555", 1)
		},
		"resource-path":        func(v *Config) { v.PullIdentity += "/extra" },
		"missing-network":      func(v *Config) { v.NetworkProfile = "" },
		"long-prefix":          func(v *Config) { v.NamePrefix = "too-long-prefix" },
		"double-hyphen":        func(v *Config) { v.NamePrefix = "ts--aa" },
		"public-registry":      func(v *Config) { v.Registry = "docker.io" },
		"repository-traversal": func(v *Config) { v.RepositoryPrefix = "studio/../source" },
		"http":                 func(v *Config) { v.ExchangeOrigin = "http://workers.example.test" },
		"ip":                   func(v *Config) { v.ExchangeOrigin = "https://127.0.0.1" },
		"port":                 func(v *Config) { v.ExchangeOrigin += ":443" },
		"userinfo":             func(v *Config) { v.ExchangeOrigin = "https://user@workers.example.test" },
		"bad-label":            func(v *Config) { v.ExchangeOrigin = "https://workers.-invalid.test" },
		"path":                 func(v *Config) { v.ExchangeOrigin += "/path" },
		"query":                func(v *Config) { v.ExchangeOrigin += "?private" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := c
			mutate(&changed)
			if Validate(changed) == nil {
				t.Fatal("invalid execution pool accepted")
			}
		})
	}
	changed := c
	changed.NetworkProfile = strings.Repeat("c", 64)
	other, err := ExecutionProfile(changed)
	if err != nil || profile == other {
		t.Fatal("isolation change reused comparable timing history", err)
	}
	job.Image = "studioregistry.azurecr.io/teststudio-other/source@sha256:" + strings.Repeat("b", 64)
	if _, err := NewPlan(c, job, now); !errors.Is(err, hostedexecution.ErrConfiguration) {
		t.Fatal("repository prefix boundary ignored", err)
	}
}
