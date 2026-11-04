package azureworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func fixture(t *testing.T, runner, operation string) (Config, hostedexecution.Job, *Plan, time.Time) {
	t.Helper()
	c := Config{Subscription: "11111111-2222-3333-4444-555555555555", ResourceGroup: "studio", Region: "eastus2", Registry: "studioregistry.azurecr.io", RepositoryPrefix: "teststudio", NamePrefix: "tsjob", NetworkProfile: strings.Repeat("a", 64), ExchangeOrigin: "https://workers.studio.internal.example.test"}
	c.WorkerEnvironment = c.resourceID("Microsoft.App", "managedEnvironments", "studio-workers")
	c.ControllerEnvironment = c.resourceID("Microsoft.App", "managedEnvironments", "studio-core")
	c.PullIdentity = c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "worker-pull")
	ref := execution.Reference{ID: "attempt-one", Installation: "studio", Project: "orders", Run: "run-one", Test: "ORDER-3", Attempt: 1, Worker: "controller-one"}
	if operation == "discover" {
		ref.Installation, ref.Test = "preparation-studio", "catalog"
		ref = (&hostedexecution.DiscoveryInspector{Name: Location, Runner: runner}).Reference(ref)
	} else {
		ref = (&hostedexecution.Executor{Name: Location, Runner: runner}).Reference(ref)
	}
	now := time.Date(2026, 11, 4, 20, 0, 0, 0, time.UTC)
	job := hostedexecution.Job{Operation: operation, Runner: runner, Reference: ref, Image: c.Registry + "/teststudio/source@sha256:" + strings.Repeat("b", 64), ExchangeOrigin: c.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: ref.ID, Token: strings.Repeat("private", 6) + "x", Deadline: now.Add(31*time.Second + time.Nanosecond)}}
	p, err := NewPlan(c, job, now)
	if err != nil {
		t.Fatal(err)
	}
	return c, job, p, now
}

func definition(p *Plan) a.Job {
	d := p.Definition()
	d.ID, d.Name, d.Type = to.Ptr(p.ResourceID()), to.Ptr(p.Name()), to.Ptr("Microsoft.App/jobs")
	d.SystemData = &a.SystemData{CreatedAt: to.Ptr(time.Now())}
	d.Properties.ProvisioningState = to.Ptr(a.JobProvisioningStateSucceeded)
	d.Properties.EventStreamEndpoint = to.Ptr("https://metadata.invalid")
	d.Properties.OutboundIPAddresses = []*string{to.Ptr("198.51.100.2")}
	d.Identity.TenantID = to.Ptr("provider-metadata")
	d.Identity.UserAssignedIdentities[p.config.PullIdentity] = &a.UserAssignedIdentity{ClientID: to.Ptr("provider-client"), PrincipalID: to.Ptr("provider-principal")}
	d.Properties.Template.Containers[0].Resources.EphemeralStorage = to.Ptr("8Gi")
	d.Properties.Template.Containers[0].Env = []*a.EnvironmentVar{}
	d.Properties.Template.InitContainers = []*a.InitContainer{}
	d.Properties.Template.Volumes = []*a.Volume{}
	return d
}

func executionReceipt(p *Plan) a.JobExecutionBase {
	name := p.Name() + "-abc1234"
	return a.JobExecutionBase{ID: to.Ptr(p.ResourceID() + "/executions/" + name), Name: to.Ptr(name)}
}

func executionResponse(t *testing.T, p *Plan, now time.Time) a.JobExecution {
	t.Helper()
	start, err := p.Start(now)
	if err != nil {
		t.Fatal(err)
	}
	receipt := executionReceipt(p)
	return a.JobExecution{ID: receipt.ID, Name: receipt.Name, Type: to.Ptr("Microsoft.App/jobs/executions"), Properties: &a.JobExecutionProperties{Template: start.Template, Status: to.Ptr(a.JobExecutionRunningStateRunning), StartTime: to.Ptr(now)}}
}

