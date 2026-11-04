package azureworker

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureProviderWaitsForProvisioningAndFencesStartAcrossRestart(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+operation, func(t *testing.T) {
				f := providerTest(t, runner, operation)
				f.jobs.async = true
				if err := f.provider.Ensure(t.Context(), f.job); err != nil {
					t.Fatal(err)
				}
				f.bindDefinition()
				s, err := f.provider.Observe(t.Context(), f.job.Reference)
				if err != nil || !s.AdmissionPending || f.jobs.starts != 0 || f.control.ensures != 0 {
					t.Fatal("provisioning released an execution", s, err)
				}
				f.reopen()
				f.now = f.now.Add(time.Second)
				f.ready()
				if err := f.provider.Ensure(t.Context(), f.job); err != nil {
					t.Fatal("a later clock changed the frozen replica budget", err)
				}
				for range 2 {
					s, err = f.provider.Observe(t.Context(), f.job.Reference)
					if err != nil {
						t.Fatal(err)
					}
				}
				if s.AdmissionPending || s.Phase != "running" || f.jobs.creates != 1 || f.jobs.starts != 1 || f.control.ensures != 1 {
					t.Fatal("restart repeated a mutation or lost admission", s)
				}
				if _, err := f.exchange.Input(t.Context(), f.job.Reference.ID, f.job.Ticket.Token, "worker"); err == nil {
					t.Fatal("provider admission alone released executable input")
				}
				if err := f.exchange.Activate(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if _, err := f.exchange.Input(t.Context(), f.job.Reference.ID, f.job.Ticket.Token, "worker"); err != nil {
					t.Fatal("controller activation did not release the original packet", err)
				}
				if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil || f.jobs.deletes != 0 {
					t.Fatal("live work was deleted without its checkpoint")
				}
				f.revoke(false)
				if err := f.provider.Stop(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil || f.jobs.deletes != 0 {
					t.Fatal("revocation replaced the result checkpoint")
				}
				f.revoke(true)
				if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if f.jobs.deletes != 1 || f.jobs.stops != 1 || f.control.removes != 1 {
					t.Fatal("cleanup did not preserve exact resources")
				}
				if err := f.provider.Ensure(t.Context(), f.job); !errors.Is(err, hostedexecution.ErrRejected) {
					t.Fatal("removed attempt launched again", err)
				}
			})
		}
	}
}

