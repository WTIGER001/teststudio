package localcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/gosuite"
)

func goContainerInput(t *testing.T, id string, timeout int) (*Executor, execution.CodedInput) {
	t.Helper()
	image := os.Getenv("TESTSTUDIO_GO_IMAGE")
	if image == "" {
		t.Skip("Set TESTSTUDIO_GO_IMAGE for isolated Go container checks")
	}
	data, err := os.ReadFile("../../../tests/selftest/go/teststudio-go.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest gosuite.Manifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, entry := range manifest.Tests {
		if entry.ID == id {
			return &Executor{Runner: "go", Images: []string{image}}, execution.CodedInput{RunID: domain.NewID(), Test: catalog.Test{ID: id, Runner: "go", Selector: entry.Selector()}, Artifact: catalog.Artifact{Image: image, Discovered: map[string]string{id: entry.Selector()}}, TimeoutMs: timeout, Capture: capture.DefaultPolicy()}
		}
	}
	t.Fatal("unknown Go fixture")
	return nil, execution.CodedInput{}
}
func assertGoRemoved(t *testing.T, run string) {
	t.Helper()
	assertRemoved(t, run)
	data, err := exec.Command("docker", "volume", "ls", "--filter", "label=teststudio.run="+run, "--format", "{{.Name}}").Output()
	if err != nil || strings.TrimSpace(string(data)) != "" {
		t.Fatalf("Go output volumes remain: %s %v", data, err)
	}
}

func TestGoContainerOutcomesHaveNoBrowserOrArtificialTarget(t *testing.T) {
	for _, tc := range []struct {
		id, state, kind string
		steps           int
	}{
		{"GO-SUM-001", "passed", "", 1}, {"GO-ROWS-001", "passed", "", 3}, {"GO-CHILD-001", "passed", "", 1},
		{"GO-FAIL-001", "failed", "assertion", 1}, {"GO-PANIC-001", "failed", "panic", 0},
		{"GO-TIMEOUT-001", "timed-out", "timeout", 0}, {"GO-MISSING-001", "unknown", "discovery", 0},
		{"GO-BUILD-001", "unknown", "build", 0}, {"GO-SKIP-001", "skipped", "", 1},
	} {
		t.Run(tc.id, func(t *testing.T) {
			driver, input := goContainerInput(t, tc.id, 200)
			started := false
			input.OnStart = func(time.Time) error { started = true; return nil }
			report := driver.Execute(t.Context(), input)
			assertGoRemoved(t, input.RunID)
			if report.Result.State != tc.state || report.Result.FailureKind != tc.kind || len(report.EvidenceErrors) != 0 || report.Result.Go == nil || len(report.Attachments) == 0 {
				t.Fatalf("Go container report: %+v evidence %v", report.Result, report.EvidenceErrors)
			}
			if tc.steps > 0 && len(report.Result.Steps) != tc.steps {
				t.Fatalf("logical subtest steps: %+v", report.Result.Steps)
			}
			if tc.state == "unknown" && (!report.Result.DurationUnavailable || report.Result.DurationMs != 0 || started) {
				t.Fatal("preparation invented a test execution")
			}
			if tc.state != "unknown" && !started {
				t.Fatal("real test start was not reported")
			}
			if tc.state == "passed" {
				if profile, err := driver.TimingProfile(t.Context(), input.Artifact); err != nil || !strings.HasPrefix(profile, "go-local-isolated-v1:go1.26.8-linux-") {
					t.Fatalf("Go runtime profile: %s %v", profile, err)
				}
			}
		})
	}
}

func TestGoContainerCancellationAndRetainedCheckpointRecovery(t *testing.T) {
	driver, input := goContainerInput(t, "GO-TIMEOUT-001", 60000)
	ref := driver.Reference(execution.Reference{ID: domain.NewID(), Installation: "go-install-" + domain.NewID(), Project: "project", Run: input.RunID, Test: input.Test.ID, Attempt: 1, Worker: "worker"})
	input.Reference = &ref
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := driver.Cleanup(ctx, ref); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	input.OnStart = func(time.Time) error { cancel(); return nil }
	input.BeforeCleanup = func(execution.CodedReport) error { return errors.New("simulated checkpoint loss") }
	report := driver.Execute(ctx, input)
	if report.Result.State != "unknown" || report.Result.FailureKind != "cancelled" {
		t.Fatalf("cancelled Go execution: %+v", report.Result)
	}
	refs, err := driver.Inventory(t.Context(), ref.Installation)
	if err != nil || len(refs) != 1 {
		t.Fatalf("Go retained inventory: %+v %v", refs, err)
	}
	if err := (&Executor{}).Cleanup(t.Context(), ref); err == nil {
		t.Fatal("Playwright driver cleaned a Go reference")
	}
	recovered := driver.Recover(t.Context(), ref, input)
	if recovered.RecoveryError != nil || recovered.Result.State != "unknown" || recovered.Result.FailureKind != "cancelled" || len(recovered.Attachments) == 0 {
		t.Fatalf("Go recovery lost evidence: %+v", recovered)
	}
	if err := driver.Cleanup(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	assertGoRemoved(t, input.RunID)
}
