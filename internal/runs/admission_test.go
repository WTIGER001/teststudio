package runs_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
)

type admissionHook struct{ before func() }

func (h *admissionHook) Prepare(context.Context, runs.Run) ([]storage.Change, error) {
	if h.before != nil {
		before := h.before
		h.before = nil
		before()
	}
	return nil, nil
}
func (*admissionHook) Complete(context.Context, runs.Run) ([]storage.Change, error) { return nil, nil }

func currentEnvironment(t *testing.T, s *runs.Service) (projects.Environment, int64) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), projects.Bucket("environments", "project"), "local")
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[projects.Environment](row)
	if err != nil {
		t.Fatal(err)
	}
	return value, row.Revision
}

func currentTest(t *testing.T, s *runs.Service, id string) (catalog.Test, int64) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), id)
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[catalog.Test](row)
	if err != nil {
		t.Fatal(err)
	}
	return value, row.Revision
}

func publishAdmission(t *testing.T, s *runs.Service, key string) catalog.Release {
	t.Helper()
	draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	release, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, key, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	return release
}

func TestRunCreationAtomicallyGuardsReviewedCatalogSuiteAndEnvironment(t *testing.T) {
	for _, mutation := range []string{"test-disabled", "suite-archived", "environment-disabled", "protected-policy", "release-changed"} {
		t.Run(mutation, func(t *testing.T) {
			s, p, count := setup(t, nil)
			if mutation == "protected-policy" {
				definition, revision := currentTest(t, s, "HEALTH-1")
				definition.Safety = "unsafe"
				if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, revision); err != nil {
					t.Fatal(err)
				}
				publishAdmission(t, s, "unsafe-before-protection")
			}
			var latest catalog.Release
			s.Completion = &admissionHook{before: func() {
				switch mutation {
				case "test-disabled":
					if err := s.Catalog.ChangeState(t.Context(), "admin", "project", "health", "HEALTH-1", "disabled", 1); err != nil {
						t.Fatal(err)
					}
				case "suite-archived":
					row, err := s.Store.Get(t.Context(), projects.Bucket("suites", "project"), "health")
					if err != nil {
						t.Fatal(err)
					}
					suite, err := storage.Decode[projects.Suite](row)
					if err != nil {
						t.Fatal(err)
					}
					suite.Archived = true
					if err := p.SaveSuite(t.Context(), "admin", "project", suite, row.Revision); err != nil {
						t.Fatal(err)
					}
				case "environment-disabled", "protected-policy":
					environment, revision := currentEnvironment(t, s)
					if mutation == "protected-policy" {
						environment.Protected = true
					} else {
						environment.Disabled = true
					}
					if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
						t.Fatal(err)
					}
				case "release-changed":
					definition, revision := currentTest(t, s, "HEALTH-1")
					definition.Check.StatusCodes = []int{201}
					if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, revision); err != nil {
						t.Fatal(err)
					}
					latest = publishAdmission(t, s, "concurrent-release")
				}
			}}
			run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "atomic-admission", "manual")
			if mutation == "release-changed" {
				if err != nil || run.Release.ID != latest.ID || run.Tests[0].Check.StatusCodes[0] != 201 {
					t.Fatal("run commits a stale active release", err)
				}
			} else {
				if err == nil {
					t.Fatal("changed reviewed state admits a run")
				}
				for _, bucket := range []string{"runs", "run-queue", "run-requests"} {
					rows, err := s.Store.All(t.Context(), bucket, "project")
					if err != nil || len(rows) != 0 {
						t.Fatal("failed admission leaves partial records", bucket, err)
					}
				}
			}
			if count.Load() != 0 {
				t.Fatal("admission starts a test")
			}
		})
	}
}

