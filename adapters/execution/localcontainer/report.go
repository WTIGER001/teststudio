package localcontainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/gosuite"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func regularFile(root *os.Root, path string, limit int64) ([]byte, error) {
	if path == "" || filepath.Base(path) != path || strings.ContainsAny(path, "\\\x00") {
		return nil, errors.New("worker artifact path must be a plain filename")
	}
	info, err := root.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("worker artifact is not a regular file within its size limit")
	}
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("worker artifact exceeds its size limit")
	}
	return data, err
}

func readReport(directory string, input execution.CodedInput, exitError error) (execution.CodedReport, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return execution.CodedReport{}, err
	}
	defer root.Close()
	data, err := regularFile(root, "report.json", 2<<20)
	if err != nil {
		return execution.CodedReport{}, fmt.Errorf("Coded worker result is unavailable: %w", err)
	}
	var report execution.CodedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return report, errors.New("Coded worker result is invalid")
	}
	report.Result = execution.NormalizeTimeout(report.Result)
	result := report.Result
	runner := "playwright"
	if input.Test.Runner == "go" {
		runner = "go"
	}
	if report.Schema != "teststudio."+runner+"/v1" || report.TestID != input.Test.ID || report.Selector != input.Test.Selector || result.TestID != input.Test.ID {
		return report, errors.New("Coded worker result does not match the selected test")
	}
	goProblem := runner == "go" && result.State == "unknown" && result.FailureKind != "" && result.DurationUnavailable
	if result.State != "passed" && result.State != "failed" && result.State != "timed-out" && result.State != "skipped" && !goProblem {
		return report, errors.New("Coded worker result has no supported terminal state")
	}
	if result.State == "timed-out" && result.FailureKind != "timeout" {
		return report, errors.New("Coded worker timeout requires a matching timeout reason")
	}
	if math.IsNaN(result.DurationMs) || math.IsInf(result.DurationMs, 0) || result.DurationMs < 0 || result.DurationMs > 3600000 || result.StartedAt.IsZero() || result.FinishedAt.Before(result.StartedAt) || result.FinishedAt.After(time.Now().Add(time.Minute)) {
		return report, errors.New("Coded worker timing is invalid")
	}
	if runner == "go" {
		selected, err := gosuite.Parse(input.Test.Selector)
		if err != nil || (result.DurationUnavailable && result.DurationMs != 0) || math.IsNaN(result.WorkerDurationMs) || math.IsInf(result.WorkerDurationMs, 0) || result.WorkerDurationMs < 0 {
			return report, errors.New("Go worker timing does not match a valid selection")
		}
		if result.Go == nil && !goProblem {
			return report, errors.New("Go worker observation is unavailable")
		}
		if g := result.Go; g != nil {
			if (!goProblem && (g.Package != selected.Package || g.Test != selected.Test || g.Toolchain == "" || g.OS == "" || g.Architecture == "")) || len(g.Package) > 1024 || len(g.Test) > 1024 || len(g.Toolchain) > 128 || len(g.OS) > 32 || len(g.Architecture) > 32 || math.IsNaN(g.BuildDurationMs) || math.IsInf(g.BuildDurationMs, 0) || g.BuildDurationMs < 0 || g.BuildDurationMs > 3600000 {
				return report, errors.New("Go worker observation is invalid")
			}
		}
	}
	if exitError != nil && !goProblem && ((result.State != "failed" && result.State != "timed-out") || result.FailureKind == "") {
		return report, errors.New("Coded worker process failed without a matching failed result")
	}
	if len(result.Steps) > 500 || len(report.Attachments) > 32 || len(report.EvidenceErrors) > 20 || len(result.Error) > 16384 {
		return report, errors.New("Coded worker result exceeds its contract limits")
	}
	for _, step := range result.Steps {
		if len(step.Title) > 2048 || len(step.Error) > 16384 || step.DurationMs < 0 || math.IsNaN(step.DurationMs) || math.IsInf(step.DurationMs, 0) {
			return report, errors.New("Coded worker step is invalid")
		}
	}
	var bytes int64
	accepted := []execution.Attachment{}
	for _, file := range report.Attachments {
		if file.Kind != "screenshot" && file.Kind != "trace" && file.Kind != "video" && file.Kind != "attachment" && file.Kind != "log" {
			report.EvidenceErrors = append(report.EvidenceErrors, "Unsupported coded worker attachment kind.")
			continue
		}
		limit := min(input.Capture.MaxFileBytes, input.Capture.MaxBytes-bytes, 64<<20)
		body, err := regularFile(root, file.Path, limit)
		if err != nil {
			report.EvidenceErrors = append(report.EvidenceErrors, file.Filename+": "+err.Error())
			continue
		}
		bytes += int64(len(body))
		file.Data = body
		accepted = append(accepted, file)
	}
	report.Attachments = accepted
	if len(report.EvidenceErrors) > 20 {
		report.EvidenceErrors = report.EvidenceErrors[:20]
	}
	return report, nil
}