func TestAzureProviderRecoversLostMutationResponsesWithoutRepeatingStart(t *testing.T) {
	for _, mode := range []string{"lost-create", "lost-start", "hidden-start"} {
		t.Run(mode, func(t *testing.T) {
			f := providerTest(t, "go", "")
			f.jobs.mode = mode
			f.jobs.hidden = mode == "hidden-start"
			if err := f.provider.Ensure(t.Context(), f.job); err != nil {
				t.Fatal(err)
			}
			f.bindDefinition()
			f.ready()
			if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			f.reopen()
			if err := f.provider.Ensure(t.Context(), f.job); err != nil {
				t.Fatal(err)
			}
			if mode == "hidden-start" {
				s, err := f.provider.Observe(t.Context(), f.job.Reference)
				if err != nil || !s.AdmissionPending || f.jobs.starts != 1 {
					t.Fatal("empty history recreated or admitted uncertain work", s, err)
				}
				f.revoke(true)
				if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil || f.jobs.deletes != 0 {
					t.Fatal("missing live history was treated as terminal")
				}
				f.jobs.hidden = false
			}
			s, err := f.provider.Observe(t.Context(), f.job.Reference)
			if err != nil || s.Phase != "running" {
				t.Fatal(s, err)
			}
			f.revoke(true)
			if err := f.provider.Stop(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			if f.jobs.creates != 1 || f.jobs.starts != 1 {
				t.Fatal("uncertain mutations repeated")
			}
		})
	}
}

func TestAzureProviderConcurrentControllersReserveOneCreateAndStart(t *testing.T) {
	f := providerTest(t, "playwright", "")
	var group sync.WaitGroup
	for range 6 {
		group.Go(func() {
			if err := f.provider.Ensure(t.Context(), f.job); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	f.bindDefinition()
	f.ready()
	for range 6 {
		group.Go(func() {
			if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.jobs.creates != 1 || f.jobs.starts != 1 || f.control.ensures != 1 {
		t.Fatal("controllers repeated a reserved mutation", f.jobs.creates, f.jobs.starts, f.control.ensures)
	}
	if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestAzureProviderRetainsNativeCompletionAndDeletionRecovery(t *testing.T) {
	f := providerTest(t, "go", "")
	f.start()
	f.jobs.items[0].Properties.Status = to.Ptr(a.JobExecutionRunningStateSucceeded)
	s, err := f.provider.Observe(t.Context(), f.job.Reference)
	if err != nil || !s.NativeCompletion || s.ExitCode != nil || s.Phase != "succeeded" {
		t.Fatal("provider completion invented a process exit code", s, err)
	}
	f.reopen()
	f.jobs.items = nil
	s, err = f.provider.Observe(t.Context(), f.job.Reference)
	if err != nil || s.Phase != "succeeded" || s.ExitCode != nil {
		t.Fatal("terminal history expired with provider inventory", s, err)
	}
	f.revoke(true)
	f.jobs.mode = "delete-pending"
	if err := f.provider.Remove(t.Context(), f.job.Reference); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("pending deletion treated as complete", err)
	}
	reads := f.jobs.reads
	if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil || f.jobs.reads != reads {
		t.Fatal("pending deletion did not back off")
	}
	f.jobs.definition = nil
	f.now = f.now.Add(31 * time.Second)
	if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil {
		t.Fatal(err)
	}
	s, err = f.provider.Observe(t.Context(), f.job.Reference)
	if err != nil || s.Phase != "succeeded" {
		t.Fatal("cleanup discarded terminal completion", s, err)
	}
	rows, err := f.provider.Inventory(t.Context(), f.job.Reference.Installation, f.job.Reference.Profile)
	if err != nil || len(rows) != 0 {
		t.Fatal("removed worker remained in recovery inventory", rows, err)
	}
}

func TestAzureProviderBlocksDriftMissingDeadlinesAndDuplicateExecutions(t *testing.T) {
	for _, mode := range []string{"deadline", "network", "definition", "duplicate", "execution"} {
		t.Run(mode, func(t *testing.T) {
			f := providerTest(t, "playwright", "")
			f.start()
			switch mode {
			case "deadline":
				f.control.deadline = nil
			case "network":
				f.control.pending = true
			case "definition":
				f.jobs.definition.Properties.Template.Containers[0].Command = []*string{to.Ptr("sh")}
			case "duplicate":
				f.jobs.items = append(f.jobs.items, clone(f.jobs.items[0]))
			case "execution":
				f.jobs.items[0].Properties.Template.Containers[0].Env[1].Value = to.Ptr(strings.Repeat("x", 43))
			}
			s, err := f.provider.Observe(t.Context(), f.job.Reference)
			if mode == "deadline" || mode == "network" {
				if err != nil || !s.AdmissionPending {
					t.Fatal("lost admission left executable authority", s, err)
				}
			} else if !errors.Is(err, hostedexecution.ErrOwnership) {
				t.Fatal("changed provider ownership admitted", err)
			}
		})
	}
}

func TestAzureProviderCancelsOwnedUnknownStateAndUnstartedProvisioning(t *testing.T) {
	for _, mode := range []string{"unknown", "provisioning"} {
		t.Run(mode, func(t *testing.T) {
			f := providerTest(t, "go", "")
			if mode == "unknown" {
				f.start()
				f.jobs.items[0].Properties.Status = to.Ptr(a.JobExecutionRunningStateUnknown)
			} else {
				f.jobs.async = true
				if err := f.provider.Ensure(t.Context(), f.job); err != nil {
					t.Fatal(err)
				}
				f.bindDefinition()
			}
			f.revoke(true)
			if err := f.provider.Stop(t.Context(), f.job.Reference); err != nil {
				t.Fatal("owned cancellation required a successful provider observation", err)
			}
			status, err := f.provider.Observe(t.Context(), f.job.Reference)
			if err != nil || status.Phase != "stopped" {
				t.Fatal("cancellation did not reach a saved terminal observation", status, err)
			}
			if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			if mode == "provisioning" && f.jobs.starts != 0 {
				t.Fatal("cancellation started pending work")
			}
		})
	}
}
