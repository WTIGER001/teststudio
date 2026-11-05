package azuresandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestSandboxControllerCancellationFencesAnInflightCreateAndKeepsItsDeadline(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	ref := f.plan.proof.Reference
	f.api.createEntered = make(chan struct{})
	f.api.releaseCreate = make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(f.api.releaseCreate) }) }
	finished := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		finished <- f.controller.Ensure(t.Context(), f.plan)
	}()
	defer func() { release(); <-done }()
	select {
	case <-f.api.createEntered:
	case err := <-finished:
		t.Fatal("create did not reach the controlled boundary", err)
	}
	f.retainCheckpoint()
	if err := f.controller.Stop(t.Context(), ref); err != hostedexecution.ErrUnavailable {
		t.Fatal("an in-flight create became absent", err)
	}
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.controls.removeCalls != 0 {
		t.Fatal("in-flight creation lost its independent deadline", err)
	}
	release()
	if err := <-finished; err != nil {
		t.Fatal("late receipt was not retained", err)
	}
	state, err := f.controller.Observe(t.Context(), ref)
	if err != nil || !state.AdmissionPending || state.Instance.ID != "native-sandbox" {
		t.Fatal("canceled late creation released work", err)
	}
	if err := f.controller.Advance(t.Context(), ref); err != hostedexecution.ErrRejected {
		t.Fatal("canceled attempt restarted")
	}
	if err := f.controller.Stop(t.Context(), ref); err != nil || f.api.stopCalls != 1 || f.api.createCalls != 1 {
		t.Fatal("late resource could not be stopped without redispatch", err)
	}
}

func TestSandboxControllerBindsTheExactPersistedWorkItem(t *testing.T) {
	for _, operation := range []string{"", "discover"} {
		t.Run(operation, func(t *testing.T) {
			f := newControllerFixture(t, "go", operation)
			ref := f.plan.proof.Reference
			row, err := f.store.Get(t.Context(), workerexchange.Bucket, ref.ID)
			if err != nil {
				t.Fatal(err)
			}
			work, err := storage.Decode[workerexchange.Record](row)
			if err != nil {
				t.Fatal(err)
			}
			var input map[string]any
			if err := json.Unmarshal(work.Input, &input); err != nil {
				t.Fatal(err)
			}
			wrong := f.plan.config.Registry + "/" + f.plan.config.RepositoryPrefix + "/different/source@sha256:" + strings.Repeat("d", 64)
			if operation == "discover" {
				input["image"] = wrong
			} else {
				input["artifact"].(map[string]any)["image"] = wrong
			}
			work.Input, err = json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(work.Input)
			work.InputHash = hex.EncodeToString(sum[:])
			change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, work)
			if err != nil || f.store.Commit(t.Context(), change) != nil {
				t.Fatal("cannot simulate a mismatched work item", err)
			}
			if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrOwnership || f.api.createCalls != 0 || f.controls.ensureCalls != 0 {
				t.Fatal("different saved source allowed a dispatch", err)
			}
		})
	}
	t.Run("reserved input changed", func(t *testing.T) {
		f := newControllerFixture(t, "go", "")
		ref := f.plan.proof.Reference
		f.controls.ensureErr = hostedexecution.ErrUnavailable
		if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable {
			t.Fatal(err)
		}
		row, err := f.store.Get(t.Context(), workerexchange.Bucket, ref.ID)
		if err != nil {
			t.Fatal(err)
		}
		work, err := storage.Decode[workerexchange.Record](row)
		if err != nil {
			t.Fatal(err)
		}
		work.Input = append(work.Input, ' ')
		sum := sha256.Sum256(work.Input)
		work.InputHash = hex.EncodeToString(sum[:])
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, work)
		if err != nil || f.store.Commit(t.Context(), change) != nil {
			t.Fatal("cannot simulate changed reserved bytes", err)
		}
		f.controls.ready = true
		if err := f.controller.Advance(t.Context(), ref); err != hostedexecution.ErrOwnership || f.api.createCalls != 0 {
			t.Fatal("changed reserved work bytes allowed a dispatch", err)
		}
	})
}
