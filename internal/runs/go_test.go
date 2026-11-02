package runs_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type goExecutor struct {
	state, kind string
	inputs      chan execution.CodedInput
}

func (e *goExecutor) Profile() string { return "go-local-isolated-v1" }
func (e *goExecutor) TimingProfile(context.Context, catalog.Artifact) (string, error) {
	return e.Profile() + ":go1.26-linux-arm64-cgo0", nil
}
func (e *goExecutor) Admit(context.Context, catalog.Test, catalog.Artifact) error { return nil }
func (e *goExecutor) Execute(_ context.Context, input execution.CodedInput) execution.CodedReport {
	e.inputs <- input
	now := time.Now().UTC()
	if input.OnStart != nil {
		_ = input.OnStart(now.Add(-time.Second))
	}
	r := execution.Result{TestID: input.Test.ID, State: e.state, FailureKind: e.kind, StartedAt: now.Add(-time.Second), FinishedAt: now, DurationMs: 1000, Go: &execution.GoObservation{Package: "./unit", Test: "TestFresh", Toolchain: "go1.26"}}
	if e.state == "unknown" {
		r.DurationUnavailable = true
		r.DurationMs = 0
		r.Error = "build preparation failed"
	}
	return execution.CodedReport{Result: r, Attachments: []execution.Attachment{{Kind: "log", Filename: "go-events.jsonl", Stream: "go.events", ContentType: "text/plain", Data: []byte("fresh Go evidence")}}}
}

func configureGo(t *testing.T, s *runs.Service, p *projects.Service) {
	t.Helper()
	ctx := t.Context()
	row, err := s.Store.Get(ctx, projects.Bucket("suites", "project"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, err := storage.Decode[projects.Suite](row)
	if err != nil {
		t.Fatal(err)
	}
	suite.RequiresTarget = false
	if err = p.SaveSuite(ctx, "admin", "project", suite, row.Revision); err != nil {
		t.Fatal(err)
	}
	row, err = s.Store.Get(ctx, catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	definition, err := storage.Decode[catalog.Test](row)
	if err != nil {
		t.Fatal(err)
	}
	definition.Runner, definition.Type, definition.Source, definition.Selector, definition.Check = "go", "unit", "fresh-go-suite", "./unit::TestFresh", nil
	if err = s.Catalog.SaveTest(ctx, "admin", "project", "health", definition, row.Revision); err != nil {
		t.Fatal(err)
	}
	draft, err := s.Catalog.CreateDraft(ctx, "admin", "project", "health", catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64), Discovered: map[string]string{definition.ID: definition.Selector}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Catalog.Publish(ctx, "admin", "project", "health", draft.ID, "go-release", 1, true); err != nil {
		t.Fatal(err)
	}
}

func TestGoDispatchIsTargetlessAndUsesNormalTimingAndTimeoutFailureReruns(t *testing.T) {
	s, p, _ := setup(t, nil)
	configureGo(t, s, p)
	req := runs.Request{Suite: "health", Retries: 1}
	if _, err := s.Create(t.Context(), "admin", "project", req, "disabled-go", "manual"); err == nil {
		t.Fatal("disabled Go profile admitted")
	}
	e := &goExecutor{state: "timed-out", kind: "timeout", inputs: make(chan execution.CodedInput, 4)}
	s.Go = e
	targeted := req
	targeted.Environment = "local"
	if _, err := s.Create(t.Context(), "admin", "project", targeted, "targeted-go", "manual"); err == nil {
		t.Fatal("isolated Go profile admitted an external target")
	}
	parent, err := s.Create(t.Context(), "admin", "project", req, "go-timeout", "manual")
	if err != nil {
		t.Fatal(err)
	}
	if parent.Timing.Tests["HEALTH-1"].Context.Profile != "go-local-isolated-v1:go1.26-linux-arm64-cgo0" {
		t.Fatal("Go runtime timing profile is missing")
	}
	start(t, s)
	parent = finished(t, s, parent.ID)
	if parent.State != "completed" || parent.Outcome != "timed-out" || parent.Counts.TimedOut != 1 || parent.Counts.Failed != 0 || len(parent.Results) != 1 || parent.Results[0].Attempt != 2 || parent.Results[0].Go == nil || !parent.Results[0].Timing.Censored {
		t.Fatalf("Go timeout: %+v", parent)
	}
	first := <-e.inputs
	if first.TimeoutMs != parent.Timing.Tests["HEALTH-1"].EffectiveTimeoutMs || first.TimingMode != parent.Timing.Tests["HEALTH-1"].Policy.Mode {
		t.Fatal("frozen timing contract did not reach the Go worker")
	}
	<-e.inputs
	child, err := s.RerunFailures(t.Context(), "admin", "project", parent.ID, "go-timeout-rerun")
	if err != nil || len(child.Tests) != 1 || child.Release.ID != parent.Release.ID {
		t.Fatalf("Go timeout rerun: %+v %v", child, err)
	}
	finished(t, s, child.ID)
}

func TestGoPreparationFailuresBlockWithoutAssertionRetriesOrInventedDurations(t *testing.T) {
	for _, kind := range []string{"build", "discovery", "configuration", "worker-report"} {
		t.Run(kind, func(t *testing.T) {
			s, p, _ := setup(t, nil)
			configureGo(t, s, p)
			e := &goExecutor{state: "unknown", kind: kind, inputs: make(chan execution.CodedInput, 4)}
			s.Go = e
			r, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Retries: 2}, "go-preparation", "manual")
			if err != nil {
				t.Fatal(err)
			}
			start(t, s)
			r = finished(t, s, r.ID)
			attempts, err := s.Store.All(t.Context(), "attempts/"+r.ID, "project")
			if err != nil {
				t.Fatal(err)
			}
			if r.State != "blocked" || r.Outcome != "unknown" || r.Counts.Failed != 0 || r.Counts.Completed != 0 || len(attempts) != 1 || len(e.inputs) != 1 {
				t.Fatalf("preparation became a test failure: %+v attempts %d", r, len(attempts))
			}
			attempt, err := storage.Decode[execution.Result](attempts[0])
			if err != nil || attempt.FailureKind != kind || !attempt.DurationUnavailable || attempt.DurationMs != 0 || !attempt.Timing.Censored {
				t.Fatalf("preparation duration: %+v %v", attempt, err)
			}
		})
	}
}