func TestProtectedManualAndScheduledSelectionsCheckEveryDependency(t *testing.T) {
	for _, unsafe := range []string{"none", "root", "dependency", "dependency-needs-review", "current-dependency", "release-dependency"} {
		t.Run(unsafe, func(t *testing.T) {
			s, p, count := setup(t, nil)
			dependency, revision := currentTest(t, s, "HEALTH-1")
			root := dependency
			root.ID, root.Title, root.Dependencies = "ROOT", "Dependent check", []string{dependency.ID}
			if unsafe == "root" {
				root.Safety = "unsafe"
			}
			if strings.HasPrefix(unsafe, "dependency") || unsafe == "release-dependency" {
				dependency.Safety = "unsafe"
				if unsafe == "dependency-needs-review" {
					dependency.Safety = "needs-review"
				}
				if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", dependency, revision); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", root, 0); err != nil {
				t.Fatal(err)
			}
			publishAdmission(t, s, "dependency-release")
			clock := time.Now().UTC()
			scheduler := &schedules.Service{Store: s.Store, Auth: s.Auth, Runs: s, Now: func() time.Time { return clock }}
			definition := scheduleDefinition("protected")
			definition.Request.IDs = []string{root.ID}
			if err := scheduler.Save(t.Context(), "admin", "project", definition, 0); err != nil {
				t.Fatal(err)
			}
			if unsafe == "current-dependency" || unsafe == "release-dependency" {
				current, revision := currentTest(t, s, dependency.ID)
				current.Safety = "unsafe"
				if unsafe == "release-dependency" {
					current.Safety = "safe"
				}
				if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", current, revision); err != nil {
					t.Fatal(err)
				}
			}
			environment, revision := currentEnvironment(t, s)
			environment.Protected = true
			if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
				t.Fatal(err)
			}
			manual, err := s.Create(t.Context(), "admin", "project", definition.Request, "manual-protected", "manual")
			if (err == nil) != (unsafe == "none") {
				t.Fatal("manual dependency admission differs from safety", err)
			}
			pending, _ := readSchedule(t, scheduler, "protected")
			clock = *pending.NextAt
			tickSchedule(t, scheduler)
			fired, _ := readSchedule(t, scheduler, "protected")
			if unsafe != "none" {
				if fired.LastState != "blocked" || fired.LastRun != "" || !strings.Contains(fired.LastError, "safe") {
					t.Fatal("unsafe schedule lacks an explicit blocked firing", fired)
				}
				if count.Load() != 0 {
					t.Fatal("unsafe selection contacts its target")
				}
				return
			}
			scheduled, _, err := s.Get(t.Context(), "project", fired.LastRun)
			if err != nil || !reflect.DeepEqual(scheduled.Tests, manual.Tests) || !reflect.DeepEqual(scheduled.Environment, manual.Environment) || scheduled.Release.ID != manual.Release.ID {
				t.Fatal("manual and scheduled immutable selections differ", err)
			}
			start(t, s)
			for _, id := range []string{manual.ID, scheduled.ID} {
				completed := finished(t, s, id)
				if completed.State != "completed" || completed.Counts.Total != 2 || completed.Counts.Passed != 2 || len(completed.Results) != 2 || completed.EvidenceState != "complete" {
					t.Fatal("safe dependency closure loses normal result contracts", completed)
				}
			}
			if count.Load() != 4 {
				t.Fatal("dependency selection duplicates or omits execution", count.Load())
			}
		})
	}
}

func TestQueuedHTTPDispatchRechecksPolicyAndPreservesItsSnapshot(t *testing.T) {
	for _, mutation := range []string{"stage", "network", "application", "location", "disabled-test", "removed-test", "protected-unsafe"} {
		t.Run(mutation, func(t *testing.T) {
			s, p, count := setup(t, nil)
			if mutation == "protected-unsafe" {
				definition, revision := currentTest(t, s, "HEALTH-1")
				definition.Safety = "unsafe"
				if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, revision); err != nil {
					t.Fatal(err)
				}
				publishAdmission(t, s, "unsafe-unprotected")
			}
			run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "queued", "manual")
			if err != nil {
				t.Fatal(err)
			}
			environment, revision := currentEnvironment(t, s)
			switch mutation {
			case "stage":
				environment.Stage = "production"
			case "network":
				environment.AllowPrivate = true
			case "application":
				if err := p.SaveApplication(t.Context(), "admin", "project", projects.Application{ID: "other", Name: "Other"}, 0); err != nil {
					t.Fatal(err)
				}
				row, err := s.Store.Get(t.Context(), projects.Bucket("suites", "project"), "health")
				if err != nil {
					t.Fatal(err)
				}
				suite, err := storage.Decode[projects.Suite](row)
				if err != nil {
					t.Fatal(err)
				}
				suite.Applications = []string{"other"}
				if err := p.SaveSuite(t.Context(), "admin", "project", suite, row.Revision); err != nil {
					t.Fatal(err)
				}
			case "location":
				environment.Locations = []string{"remote"}
			case "protected-unsafe":
				environment.Protected = true
			case "disabled-test":
				if err := s.Catalog.ChangeState(t.Context(), "admin", "project", "health", "HEALTH-1", "disabled", 1); err != nil {
					t.Fatal(err)
				}
			case "removed-test":
				row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
				if err != nil || s.Store.Commit(t.Context(), storage.Change{Record: row, Delete: true}) != nil {
					t.Fatal("owned test removal fails", err)
				}
			}
			if mutation != "disabled-test" && mutation != "removed-test" && mutation != "application" {
				if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
					t.Fatal(err)
				}
			}
			start(t, s)
			final := finished(t, s, run.ID)
			if final.State != "blocked" || final.Error == "" || len(final.Results) != 0 || count.Load() != 0 {
				t.Fatal("changed policy dispatches a queued test", final.State, final.Error, count.Load())
			}
			if !reflect.DeepEqual(final.Tests, run.Tests) || !reflect.DeepEqual(final.Environment, run.Environment) || !reflect.DeepEqual(final.Release, run.Release) {
				t.Fatal("current policy rewrites the frozen run")
			}
		})
	}
}

func TestRunAdmissionSupportsMoreThanOneGuardBatch(t *testing.T) {
	s, _, _ := setup(t, nil)
	base, _ := currentTest(t, s, "HEALTH-1")
	for index := 0; index < 100; index++ {
		definition := base
		definition.ID = fmt.Sprintf("EXTRA-%03d", index)
		if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, 0); err != nil {
			t.Fatal(err)
		}
	}
	publishAdmission(t, s, "large-selection")
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "many-guards", "manual")
	if err != nil || len(run.Tests) != 101 {
		t.Fatal("atomic admission cannot cross its internal guard page", err)
	}
}
