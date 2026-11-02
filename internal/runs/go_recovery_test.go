package runs_test

import (
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

type goRecoveryExecutor struct{ *recoveryExecutor }

func (e *goRecoveryExecutor) Profile() string { return "go-local-isolated-v1" }
func (e *goRecoveryExecutor) Reference(ref execution.Reference) execution.Reference {
	ref = e.recoveryExecutor.Reference(ref)
	ref.Profile = e.Profile()
	return ref
}

func TestRecoverySelectsTheRecordedGoProfileWhenNewDispatchIsDisabled(t *testing.T) {
	s, _, _ := setup(t, nil)
	playwright := recoveredDriver("passed")
	goDriver := &goRecoveryExecutor{recoveredDriver("failed")}
	run, ref := lostExecution(t, s, goDriver, false)
	current, row, err := s.Get(t.Context(), run.Project, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.Tests[0].Runner = "go"
	current.Tests[0].Selector = "./unit::TestFresh"
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, current)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	s.RecoveryDriver = playwright
	s.RecoveryDrivers = []execution.ManagedExecutor{goDriver}
	s.RecoveryInterval = 5 * time.Millisecond
	start(t, s)
	record := awaitExecution(t, s, ref, "cleaned")
	if goDriver.recoveries.Load() != 1 || goDriver.cleanups.Load() != 1 || playwright.recoveries.Load() != 0 || playwright.cleanups.Load() != 0 || record.RecoveredResult == nil || record.RecoveredResult.State != "failed" {
		t.Fatalf("wrong recovery driver: %+v", record)
	}
	final, _, err := s.Get(t.Context(), run.Project, run.ID)
	if err != nil || final.State != "incomplete" || final.Outcome != "unknown" || final.Counts.Completed != 0 {
		t.Fatalf("recovery changed the interrupted logical outcome: %+v %v", final, err)
	}
}
