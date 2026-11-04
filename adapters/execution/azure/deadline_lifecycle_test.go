package azureworker

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureDeadlineSDKSurvivesLostCreateAndControllerRestart(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+operation, func(t *testing.T) {
				f := providerTest(t, runner, operation)
				wire := &deadlineWire{mode: "lost-create"}
				client := attachDeadlines(t, f, wire)
				if err := f.provider.Ensure(t.Context(), f.job); err != nil {
					t.Fatal(err)
				}
				f.bindDefinition()
				s, err := f.provider.Observe(t.Context(), f.job.Reference)
				if err != nil || !s.AdmissionPending || wire.creates != 1 || f.jobs.starts != 0 {
					t.Fatal("unknown timer creation released or repeated work", s, err)
				}
				client.Close()
				f.reopen()
				f.now = f.now.Add(time.Second)
				client = attachDeadlines(t, f, wire)
				for range 2 {
					s, err = f.provider.Observe(t.Context(), f.job.Reference)
					if err != nil {
						t.Fatal(err)
					}
				}
				if s.AdmissionPending || s.Phase != "running" || f.jobs.starts != 1 || wire.creates != 1 {
					t.Fatal("owned timer did not resume once", s)
				}
				if err := client.Ensure(t.Context(), wire.spec); err != nil || wire.creates != 1 {
					t.Fatal("existing timer was rearmed", err)
				}
				if err := client.Remove(t.Context(), wire.spec); err == nil || wire.deletes != 0 {
					t.Fatal("timer removed for live work")
				}
				f.revoke(true)
				if err := client.Remove(t.Context(), wire.spec); err == nil || wire.deletes != 0 {
					t.Fatal("timer removed before job absence")
				}
				if err := f.provider.Stop(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil {
					t.Fatal(err)
				}
				if wire.creates != 1 || wire.deletes != 1 || wire.resource != nil || f.jobs.deletes != 1 {
					t.Fatal("cleanup lost ordering or repeated a mutation")
				}
				requireDeadlineError(t, client.Ensure(t.Context(), wire.spec), hostedexecution.ErrRejected)
			})
		}
	}
}

func TestAzureDeadlineUnseenCreationCannotBecomeSafeAbsence(t *testing.T) {
	f := providerTest(t, "go", "")
	wire := &deadlineWire{mode: "lost-create"}
	client := attachDeadlines(t, f, wire)
	s := prepareDeadline(t, f, client)
	requireDeadlineError(t, client.Ensure(t.Context(), s), hostedexecution.ErrUnavailable)
	wire.hidden = true
	requireDeadlineError(t, client.Verify(t.Context(), s), hostedexecution.ErrNotFound)
	f.revoke(true)
	if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil || wire.deletes != 0 {
		t.Fatal("unknown deadline creation was declared absent")
	}
	d, _, _, err := f.provider.load(t.Context(), f.job.Reference)
	if err != nil || d.Removed || !d.DeadlineRemoving {
		t.Fatal("cleanup lost its durable fence", err)
	}
	wire.hidden = false
	f.now = f.now.Add(31 * time.Second)
	if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil || wire.deletes != 1 {
		t.Fatal("late timer was not reclaimed", err)
	}
}

