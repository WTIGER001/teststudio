package azuresandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestSandboxControllerRequiresDeadlineAndFreshAuthorizationBeforeCreating(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	ref := f.plan.proof.Reference
	f.controls.ensureErr = errors.New("lost deadline acknowledgment")
	if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable || f.api.createCalls != 0 {
		t.Fatal("unverified deadline allowed a sandbox")
	}
	f.reopen()
	f.controls.ready = true
	f.denied.Store(true)
	if err := f.controller.Advance(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.api.createCalls != 0 {
		t.Fatal("revoked publication authority allowed a sandbox")
	}
	f.denied.Store(false)
	if err := f.controller.Advance(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	state, err := f.controller.Observe(t.Context(), ref)
	if err != nil || state.AdmissionPending || state.Instance.State != "Running" || state.Quiesced {
		t.Fatal("owned sandbox failed admission", err)
	}
	row, err := f.store.Get(t.Context(), workerexchange.Bucket, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	work, err := storage.Decode[workerexchange.Record](row)
	if err != nil || work.DispatchReady {
		t.Fatal("dispatch released executable input before explicit activation")
	}
	if err := f.exchange.Activate(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if err := f.controller.Ensure(t.Context(), f.plan); err != nil || f.api.createCalls != 1 || f.controls.ensureCalls != 1 {
		t.Fatal("retry redispatched an activated attempt", err)
	}
}

func TestSandboxControllerReopensLostReceiptsWithoutRedispatchOrTicketPersistence(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				f := newControllerFixture(t, runner, operation)
				f.api.lostCreate = true
				ref := f.plan.proof.Reference
				if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable {
					t.Fatal("lost create acknowledgment became a known outcome", err)
				}
				row, err := f.store.Get(t.Context(), DispatchBucket, ref.ID)
				if err != nil || strings.Contains(string(row.Data), f.plan.token) {
					t.Fatal("durable dispatch exposed a usable ticket", err)
				}
				f.reopen()
				if err := f.controller.Advance(t.Context(), ref); err != nil {
					t.Fatal(err)
				}
				state, err := f.controller.Observe(t.Context(), ref)
				if err != nil || state.Instance.ID != "native-sandbox" || state.AdmissionPending || f.api.createCalls != 1 {
					t.Fatal("lost receipt recovery repeated creation", err)
				}
				refs, err := f.controller.Inventory(t.Context(), ref.Installation, ref.Profile)
				if err != nil || len(refs) != 1 || refs[0] != ref {
					t.Fatal("durable inventory lost the owned attempt", err)
				}
				other := *f.plan
				other.proof.Fingerprint = strings.Repeat("f", 64)
				if err := f.controller.Ensure(t.Context(), &other); err != hostedexecution.ErrOwnership {
					t.Fatal("conflicting dispatch proof reused the attempt")
				}
			})
		}
	}
}

func TestSandboxControllerConcurrentReplicasReserveOnlyOneCreate(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	var replicas sync.WaitGroup
	for range 12 {
		replicas.Go(func() {
			err := f.controller.Ensure(t.Context(), f.plan)
			if err != nil && err != hostedexecution.ErrUnavailable {
				t.Errorf("unexpected concurrent dispatch error: %v", err)
			}
		})
	}
	replicas.Wait()
	if err := f.controller.Advance(t.Context(), f.plan.proof.Reference); err != nil {
		t.Fatal(err)
	}
	if f.api.createCalls != 1 || f.controls.ensureCalls != 1 {
		t.Fatal("concurrent replicas repeated an external mutation")
	}
}

func TestSandboxControllerRetainsUnknownCreationAndItsDeadlineDuringCleanup(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	ref := f.plan.proof.Reference
	f.api.emptyCreate = true
	if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable {
		t.Fatal(err)
	}
	f.retainCheckpoint()
	if err := f.controller.Stop(t.Context(), ref); err != hostedexecution.ErrUnavailable {
		t.Fatal("empty inventory proved an uncertain create never happened", err)
	}
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.controls.removeCalls != 0 {
		t.Fatal("cleanup removed the deadline before creation was reconciled", err)
	}
	f.reopen()
	if err := f.controller.Advance(t.Context(), ref); err != hostedexecution.ErrRejected || f.api.createCalls != 1 {
		t.Fatal("canceled uncertain work was restarted", err)
	}
	d, _, _, err := f.controller.load(t.Context(), ref)
	if err != nil || d.Removed || !d.Stopping || !d.CreateAttempted || d.Receipt != "" {
		t.Fatal("unresolved ownership was discarded", err)
	}
}

