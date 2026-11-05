package azuresandbox

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestSandboxRuntimeModesCannotControlTheOtherWorkload(t *testing.T) {
	for _, mode := range []string{"execution", "discovery"} {
		t.Run(mode, func(t *testing.T) {
			operation := ""
			if mode == "execution" {
				operation = "discover"
			}
			f := newControllerFixture(t, "go", operation)
			source := fixtureImagePlan(t, f.plan)
			a := &imageControl{t: t, items: map[string]DiskImage{}}
			m, err := NewImageManager(ImageManagerOptions{Config: source.config, PullIdentity: source.proof.PullIdentity, Images: a, Exchange: f.exchange, Admission: imageAdmission{}, AllowImport: true})
			if err != nil {
				t.Fatal(err)
			}
			f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
				t.Error("another workload executed code")
				return CommandResult{}, nil
			})
			p := fixtureProvider(t, f, m, source.proof.Image)
			job := hostedexecution.Job{Reference: f.plan.proof.Reference, Image: f.plan.proof.Image, Runner: "go", Operation: operation, ExchangeOrigin: source.config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: f.plan.proof.Reference.ID, Token: f.plan.token, Deadline: f.plan.proof.Deadline}}
			image, err := p.InspectImage(t.Context(), job.Image)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.reserveIntent(t.Context(), job, image); err != nil {
				t.Fatal(err)
			}
			scoped, err := NewProvider(ProviderOptions{Mode: mode, Controller: f.controller, Images: m, Inspector: p.inspector})
			if err != nil {
				t.Fatal(err)
			}
			if err := scoped.Ensure(t.Context(), job); err != hostedexecution.ErrConfiguration {
				t.Fatal("wrong workload was admitted", err)
			}
			if _, err := scoped.Observe(t.Context(), job.Reference); err != hostedexecution.ErrOwnership {
				t.Fatal("wrong workload was observed", err)
			}
			if err := scoped.Stop(t.Context(), job.Reference); err != hostedexecution.ErrOwnership {
				t.Fatal("wrong workload was stopped", err)
			}
			if err := scoped.Remove(t.Context(), job.Reference); err != hostedexecution.ErrOwnership {
				t.Fatal("wrong workload was removed", err)
			}
			if _, err := scoped.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile); err != hostedexecution.ErrConfiguration {
				t.Fatal("wrong workload was enumerated", err)
			}
			state, err := f.exchange.State(t.Context(), job.Reference)
			if err != nil || state.Revoked || a.imports != 0 || f.api.createCalls != 0 {
				t.Fatal("wrong mode altered rightful work", err)
			}
			settings := runtimeSettings()
			settings.Mode = mode
			credential := &runtimeCredential{token: "unused-construction-canary"}
			runtime, err := OpenRuntime(RuntimeOptions{Settings: settings, Exchange: f.exchange, Admission: f.controls, ImageAdmission: imageAdmission{}, Credentials: func(string) (azcore.TokenCredential, error) { return credential, nil }})
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Close()
			if runtime.Provider.mode != mode || runtime.Provider.images.writer != (mode == "discovery") || len(credential.scopes) != 0 {
				t.Fatal("native factory widened its workload mode")
			}
		})
	}
}
