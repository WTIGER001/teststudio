package runs_test

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type timeoutExecutor struct {
	mu    sync.Mutex
	calls map[string]int
}

func (e *timeoutExecutor) Execute(_ context.Context, test catalog.Test, _ projects.Environment) execution.Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls[test.ID]++
	now := time.Now().UTC()
	r := execution.Result{TestID: test.ID, State: "passed", StartedAt: now.Add(-time.Second), FinishedAt: now, DurationMs: 1000}
	switch test.ID {
	case "HEALTH-1":
		r.State, r.FailureKind = "failed", "timeout"
	case "HEALTH-2":
		r.State, r.FailureKind = "failed", "assertion"
	case "HEALTH-4":
		if e.calls[test.ID] == 1 {
			r.State, r.FailureKind = "timed-out", "timeout"
		}
	}
	return r
}

func addTimeoutDefinitions(t *testing.T, s *runs.Service, ids ...string) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	definition, err := storage.Decode[catalog.Test](row)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		definition.ID = id
		if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, 0); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "timeout-release", 1, true); err != nil {
		t.Fatal(err)
	}
}

func TestTimeoutsHaveSeparateCountsAndRerunAlongsideFailuresAndFlakyTests(t *testing.T) {
	s, _, _ := setup(t, nil)
	addTimeoutDefinitions(t, s, "HEALTH-2", "HEALTH-3", "HEALTH-4")
	s.HTTP = &timeoutExecutor{calls: map[string]int{}}
	parent, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local", Retries: 1}, "timeouts", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	parent = finished(t, s, parent.ID)
	if parent.State != "completed" || parent.Outcome != "failed" || parent.Counts != (runs.Counts{Total: 4, Passed: 1, Failed: 1, TimedOut: 1, Flaky: 1, Completed: 4}) {
		t.Fatalf("mixed outcomes: %+v", parent)
	}
	attempts, err := s.Store.All(t.Context(), "attempts/"+parent.ID, "project")
	if err != nil || len(attempts) != 7 {
		t.Fatalf("retry counts: %d %v", len(attempts), err)
	}
	for _, row := range attempts {
		result, err := storage.Decode[execution.Result](row)
		if err != nil {
			t.Fatal(err)
		}
		if result.FailureKind == "timeout" && (result.State != "timed-out" || result.Timing == nil || !result.Timing.Censored) {
			t.Fatalf("timeout attempt: %+v", result)
		}
	}
	child, err := s.RerunFailures(t.Context(), "admin", "project", parent.ID, "rerun-timeouts")
	if err != nil {
		t.Fatal(err)
	}
	ids := slices.Clone(child.Request.IDs)
	slices.Sort(ids)
	if !slices.Equal(ids, []string{"HEALTH-1", "HEALTH-2", "HEALTH-4"}) || child.Request.Parent != parent.ID || child.Release.ID != parent.Release.ID {
		t.Fatalf("rerun selection: %+v", child.Request)
	}
	finished(t, s, child.ID)
}

func TestRunDeadlineRetainsTimedOutTestForFailureRerun(t *testing.T) {
	s, _, _ := setup(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	addTimeoutDefinitions(t, s, "HEALTH-2")
	parent, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local", TimeoutSeconds: 1}, "deadline-timeout", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	parent = finished(t, s, parent.ID)
	if parent.State != "timed-out" || parent.Outcome != "timed-out" || parent.Counts.TimedOut != 1 || parent.Counts.Failed != 0 || parent.Counts.Completed != 1 || len(parent.Results) != 1 {
		t.Fatalf("deadline lost its timeout: %+v", parent)
	}
	if parent.Results[0].State != "timed-out" || parent.Results[0].FailureKind != "timeout" {
		t.Fatalf("deadline result: %+v", parent.Results[0])
	}
	child, err := s.RerunFailures(t.Context(), "admin", "project", parent.ID, "deadline-rerun")
	if err != nil || len(child.Request.IDs) != 1 || child.Request.IDs[0] != parent.Results[0].TestID {
		t.Fatalf("deadline rerun: %+v %v", child, err)
	}
	finished(t, s, child.ID)
}