func TestAzureDeadlineConcurrentControllersCreateOnlyOnce(t *testing.T) {
	f := providerTest(t, "playwright", "")
	wire := &deadlineWire{}
	client := attachDeadlines(t, f, wire)
	s := prepareDeadline(t, f, client)
	var group sync.WaitGroup
	for range 6 {
		group.Go(func() {
			err := client.Ensure(t.Context(), s)
			if err != nil && !errors.Is(err, hostedexecution.ErrUnavailable) && !errors.Is(err, hostedexecution.ErrOwnership) {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if wire.creates != 1 || client.Verify(t.Context(), s) != nil {
		t.Fatal("controllers repeated creation or failed to reconcile")
	}
}

func TestAzureDeadlineRejectsUnreadyDriftAndUnownedResources(t *testing.T) {
	for _, mode := range []string{"provisioning", "foreign", "drift", "create-rejected"} {
		t.Run(mode, func(t *testing.T) {
			f := providerTest(t, "go", "")
			wire := &deadlineWire{mode: mode}
			client := attachDeadlines(t, f, wire)
			s := prepareDeadline(t, f, client)
			if mode == "foreign" {
				d := deadlineDefinition(wire.config, s)
				wire.resource = &d
			}
			err := client.Ensure(t.Context(), s)
			if mode == "foreign" {
				requireDeadlineError(t, err, hostedexecution.ErrOwnership)
			} else if mode == "create-rejected" {
				requireDeadlineError(t, err, hostedexecution.ErrRejected)
			} else if err != nil {
				t.Fatal(err)
			}
			if mode == "drift" {
				wire.resource.Properties.State = to.Ptr(l.WorkflowStateDisabled)
			}
			if err := client.Verify(t.Context(), s); err == nil {
				t.Fatal("unready or foreign timer admitted work")
			}
			if f.jobs.starts != 0 {
				t.Fatal("negative fixture started a worker")
			}
			f.revoke(true)
			err = f.provider.Remove(t.Context(), f.job.Reference)
			if mode == "foreign" || mode == "drift" {
				if err == nil || wire.deletes != 0 {
					t.Fatal("unowned timer deleted")
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAzureDeadlineDeleteBackoffAndLostResponseRecover(t *testing.T) {
	for _, mode := range []string{"delete-pending", "lost-delete"} {
		t.Run(mode, func(t *testing.T) {
			f := providerTest(t, "go", "")
			wire := &deadlineWire{mode: mode}
			client := attachDeadlines(t, f, wire)
			f.start()
			f.revoke(true)
			if err := f.provider.Stop(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
				t.Fatal(err)
			}
			if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil {
				t.Fatal("uncertain deletion was reported complete")
			}
			if wire.deletes != 1 {
				t.Fatal("first delete missing")
			}
			if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil || wire.deletes != 1 {
				t.Fatal("delete backoff was ignored")
			}
			client.Close()
			f.reopen()
			f.now = f.now.Add(31 * time.Second)
			attachDeadlines(t, f, wire)
			wire.resource = nil
			if err := f.provider.Remove(t.Context(), f.job.Reference); err != nil || wire.deletes != 1 {
				t.Fatal("confirmed absence did not recover", err)
			}
		})
	}
}

func TestAzureDeadlineConcurrentDeletionRetainsOneBackoffReservation(t *testing.T) {
	f := providerTest(t, "go", "")
	wire := &deadlineWire{mode: "delete-pending"}
	client := attachDeadlines(t, f, wire)
	f.start()
	f.revoke(true)
	if err := f.provider.Stop(t.Context(), f.job.Reference); err != nil {
		t.Fatal(err)
	}
	if _, err := f.provider.Observe(t.Context(), f.job.Reference); err != nil {
		t.Fatal(err)
	}
	if err := f.provider.Remove(t.Context(), f.job.Reference); err == nil {
		t.Fatal("pending deletion was reported complete")
	}
	f.now = f.now.Add(31 * time.Second)
	var group sync.WaitGroup
	for range 6 {
		group.Go(func() {
			if err := client.Remove(t.Context(), wire.spec); err == nil {
				t.Error("pending timer deletion completed")
			}
		})
	}
	group.Wait()
	if wire.deletes != 2 {
		t.Fatal("controllers repeated one reserved deletion", wire.deletes)
	}
}

func TestAzureDeadlineSavedProfileAndSpecificationCannotBeReplaced(t *testing.T) {
	f := providerTest(t, "go", "")
	wire := &deadlineWire{}
	client := attachDeadlines(t, f, wire)
	spec := prepareDeadline(t, f, client)
	if err := client.Ensure(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	reads := wire.reads
	changed := spec
	changed.At = changed.At.Add(time.Second)
	requireDeadlineError(t, client.Verify(t.Context(), changed), hostedexecution.ErrOwnership)
	changed = spec
	changed.Specification = changed.Specification[:63] + "g"
	requireDeadlineError(t, client.Verify(t.Context(), changed), hostedexecution.ErrOwnership)
	original := client.config
	client.config.StopIdentity = client.config.Worker.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "replacement-stop")
	requireDeadlineError(t, client.Verify(t.Context(), spec), hostedexecution.ErrOwnership)
	client.config = original
	if wire.reads != reads || wire.creates != 1 {
		t.Fatal("changed identity or proof reached the provider")
	}
	if err := client.Verify(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
}
