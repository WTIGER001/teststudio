package runs_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func setup(t *testing.T, handler http.Handler) (*runs.Service, *projects.Service, *atomic.Int32) {
	t.Helper()
	root := t.TempDir()
	db, err := testkit.Open(t, filepath.Join(root, "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close(); db.Close() })
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	count := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if handler != nil {
			handler.ServeHTTP(w, r)
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "up"})
		}
	}))
	t.Cleanup(server.Close)
	p := &projects.Service{Store: db, Auth: auth}
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "project", Name: "Project"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveApplication(t.Context(), "admin", "project", projects.Application{ID: "app", Name: "Application"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveEnvironment(t.Context(), "admin", "project", projects.Environment{ID: "local", Name: "Local", Application: "app", Stage: "local", Endpoints: map[string]string{"api": server.URL}}, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveSuite(t.Context(), "admin", "project", projects.Suite{ID: "health", Name: "Health", Applications: []string{"app"}, RequiresTarget: true}, 0); err != nil {
		t.Fatal(err)
	}
	c := &catalog.Service{Store: db, Auth: auth}
	test := catalog.Test{ID: "HEALTH-1", Title: "Health", Application: "app", Category: "Availability", Type: "https", Runner: "http", State: "enabled", Safety: "safe", Rationale: "Reads the local health endpoint.", TimeoutSeconds: 10, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/healthz", Method: "GET", StatusCodes: []int{200}, JSON: map[string]any{"status": "up"}}}
	if err := c.SaveTest(t.Context(), "admin", "project", "health", test, 0); err != nil {
		t.Fatal(err)
	}
	d, err := c.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Publish(t.Context(), "admin", "project", "health", d.ID, "release", 1, true); err != nil {
		t.Fatal(err)
	}
	s := &runs.Service{Store: db, Auth: auth, Catalog: c, Evidence: &evidence.Service{Store: db, Blobs: files}, HTTP: &execution.HTTP{}, Location: "local", Concurrency: 1, PollInterval: 5 * time.Millisecond, HeartbeatInterval: 10 * time.Millisecond}
	return s, p, count
}

func start(t *testing.T, s *runs.Service) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := s.Start(ctx)
	stop := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("workers did not stop")
		}
	}
	t.Cleanup(stop)
	return stop
}

func finished(t *testing.T, s *runs.Service, id string) runs.Run {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		v, _, err := s.Get(t.Context(), "project", id)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(v.State) {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("run did not finish")
	return runs.Run{}
}

func TestRunFreezesEnvironmentAndTwoWorkersExecuteOnce(t *testing.T) {
	s, p, count := setup(t, nil)
	req := runs.Request{Suite: "health", Environment: "local"}
	run, err := s.Create(t.Context(), "admin", "project", req, "one", "manual")
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Create(t.Context(), "admin", "project", req, "one", "manual")
	if err != nil || again.ID != run.ID {
		t.Fatal("run request was not idempotent")
	}
	changed := *run.Environment
	changed.Endpoints = map[string]string{"api": "http://127.0.0.1:1"}
	if err := p.SaveEnvironment(t.Context(), "admin", "project", changed, 1); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	start(t, s)
	result := finished(t, s, run.ID)
	if result.State != "completed" || result.Outcome != "passed" || result.Counts.Passed != 1 || result.EvidenceState != "complete" {
		t.Fatalf("run: %+v", result)
	}
	if count.Load() != 1 {
		t.Fatalf("duplicate execution: %d", count.Load())
	}
	files, err := s.Store.All(t.Context(), evidence.Bucket(run.ID), "project")
	if err != nil || len(files) != 4 {
		t.Fatalf("captured logs: %d %v", len(files), err)
	}
	if len(result.Release.Draft.Tests) != 1 || result.Environment.Endpoints["api"] == changed.Endpoints["api"] {
		t.Fatal("run snapshot changed")
	}
}

type retryExecutor struct{ calls atomic.Int32 }

func (e *retryExecutor) Execute(_ context.Context, _ string, test catalog.Test, _ projects.Environment) execution.Result {
	now := time.Now().UTC()
	state := "passed"
	if e.calls.Add(1) == 1 {
		state = "failed"
	}
	return execution.Result{TestID: test.ID, State: state, StartedAt: now, FinishedAt: now}
}

func TestRetriesRemainFlakyWithoutInflatingCounts(t *testing.T) {
	s, _, _ := setup(t, nil)
	executor := &retryExecutor{}
	s.HTTP = executor
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local", Retries: 1}, "retry", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.Outcome != "flaky" || result.Counts.Flaky != 1 || result.Counts.Total != 1 || result.Counts.Passed != 0 {
		t.Fatalf("flaky reduction: %+v", result.Counts)
	}
	attempts, err := s.Store.All(t.Context(), "attempts/"+run.ID, "project")
	if err != nil || len(attempts) != 2 || executor.calls.Load() != 2 {
		t.Fatalf("attempt records: %d calls=%d %v", len(attempts), executor.calls.Load(), err)
	}
}

