package localcontainer

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func contractFixture(t *testing.T) (string, execution.CodedInput, execution.CodedReport) {
	t.Helper()
	directory := t.TempDir()
	now := time.Now().UTC()
	input := execution.CodedInput{Test: catalog.Test{ID: "UI-1", Selector: "@UI-1"}, Capture: capture.DefaultPolicy()}
	report := execution.CodedReport{Schema: "teststudio.playwright/v1", TestID: "UI-1", Selector: "@UI-1", Result: execution.Result{TestID: "UI-1", State: "passed", StartedAt: now.Add(-time.Second), FinishedAt: now, DurationMs: 1000}}
	return directory, input, report
}
func writeReport(t *testing.T, directory string, report execution.CodedReport) {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "report.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerContractCannotPassMissingWrongOrCrashingTests(t *testing.T) {
	directory, input, report := contractFixture(t)
	if _, err := readReport(directory, input, nil); err == nil {
		t.Fatal("missing report passed")
	}
	writeReport(t, directory, report)
	if _, err := readReport(directory, input, errors.New("worker failed")); err == nil {
		t.Fatal("worker exit failure became pass")
	}
	report.TestID = "different"
	writeReport(t, directory, report)
	if _, err := readReport(directory, input, nil); err == nil {
		t.Fatal("wrong test became pass")
	}
	report.TestID = input.Test.ID
	report.Result.State = "failed"
	report.Result.FailureKind = "assertion"
	writeReport(t, directory, report)
	parsed, err := readReport(directory, input, errors.New("assertion failed"))
	if err != nil || parsed.Result.State != "failed" {
		t.Fatalf("assertion failure lost: %+v %v", parsed, err)
	}
}

func TestUndispatchedWorkerDoesNotInventExecutionDuration(t *testing.T) {
	_, input, _ := contractFixture(t)
	report := (&Executor{}).Execute(t.Context(), input)
	if report.Result.State != "unknown" || !report.Result.DurationUnavailable || report.Result.DurationMs != 0 || report.Result.WorkerDurationMs < 0 {
		t.Fatalf("unmeasured execution duration: %+v", report.Result)
	}
}

func TestGoPreparationReportsStayUnknownAndCannotHideAProcessFailure(t *testing.T) {
	directory, input, report := contractFixture(t)
	input.Test.Runner = "go"
	input.Test.Selector = "./unit::TestFresh"
	report.Schema = "teststudio.go/v1"
	report.Selector = input.Test.Selector
	report.Result.State = "unknown"
	report.Result.FailureKind = "build"
	report.Result.DurationUnavailable = true
	report.Result.DurationMs = 0
	writeReport(t, directory, report)
	parsed, err := readReport(directory, input, errors.New("build failed"))
	if err != nil || parsed.Result.State != "unknown" || !parsed.Result.DurationUnavailable {
		t.Fatalf("Go build problem lost: %+v %v", parsed, err)
	}
	report.Result.State = "passed"
	report.Result.FailureKind = ""
	report.Result.DurationUnavailable = false
	writeReport(t, directory, report)
	if _, err := readReport(directory, input, errors.New("worker failed")); err == nil {
		t.Fatal("Go worker exit became a pass")
	}
	if _, err := readReport(directory, input, nil); err == nil {
		t.Fatal("Go result passed without runtime metadata")
	}
}

func TestWorkerContractReadsNewAndLegacyTimeoutsWithoutCallingThemFailures(t *testing.T) {
	for _, state := range []string{"failed", "timed-out"} {
		directory, input, report := contractFixture(t)
		report.Result.State = state
		report.Result.FailureKind = "timeout"
		writeReport(t, directory, report)
		parsed, err := readReport(directory, input, errors.New("worker timed out"))
		if err != nil || parsed.Result.State != "timed-out" || parsed.Result.FailureKind != "timeout" {
			t.Fatalf("timeout contract: %+v %v", parsed, err)
		}
		report.Result.State = "timed-out"
		report.Result.FailureKind = ""
		writeReport(t, directory, report)
		if _, err := readReport(directory, input, nil); err == nil {
			t.Fatal("timeout without a timeout reason was accepted")
		}
	}
}

func TestEvidenceImportRejectsEscapeLinksAndOversizeWithoutChangingResult(t *testing.T) {
	directory, input, report := contractFixture(t)
	input.Capture.MaxFileBytes = 8
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "large"), []byte("large body"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "image"), []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	report.Attachments = []execution.Attachment{{Path: "../outside", Filename: "escape", Kind: "screenshot"}, {Path: "linked", Filename: "linked", Kind: "trace"}, {Path: "large", Filename: "large", Kind: "attachment"}, {Path: "image", Filename: "image.png", Kind: "screenshot", ContentType: "image/png"}}
	writeReport(t, directory, report)
	parsed, err := readReport(directory, input, nil)
	if err != nil || parsed.Result.State != "passed" || len(parsed.EvidenceErrors) != 3 || len(parsed.Attachments) != 1 || string(parsed.Attachments[0].Data) != "image" {
		t.Fatalf("evidence confinement: %+v %v", parsed, err)
	}
}

func TestBoundedWorkerOutputContinuesDraining(t *testing.T) {
	var b boundedOutput
	data := make([]byte, 3<<20)
	n, err := io.Copy(&b, io.LimitReader(bytes.NewReader(data), int64(len(data))))
	if err != nil || n != int64(len(data)) || b.Len() != 2<<20 || !b.Truncated {
		t.Fatal("worker output limit does not drain")
	}
}

func TestProgressLinesSurviveChunkingAndIgnoreOversizedLines(t *testing.T) {
	lines := []string{}
	b := boundedOutput{OnLine: func(line string) { lines = append(lines, line) }}
	_, _ = b.Write([]byte("start"))
	_, _ = b.Write([]byte("ed\n"))
	_, _ = b.Write(make([]byte, 70000))
	_, _ = b.Write([]byte("\nfinished\n"))
	if len(lines) != 2 || lines[0] != "started" || lines[1] != "finished" || len(b.line) > 65536 {
		t.Fatalf("progress framing: %v", lines)
	}
}