func TestAzureJobPlansFixDefinitionAndKeepTicketsInStartOnly(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				c, job, p, now := fixture(t, runner, operation)
				if err := p.VerifyDefinition(definition(p)); err != nil {
					t.Fatal(err)
				}
				draft := p.Definition()
				settings := draft.Properties.Configuration
				if len(p.Name()) >= 32 || strings.Contains(p.Name(), "--") || len(draft.Identity.UserAssignedIdentities) != 1 || value(draft.Identity.Type) != a.ManagedServiceIdentityTypeUserAssigned || len(settings.IdentitySettings) != 1 || value(settings.IdentitySettings[0].Lifecycle) != a.IdentitySettingsLifeCycleNone || value(settings.Registries[0].Identity) != c.PullIdentity || len(settings.Secrets) != 0 {
					t.Fatal("pull identity entered the worker runtime or the job name is invalid")
				}
				if value(settings.TriggerType) != a.TriggerTypeManual || value(settings.ReplicaTimeout) != 32 || value(settings.ReplicaRetryLimit) != 0 || value(settings.ManualTriggerConfig.Parallelism) != 1 || value(settings.ManualTriggerConfig.ReplicaCompletionCount) != 1 || settings.EventTriggerConfig != nil || settings.ScheduleTriggerConfig != nil || value(draft.Properties.WorkloadProfileName) != "Consumption" {
					t.Fatal("manual execution budget or logical count changed")
				}
				container := draft.Properties.Template.Containers[0]
				if len(container.Command) != 1 || value(container.Command[0]) != "/runner/teststudio-job" || value(container.Resources.CPU) != 2 || value(container.Resources.Memory) != "4Gi" || len(container.Env) != 0 || len(draft.Properties.Template.InitContainers) != 0 || len(draft.Properties.Template.Volumes) != 0 || len(container.VolumeMounts) != 0 || (len(container.Args) == 3) != (operation == "discover") {
					t.Fatal("job accepted a command, credentials, storage mount, or extra container")
				}
				data, _ := json.Marshal(draft)
				proofData, _ := json.Marshal(p.Proof())
				if strings.Contains(string(data)+string(proofData)+fmt.Sprintf("%v %#v", p, p), job.Ticket.Token) {
					t.Fatal("ticket escaped into saved configuration or diagnostics")
				}
				start, err := p.Start(now)
				if err != nil || value(start.Template.Containers[0].Env[1].Value) != job.Ticket.Token {
					t.Fatal("start did not retain its bounded ticket", err)
				}
				draft.Properties.Template.Containers[0].Command[0] = to.Ptr("untrusted")
				start.Template.Containers[0].Image = to.Ptr("untrusted")
				if p.VerifyDefinition(definition(p)) != nil {
					t.Fatal("returned SDK objects mutated the retained plan")
				}
				if _, err := p.Start(job.Ticket.Deadline); !errors.Is(err, hostedexecution.ErrRejected) {
					t.Fatal("expired plan launched", err)
				}
			})
		}
	}
}

