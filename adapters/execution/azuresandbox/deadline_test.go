package azuresandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxDeadlineWorkflowBindsIndependentScopeAndPrivateHistory(t *testing.T) {
	f := newTimerFixture(t, "go", "")
	actual := f.transport.decorate(f.config.definition(f.spec))
	if err := f.config.verify(f.spec, actual, true); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(actual)
	if strings.Contains(string(data), f.plan.token) || strings.Contains(string(data), "TESTSTUDIO_JOB_TOKEN") {
		t.Fatal("timer retains a usable ticket")
	}
	if !strings.Contains(string(data), `"audience":"https://dynamicsessions.io"`) {
		t.Fatal("timer audience differs from the pinned SDK resource")
	}
	definition := actual.Properties.Definition.(map[string]any)
	trigger := definition["triggers"].(map[string]any)["Deadline"].(map[string]any)
	recurrence := trigger["recurrence"].(map[string]any)
	due, err := time.Parse(time.RFC3339, recurrence["startTime"].(string))
	if err != nil || due.Before(f.spec.Proof.Deadline) || due.Sub(f.spec.Proof.Deadline) >= time.Second {
		t.Fatal("timer moves the absolute deadline")
	}
	if directory := os.Getenv("TESTSTUDIO_SANDBOX_DEADLINE_OUTPUT"); directory != "" {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		data, err := json.MarshalIndent(definition, "", "  ")
		if err != nil || os.WriteFile(filepath.Join(directory, "workflow.json"), data, 0o600) != nil {
			t.Fatal("cannot retain the generated workflow", err)
		}
	}
	changes := map[string]func(*l.Workflow){
		"disabled":      func(w *l.Workflow) { w.Properties.State = to.Ptr(l.WorkflowStateDisabled) },
		"identity":      func(w *l.Workflow) { w.Identity.UserAssignedIdentities = map[string]*l.UserAssignedIdentity{} },
		"timer-profile": func(w *l.Workflow) { w.Tags["teststudio-deadline-profile"] = to.Ptr(strings.Repeat("d", 64)) },
		"scope": func(w *l.Workflow) {
			actions := w.Properties.Definition.(map[string]any)["actions"].(map[string]any)
			actions["Initialize"].(map[string]any)["inputs"].(map[string]any)["variables"].([]any)[0].(map[string]any)["value"] = "https://unrelated.example/sandboxes"
		},
		"private-history": func(w *l.Workflow) {
			actions := w.Properties.Definition.(map[string]any)["actions"].(map[string]any)
			pages := actions["Pages"].(map[string]any)["actions"].(map[string]any)
			page := pages["CheckCursor"].(map[string]any)["actions"].(map[string]any)
			delete(page["List"].(map[string]any), "runtimeConfiguration")
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			w := f.transport.decorate(f.config.definition(f.spec))
			change(&w)
			if f.config.verify(f.spec, w, true) != hostedexecution.ErrOwnership {
				t.Fatal("changed timer was adopted")
			}
		})
	}
}

func TestSandboxDeadlineSDKReconcilesLostCreationBeforeAllowingDispatch(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				f := newTimerFixture(t, runner, operation)
				f.transport.lostCreate = true
				if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable || f.api.createCalls != 0 || f.transport.puts != 1 {
					t.Fatal("unverified timer allowed dispatch", err)
				}
				f.reopenTimer()
				if err := f.controller.Advance(t.Context(), f.plan.proof.Reference); err != nil || f.api.createCalls != 1 || f.transport.puts != 1 {
					t.Fatal("recovery repeated timer creation or lost dispatch", err)
				}
				d, _, _, err := f.controller.load(t.Context(), f.plan.proof.Reference)
				if err != nil || !d.TimerSeen || d.TimerProfile != f.config.profile() {
					t.Fatal("timer observation was not durable", err)
				}
			})
		}
	}
}

