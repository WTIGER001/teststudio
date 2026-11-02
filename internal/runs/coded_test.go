package runs_test

import (
	"context"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"strings"
	"testing"
	"time"
)

type codedExecutor struct{ report execution.CodedReport }

func (e *codedExecutor) Profile() string                                             { return "playwright-local-isolated-v1" }
func (e *codedExecutor) Admit(context.Context, catalog.Test, catalog.Artifact) error { return nil }
func (e *codedExecutor) Execute(ctx context.Context, input execution.CodedInput) execution.CodedReport {
	report := e.report
	now := time.Now().UTC()
	report.Result.TestID = input.Test.ID
	report.Result.StartedAt = now.Add(-time.Second)
	report.Result.FinishedAt = now
	report.Result.DurationMs = 1000
	return report
}

func TestIsolatedPlaywrightUsesNormalAttemptsTimingEvidenceAndTargetlessSelection(t *testing.T) {
	for _, state := range []string{"passed", "failed", "unknown"} {
		t.Run(state, func(t *testing.T) {
			s, p, _ := setup(t, nil)
			suiteRow, err := s.Store.Get(t.Context(), projects.Bucket("suites", "project"), "health")
			if err != nil {
				t.Fatal(err)
			}
			suite, err := storage.Decode[projects.Suite](suiteRow)
			if err != nil {
				t.Fatal(err)
			}
			suite.RequiresTarget = false
			if err := p.SaveSuite(t.Context(), "admin", "project", suite, suiteRow.Revision); err != nil {
				t.Fatal(err)
			}
			row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
			if err != nil {
				t.Fatal(err)
			}
			test, err := storage.Decode[catalog.Test](row)
			if err != nil {
				t.Fatal(err)
			}
			test.Runner = "playwright"
			test.Type = "browser"
			test.Source = "selftest"
			test.Selector = "@HEALTH-1"
			test.Check = nil
			if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, row.Revision); err != nil {
				t.Fatal(err)
			}
			artifact := catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64), SourceRevision: "selftest", Discovered: map[string]string{test.ID: test.Selector}}
			d, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", artifact)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", d.ID, "coded", 1, true); err != nil {
				t.Fatal(err)
			}
			req := runs.Request{Suite: "health"}
			if _, err := s.Create(t.Context(), "admin", "project", req, "unconfigured", "manual"); err == nil {
				t.Fatal("unconfigured Playwright admitted")
			}
			s.Playwright = &codedExecutor{report: execution.CodedReport{Result: execution.Result{State: state, Steps: []execution.Step{{Title: "Open page", Category: "test.step", DurationMs: 1000}}}, Attachments: []execution.Attachment{{Kind: "log", Stream: "playwright.stdout", Filename: "stdout.log", ContentType: "text/plain", Data: []byte("test log")}, {Kind: "screenshot", Filename: "page.png", ContentType: "image/png", Data: []byte("image")}, {Kind: "trace", Filename: "trace.zip", ContentType: "application/zip", Data: []byte("trace")}}}}
			targeted := req
			targeted.Environment = "local"
			if _, err := s.Create(t.Context(), "admin", "project", targeted, "wrong-profile", "manual"); err == nil {
				t.Fatal("isolated profile ignored an external target")
			}
			v, err := s.Create(t.Context(), "admin", "project", req, "coded", "manual")
			if err != nil {
				t.Fatal(err)
			}
			if v.Timing.Tests[test.ID].Context.Profile != "playwright-local-isolated-v1" {
				t.Fatal("coded timing mixed with native profile")
			}
			start(t, s)
			v = finished(t, s, v.ID)
			if len(v.Results) != 1 || v.Results[0].State != state || len(v.Results[0].Steps) != 1 {
				t.Fatalf("coded reduction: %+v", v)
			}
			if state == "unknown" && v.Outcome != "unknown" {
				t.Fatal("lost worker passed")
			}
			artifacts, err := s.Store.All(t.Context(), "artifacts/"+v.ID, "project")
			if err != nil {
				t.Fatal(err)
			}
			media := 0
			codedLog := false
			for _, row := range artifacts {
				a, err := storage.Decode[map[string]any](row)
				if err != nil {
					t.Fatal(err)
				}
				if a["kind"] == "log" && a["stream"] == "playwright.stdout" && a["sequence"] == float64(1) {
					codedLog = true
				}
				if a["kind"] == "trace" || a["kind"] == "screenshot" {
					media++
				}
			}
			if !codedLog {
				t.Fatal("coded log stream or ordering metadata is missing")
			}
			expected := 2
			if state == "passed" {
				expected = 0
			}
			if media != expected {
				t.Fatalf("capture policy: got %d media, want %d", media, expected)
			}
		})
	}
}
