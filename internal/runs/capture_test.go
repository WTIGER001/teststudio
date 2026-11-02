package runs_test

import (
	"errors"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestCapturePoliciesFreezeSuiteTestAndKindRetention(t *testing.T) {
	s, p, _ := setup(t, nil)
	projectPolicy := capture.DefaultPolicy()
	projectPolicy.RetentionDays = 7
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "project", Name: "Project", Capture: &projectPolicy}, 1); err != nil {
		t.Fatal(err)
	}
	r, err := s.Store.Get(t.Context(), projects.Bucket("suites", "project"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, err := storage.Decode[projects.Suite](r)
	if err != nil {
		t.Fatal(err)
	}
	suitePolicy := capture.DefaultPolicy()
	suitePolicy.RetentionDays = 14
	suitePolicy.KindRetention = map[string]int{"log": 4}
	suite.Capture = &suitePolicy
	if err := p.SaveSuite(t.Context(), "admin", "project", suite, r.Revision); err != nil {
		t.Fatal(err)
	}
	r, err = s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	test, err := storage.Decode[catalog.Test](r)
	if err != nil {
		t.Fatal(err)
	}
	testPolicy := capture.DefaultPolicy()
	testPolicy.Logs = false
	testPolicy.RetentionDays = 5
	test.Capture = &testPolicy
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, r.Revision); err != nil {
		t.Fatal(err)
	}
	draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil || draft.Capture.RetentionDays != 14 {
		t.Fatalf("suite override: %+v %v", draft.Capture, err)
	}
	if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "capture", 1, true); err != nil {
		t.Fatal(err)
	}
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "capture", "manual")
	if err != nil {
		t.Fatal(err)
	}
	if run.Capture.RetentionDays != 14 || run.TestCapture["HEALTH-1"].RetentionDays != 5 || run.TestCapture["HEALTH-1"].Logs {
		t.Fatalf("capture snapshot: %+v", run)
	}
	projectPolicy.RetentionDays = 1
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "project", Name: "Project", Capture: &projectPolicy}, 2); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.Outcome != "passed" || result.EvidenceState != "complete" {
		t.Fatalf("capture execution: %+v", result)
	}
	rows, err := s.Store.All(t.Context(), evidence.Bucket(run.ID), "project")
	if err != nil || len(rows) != 2 {
		t.Fatalf("test log opt-out: %d %v", len(rows), err)
	}
	for _, row := range rows {
		a, err := storage.Decode[evidence.Artifact](row)
		if err != nil || a.Test != "" || a.Policy == nil || a.Policy.RetentionDays != 14 || a.ExpiresAt.Sub(a.CreatedAt) < 4*24*time.Hour-time.Second || a.ExpiresAt.Sub(a.CreatedAt) > 4*24*time.Hour+time.Second {
			t.Fatalf("artifact policy: %+v %v", a, err)
		}
	}
}

func TestRunCaptureOverrideRequiresCurrentPermissionAndProjectLimits(t *testing.T) {
	s, _, _ := setup(t, nil)
	if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "operator", Name: "Operator"}, 0); err != nil {
		t.Fatal(err)
	}
	principal := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	if err := s.Auth.Engine.AssignRole(t.Context(), principal, "builtin.teststudio.operator", map[string]any{"team": "project"}); err != nil {
		t.Fatal(err)
	}
	policy := capture.DefaultPolicy()
	policy.Logs = false
	req := runs.Request{Suite: "health", Environment: "local", Capture: &policy}
	if _, err := s.Create(t.Context(), "operator", "project", req, "blocked", "manual"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("unprivileged capture override: %v", err)
	}
	if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "operator", TeamScope: "project", PermissionName: "teststudio.evidence.override-capture", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	run, err := s.Create(t.Context(), "operator", "project", req, "allowed", "manual")
	if err != nil || run.Capture.Logs || run.TestCapture["HEALTH-1"].Logs {
		t.Fatalf("run override: %+v %v", run, err)
	}
	policy.MaxBytes = 2 << 30
	if _, err := s.Create(t.Context(), "operator", "project", req, "too-large", "manual"); err == nil {
		t.Fatal("capture override exceeded project limits")
	}
	if err := s.Auth.Engine.DeleteGrantsForOwner(t.Context(), permissions.PrincipalUser, "operator"); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.State != "blocked" || result.Outcome != "unknown" || result.Counts.Completed != 0 {
		t.Fatalf("revoked capture override executed: %+v", result)
	}
}

func TestQueuedRunChecksTightenedCaptureLimitsBeforeDispatch(t *testing.T) {
	s, p, count := setup(t, nil)
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "before-limit-change", "manual")
	if err != nil {
		t.Fatal(err)
	}
	limits := capture.DefaultLimits()
	limits.RunBytes = 1 << 20
	policy := capture.DefaultPolicy()
	policy.MaxBytes = 1 << 20
	policy.MaxFileBytes = 64 << 10
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "project", Name: "Project", Capture: &policy, CaptureLimits: &limits}, 1); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.State != "blocked" || count.Load() != 0 {
		t.Fatalf("queued run bypassed new limits: %+v", result)
	}
}

func TestNewlyProtectedEnvironmentRejectsUnsafeQueuedRelease(t *testing.T) {
	s, p, count := setup(t, nil)
	r, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	test, err := storage.Decode[catalog.Test](r)
	if err != nil {
		t.Fatal(err)
	}
	test.Safety = "unsafe"
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, r.Revision); err != nil {
		t.Fatal(err)
	}
	draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "unsafe", 1, true); err != nil {
		t.Fatal(err)
	}
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "before-protection-change", "manual")
	if err != nil {
		t.Fatal(err)
	}
	env := *run.Environment
	env.Protected = true
	if err := p.SaveEnvironment(t.Context(), "admin", "project", env, 1); err != nil {
		t.Fatal(err)
	}
	start(t, s)
	result := finished(t, s, run.ID)
	if result.State != "blocked" || count.Load() != 0 {
		t.Fatalf("queued unsafe release bypassed protection: %+v", result)
	}
}