func TestAzureJobDefinitionRejectsWritableDrift(t *testing.T) {
	cases := map[string]func(*a.Job){
		"foreign-resource": func(d *a.Job) {
			d.ID = to.Ptr("/subscriptions/foreign/resourceGroups/foreign/providers/Microsoft.App/jobs/foreign")
		},
		"changed-tag":      func(d *a.Job) { d.Tags["teststudio-project"] = to.Ptr("another-project") },
		"core-environment": func(d *a.Job) { d.Properties.EnvironmentID = to.Ptr("core-environment") },
		"runtime-identity": func(d *a.Job) { d.Identity.Type = to.Ptr(a.ManagedServiceIdentityTypeSystemAssignedUserAssigned) },
		"extra-identity":   func(d *a.Job) { d.Identity.UserAssignedIdentities["unexpected"] = &a.UserAssignedIdentity{} },
		"identity-lifecycle": func(d *a.Job) {
			d.Properties.Configuration.IdentitySettings[0].Lifecycle = to.Ptr(a.IdentitySettingsLifeCycleMain)
		},
		"secret": func(d *a.Job) {
			d.Properties.Configuration.Secrets = []*a.Secret{{Name: to.Ptr("secret"), Value: to.Ptr("private-canary")}}
		},
		"registry-password": func(d *a.Job) { d.Properties.Configuration.Registries[0].PasswordSecretRef = to.Ptr("registry-secret") },
		"parallelism":       func(d *a.Job) { d.Properties.Configuration.ManualTriggerConfig.Parallelism = to.Ptr[int32](2) },
		"retries":           func(d *a.Job) { d.Properties.Configuration.ReplicaRetryLimit = to.Ptr[int32](1) },
		"timeout":           func(d *a.Job) { d.Properties.Configuration.ReplicaTimeout = to.Ptr[int32](7200) },
		"event": func(d *a.Job) {
			d.Properties.Configuration.EventTriggerConfig = &a.JobConfigurationEventTriggerConfig{Parallelism: to.Ptr[int32](1)}
		},
		"command": func(d *a.Job) { d.Properties.Template.Containers[0].Command = []*string{to.Ptr("sh")} },
		"image": func(d *a.Job) {
			d.Properties.Template.Containers[0].Image = to.Ptr("studioregistry.azurecr.io/teststudio/source:latest")
		},
		"environment": func(d *a.Job) {
			d.Properties.Template.Containers[0].Env = []*a.EnvironmentVar{{Name: to.Ptr("injected"), Value: to.Ptr("private-canary")}}
		},
		"cpu":    func(d *a.Job) { d.Properties.Template.Containers[0].Resources.CPU = to.Ptr(1.0) },
		"memory": func(d *a.Job) { d.Properties.Template.Containers[0].Resources.Memory = to.Ptr("2Gi") },
		"mount": func(d *a.Job) {
			d.Properties.Template.Containers[0].VolumeMounts = []*a.VolumeMount{{MountPath: to.Ptr("/private"), VolumeName: to.Ptr("journals")}}
		},
		"sidecar": func(d *a.Job) {
			d.Properties.Template.Containers = append(d.Properties.Template.Containers, &a.Container{Name: to.Ptr("sidecar")})
		},
		"init": func(d *a.Job) { d.Properties.Template.InitContainers = []*a.InitContainer{{Name: to.Ptr("init")}} },
		"volume": func(d *a.Job) {
			d.Properties.Template.Volumes = []*a.Volume{{Name: to.Ptr("shared"), StorageType: to.Ptr(a.StorageTypeAzureFile)}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, p, _ := fixture(t, "go", "")
			d := definition(p)
			mutate(&d)
			if err := p.VerifyDefinition(d); !errors.Is(err, hostedexecution.ErrOwnership) || strings.Contains(err.Error(), "canary") {
				t.Fatal("changed job accepted or leaked private diagnostics", err)
			}
		})
	}
	_, _, p, _ := fixture(t, "go", "")
	d := definition(p)
	d.Properties.ProvisioningState = to.Ptr(a.JobProvisioningStateInProgress)
	if err := p.VerifyDefinition(d); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("unfinished resource admitted", err)
	}
}

func TestAzureJobRecoveryProofCannotStartOrChangeScope(t *testing.T) {
	c, job, p, now := fixture(t, "playwright", "discover")
	var proof Proof
	data, _ := json.Marshal(p.Proof())
	if err := json.Unmarshal(data, &proof); err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(c, proof)
	if err != nil || restored.Fingerprint() != p.Fingerprint() || restored.VerifyDefinition(definition(p)) != nil {
		t.Fatal("proof did not retain its frozen contract", err)
	}
	if _, err := restored.Start(now); !errors.Is(err, hostedexecution.ErrRejected) {
		t.Fatal("recovery recreated launch authority", err)
	}
	if _, err := restored.VerifyExecution(executionReceipt(p), executionResponse(t, p, now)); err != nil {
		t.Fatal("recovery cannot verify original execution", err)
	}
	proof.TimeoutSeconds++
	if _, err := Restore(c, proof); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("changed native timeout restored", err)
	}
	c.NetworkProfile = strings.Repeat("d", 64)
	if _, err := Restore(c, p.Proof()); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("another isolation pool restored", err)
	}
	job.Ticket.Deadline = now.Add(3 * time.Hour)
	if _, err := NewPlan(c, job, now); !errors.Is(err, hostedexecution.ErrConfiguration) {
		t.Fatal("unbounded deadline accepted", err)
	}
}