func TestQueuedCancellationAndLostWorkerNeverPass(t *testing.T) {
	s, _, count := setup(t, nil)
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "cancel", "manual")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancel(t.Context(), "admin", "project", run.ID); err != nil {
		t.Fatal(err)
	}
	stop := start(t, s)
	result := finished(t, s, run.ID)
	if result.State != "cancelled" || result.Outcome != "unknown" || count.Load() != 0 {
		t.Fatalf("cancelled execution: %+v", result)
	}
	stop()
	lost, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "lost", "manual")
	if err != nil {
		t.Fatal(err)
	}
	_, row, err := s.Get(t.Context(), "project", lost.ID)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := s.Store.Get(t.Context(), "run-queue", lost.ID)
	if err != nil {
		t.Fatal(err)
	}
	lost.State = "running"
	lost.Worker = "stopped"
	expired := time.Now().Add(-time.Minute)
	lost.LeaseUntil = &expired
	change, _ := storage.Put("runs", lost.ID, "project", row.Revision, lost)
	lease, _ := storage.Put("run-leases", lost.ID, "project", 0, lost.ID)
	if err := s.Store.Commit(t.Context(), change, lease, storage.Change{Record: queue, Delete: true}); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result = finished(t, s, lost.ID)
	if result.State != "incomplete" || result.Outcome != "unknown" || count.Load() != 0 {
		t.Fatalf("lost worker was counted as a pass: %+v", result)
	}
}

type failedBlobs struct{ artifacts.Store }

func (s failedBlobs) Put(context.Context, string, io.Reader, int64) (artifacts.Object, error) {
	return artifacts.Object{}, errors.New("artifact storage failed")
}

func TestCaptureFailureDoesNotRewriteObservedOutcome(t *testing.T) {
	s, _, _ := setup(t, nil)
	s.Evidence.Blobs = failedBlobs{Store: s.Evidence.Blobs}
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "evidence", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.Outcome != "passed" || result.EvidenceState != "incomplete" || len(result.EvidenceErrors) == 0 {
		t.Fatalf("capture changed test outcome: %+v", result)
	}
}

func TestRunningCancellationRetainsTheInterruptedAttempt(t *testing.T) {
	s, _, count := setup(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "running-cancel", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	deadline := time.Now().Add(3 * time.Second)
	for count.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if count.Load() != 1 {
		t.Fatal("check did not start")
	}
	if _, err := s.Cancel(t.Context(), "admin", "project", run.ID); err != nil {
		t.Fatal(err)
	}
	result := finished(t, s, run.ID)
	if result.State != "cancelled" || result.Outcome != "unknown" {
		t.Fatalf("running cancellation: %+v", result)
	}
	attempts, err := s.Store.All(t.Context(), "attempts/"+run.ID, "project")
	if err != nil || len(attempts) != 1 {
		t.Fatalf("interrupted attempt disappeared: %d %v", len(attempts), err)
	}
}

func TestRevokedExecutionPermissionBlocksQueuedWork(t *testing.T) {
	s, _, count := setup(t, nil)
	if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "operator", Name: "Operator"}, 0); err != nil {
		t.Fatal(err)
	}
	principal := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	if err := s.Auth.Engine.AssignRole(t.Context(), principal, "builtin.teststudio.operator", map[string]any{"team": "project"}); err != nil {
		t.Fatal(err)
	}
	run, err := s.Create(t.Context(), "operator", "project", runs.Request{Suite: "health", Environment: "local"}, "revocation", "manual")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Engine.UnassignRole(t.Context(), principal, "builtin.teststudio.operator", map[string]any{"team": "project"}); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.State != "blocked" || result.Outcome != "unknown" || count.Load() != 0 {
		t.Fatalf("revoked user executed work: %+v", result)
	}
}