func TestSandboxDeadlineUnknownCreationCannotBeDiscardedDuringCleanup(t *testing.T) {
	f := newTimerFixture(t, "go", "")
	f.transport.emptyCreate = true
	if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable {
		t.Fatal(err)
	}
	f.reopenTimer()
	if err := f.controller.Advance(t.Context(), f.plan.proof.Reference); err != hostedexecution.ErrUnavailable || f.api.createCalls != 0 {
		t.Fatal("a missing timer proved the uncertain execution absent", err)
	}
	f.retainCheckpoint()
	if err := f.controller.Remove(t.Context(), f.plan.proof.Reference); err != hostedexecution.ErrUnavailable || f.transport.deletes != 0 {
		t.Fatal("an uncertain timer became absent", err)
	}
	f.reopenTimer()
	late := f.transport.decorate(f.config.definition(f.spec))
	f.transport.workflow = &late
	if err := f.controller.Remove(t.Context(), f.plan.proof.Reference); err != nil || f.transport.deletes != 1 || f.api.createCalls != 0 {
		t.Fatal("late timer could not be removed without dispatch", err)
	}
}

func TestSandboxDeadlineCleanupConfirmsWorkerAndTimerAbsenceAcrossRestart(t *testing.T) {
	f := newTimerFixture(t, "playwright", "")
	ref := f.plan.proof.Reference
	if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
		t.Fatal(err)
	}
	if err := f.timer.Remove(t.Context(), f.spec); err != hostedexecution.ErrUnavailable || f.transport.deletes != 0 {
		t.Fatal("timer cleanup bypassed result checkpoints")
	}
	f.retainCheckpoint()
	if err := f.timer.Remove(t.Context(), f.spec); err != hostedexecution.ErrUnavailable {
		t.Fatal("active sandbox lost its timer")
	}
	if err := f.controller.Stop(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	f.instanceState("Stopped")
	f.api.keepDeleted = true
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable || f.transport.deletes != 0 {
		t.Fatal("delete acknowledgment removed a timer")
	}
	f.api.mu.Lock()
	delete(f.api.items, "native-sandbox")
	f.api.mu.Unlock()
	f.transport.lostDelete = true
	if err := f.controller.Remove(t.Context(), ref); err != hostedexecution.ErrUnavailable {
		t.Fatal("lost timer deletion was forgotten")
	}
	f.reopenTimer()
	if err := f.controller.Remove(t.Context(), ref); err != nil || f.transport.deletes != 1 {
		t.Fatal("cleanup did not reconcile known timer absence", err)
	}
}

func TestSandboxDeadlineRejectsRedirectsAndPreexistingForeignTimers(t *testing.T) {
	t.Run("rejected create", func(t *testing.T) {
		f := newTimerFixture(t, "go", "")
		f.transport.rejectCreate = true
		if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrRejected || f.api.createCalls != 0 {
			t.Fatal("known timer rejection became an ambiguous dispatch", err)
		}
		f.retainCheckpoint()
		if err := f.controller.Remove(t.Context(), f.plan.proof.Reference); err != nil || f.transport.deletes != 0 {
			t.Fatal("rejected timer could not be cleared through confirmed absence", err)
		}
	})
	t.Run("redirect", func(t *testing.T) {
		f := newTimerFixture(t, "go", "")
		f.transport.redirect = true
		if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrUnavailable || f.transport.puts != 0 || f.api.createCalls != 0 {
			t.Fatal("redirected timer request allowed dispatch", err)
		}
	})
	t.Run("preexisting", func(t *testing.T) {
		f := newTimerFixture(t, "go", "")
		w := f.transport.decorate(f.config.definition(f.spec))
		f.transport.workflow = &w
		if err := f.controller.Ensure(t.Context(), f.plan); err != hostedexecution.ErrOwnership || f.transport.puts != 0 || f.api.createCalls != 0 {
			t.Fatal("an unreserved timer was overwritten or adopted", err)
		}
	})
}
