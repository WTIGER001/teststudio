package runs_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type policyChangingHTTP struct {
	calls   atomic.Int32
	change  func() error
	problem chan error
}

func (e *policyChangingHTTP) Execute(_ context.Context, _ string, test catalog.Test, _ projects.Environment) execution.Result {
	if e.calls.Add(1) == 1 {
		e.problem <- e.change()
	}
	now := time.Now().UTC()
	return execution.Result{TestID: test.ID, State: "failed", FailureKind: "http-status", Error: "Owned retry fixture requests a second attempt", StartedAt: now.Add(-time.Millisecond), FinishedAt: now, DurationMs: 1}
}

func TestAttemptAdmissionStopsRetriesAfterSafetyOrPermissionChanges(t *testing.T) {
	for _, mutation := range []string{"safety", "disabled", "protected-permission"} {
		t.Run(mutation, func(t *testing.T) {
			s, p, _ := setup(t, nil)
			environment, revision := currentEnvironment(t, s)
			environment.Protected = true
			if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
				t.Fatal(err)
			}
			executor := &policyChangingHTTP{problem: make(chan error, 1), change: func() error {
				switch mutation {
				case "safety":
					definition, revision := currentTest(t, s, "HEALTH-1")
					definition.Safety = "unsafe"
					return s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, revision)
				case "disabled":
					return s.Catalog.ChangeState(t.Context(), "admin", "project", "health", "HEALTH-1", "disabled", 1)
				default:
					return s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "project", PermissionName: "teststudio.runs.execute-protected", Effect: permissions.EffectDeny})
				}
			}}
			s.HTTP = executor
			run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local", Retries: 1}, "retry-policy", "manual")
			if err != nil {
				t.Fatal(err)
			}
			start(t, s)
			final := finished(t, s, run.ID)
			if err := <-executor.problem; err != nil {
				t.Fatal(err)
			}
			if executor.calls.Load() != 1 || final.State != "blocked" || final.Error == "" || final.Outcome != "unknown" {
				t.Fatal("retry ignores changed safety or permissions", final.State, final.Error)
			}
			attempts, err := s.Store.All(t.Context(), "attempts/"+run.ID, "project")
			if err != nil || len(attempts) != 1 {
				t.Fatal("blocked retry loses its recorded first observation", len(attempts), err)
			}
			observed, err := storage.Decode[execution.Result](attempts[0])
			if err != nil || observed.State != "failed" || observed.Attempt != 1 {
				t.Fatal("blocked retry rewrites its observed attempt", err)
			}
			if mutation == "protected-permission" && !strings.Contains(final.Error, "permission") {
				t.Fatal("revoked protected permission lacks a readable reason")
			}
		})
	}
}