func TestSandboxControllerSeparatesStopAcknowledgmentCheckpointsAndRemoval(t *testing.T) {
	f := newControllerFixture(t, "playwright", "")
	ref := f.plan.proof.Reference
	if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
		t.Fatal(err)
	}
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.api.deleteCalls != 0 {
		t.Fatal("missing result checkpoint allowed deletion")
	}
	if err := f.controller.Stop(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	state, err := f.controller.Observe(t.Context(), ref)
	if err != nil || state.Quiesced || state.Instance.State != "Stopping" || !state.AdmissionPending {
		t.Fatal("accepted stop became confirmed quiescence", err)
	}
	f.retainCheckpoint()
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.api.deleteCalls != 0 {
		t.Fatal("running sandbox was deleted without quiescence")
	}
	f.instanceState("Stopped")
	state, err = f.controller.Observe(t.Context(), ref)
	if err != nil || !state.Quiesced || state.Instance.State != "Stopped" || !state.AdmissionPending {
		t.Fatal("quiescence became executable admission or a passing result", err)
	}
	f.api.keepDeleted = true
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.api.deleteCalls != 1 || f.controls.removeCalls != 0 {
		t.Fatal("accepted deletion removed the independent deadline")
	}
	f.reopen()
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.api.deleteCalls != 1 {
		t.Fatal("cleanup ignored durable deletion backoff")
	}
	f.api.mu.Lock()
	delete(f.api.items, "native-sandbox")
	f.api.mu.Unlock()
	f.controls.removeLost = true
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable {
		t.Fatal("lost deadline cleanup acknowledgment was discarded")
	}
	f.reopen()
	if err := f.controller.Remove(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if err := f.controller.Remove(t.Context(), ref); err != nil || f.api.deleteCalls != 1 || f.controls.removeCalls != 2 {
		t.Fatal("completed cleanup repeated a resource operation", err)
	}
	refs, err := f.controller.Inventory(t.Context(), ref.Installation, ref.Profile)
	if err != nil || len(refs) != 0 {
		t.Fatal("removed attempt remained active", err)
	}
}

func TestSandboxControllerRejectsChangedOwnershipAndDuplicateLostReceipts(t *testing.T) {
	t.Run("changed owned instance", func(t *testing.T) {
		f := newControllerFixture(t, "go", "")
		if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
			t.Fatal(err)
		}
		f.api.mu.Lock()
		value := f.api.items["native-sandbox"]
		value.DefinitionHash = strings.Repeat("d", 64)
		f.api.items[value.ID] = value
		f.api.mu.Unlock()
		if err := f.controller.Stop(t.Context(), f.plan.proof.Reference); err != hostedexecution.ErrOwnership || f.api.stopCalls != 0 {
			t.Fatal("changed native execution was controlled as an owned resource")
		}
	})
	t.Run("duplicate receipts", func(t *testing.T) {
		f := newControllerFixture(t, "go", "")
		f.api.lostCreate = true
		if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable {
			t.Fatal(err)
		}
		f.api.mu.Lock()
		copy := f.api.items["native-sandbox"]
		copy.ID = "duplicate-sandbox"
		f.api.items[copy.ID] = copy
		f.api.mu.Unlock()
		f.reopen()
		if _, err := f.controller.Observe(t.Context(), f.plan.proof.Reference); err != hostedexecution.ErrOwnership || f.api.createCalls != 1 {
			t.Fatal("ambiguous native instances were silently adopted")
		}
	})
}

func TestSandboxControllerAdmissionLossAndCrossProjectReferencesRemainBlocked(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	ref := f.plan.proof.Reference
	if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
		t.Fatal(err)
	}
	f.controls.instanceErr = fmt.Errorf("private-placement-canary: %w", hostedexecution.ErrUnavailable)
	state, err := f.controller.Observe(t.Context(), ref)
	if err != nil || !state.AdmissionPending {
		t.Fatal("missing placement admission released work", err)
	}
	f.controls.instanceErr = fmt.Errorf("private-placement-canary: %w", hostedexecution.ErrOwnership)
	if _, err := f.controller.Observe(t.Context(), ref); err != hostedexecution.ErrOwnership {
		t.Fatal("placement ownership rejection was lost or exposed private diagnostics", err)
	}
	f.controls.instanceErr = nil
	f.controls.ready = false
	state, err = f.controller.Observe(t.Context(), ref)
	if err != nil || !state.AdmissionPending {
		t.Fatal("lost independent deadline released work", err)
	}
	foreign := ref
	foreign.Project = "different-project"
	if err := f.controller.Stop(t.Context(), foreign); err != hostedexecution.ErrOwnership || f.api.stopCalls != 0 {
		t.Fatal("cross-project reference controlled the sandbox")
	}
	data, _ := json.Marshal(state)
	if strings.Contains(string(data), f.plan.token) {
		t.Fatal("control observation exposed a ticket")
	}
	row, err := f.store.Get(t.Context(), DispatchBucket, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	d, err := storage.Decode[dispatch](row)
	if err != nil {
		t.Fatal(err)
	}
	d.Removed = true
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, d)
	if err != nil || f.store.Commit(t.Context(), change) != nil {
		t.Fatal("cannot simulate an invalid removed checkpoint", err)
	}
	if _, err := f.controller.Inventory(t.Context(), ref.Installation, ref.Profile); err != hostedexecution.ErrOwnership {
		t.Fatal("an invalid removed flag hid an active execution", err)
	}
}
