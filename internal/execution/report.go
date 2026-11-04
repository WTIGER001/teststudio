package execution

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/gosuite"
	"math"
	"time"
)

func ValidateCodedReport(input CodedInput, report CodedReport, exitError error) error {
	result := report.Result
	runner := "playwright"
	if input.Test.Runner == "go" {
		runner = "go"
	}
	if report.Schema != "teststudio."+runner+"/v1" || report.TestID != input.Test.ID || report.Selector != input.Test.Selector || result.TestID != input.Test.ID {
		return errors.New("Coded worker result does not match the selected test")
	}
	goProblem := runner == "go" && result.State == "unknown" && result.FailureKind != "" && result.DurationUnavailable
	if result.State != "passed" && result.State != "failed" && result.State != "timed-out" && result.State != "skipped" && !goProblem {
		return errors.New("Coded worker result has no supported terminal state")
	}
	if result.State == "timed-out" && result.FailureKind != "timeout" {
		return errors.New("Coded worker timeout requires a matching timeout reason")
	}
	if math.IsNaN(result.DurationMs) || math.IsInf(result.DurationMs, 0) || result.DurationMs < 0 || result.DurationMs > 3600000 || result.StartedAt.IsZero() || result.FinishedAt.Before(result.StartedAt) || result.FinishedAt.After(time.Now().Add(time.Minute)) {
		return errors.New("Coded worker timing is invalid")
	}
	if runner == "go" {
		selected, err := gosuite.Parse(input.Test.Selector)
		if err != nil || (result.DurationUnavailable && result.DurationMs != 0) || math.IsNaN(result.WorkerDurationMs) || math.IsInf(result.WorkerDurationMs, 0) || result.WorkerDurationMs < 0 {
			return errors.New("Go worker timing does not match a valid selection")
		}
		if result.Go == nil && !goProblem {
			return errors.New("Go worker observation is unavailable")
		}
		if g := result.Go; g != nil {
			if (!goProblem && (g.Package != selected.Package || g.Test != selected.Test || g.Toolchain == "" || g.OS == "" || g.Architecture == "")) || len(g.Package) > 1024 || len(g.Test) > 1024 || len(g.Toolchain) > 128 || len(g.OS) > 32 || len(g.Architecture) > 32 || math.IsNaN(g.BuildDurationMs) || math.IsInf(g.BuildDurationMs, 0) || g.BuildDurationMs < 0 || g.BuildDurationMs > 3600000 {
				return errors.New("Go worker observation is invalid")
			}
		}
	}
	if exitError != nil && !goProblem && ((result.State != "failed" && result.State != "timed-out") || result.FailureKind == "") {
		return errors.New("Coded worker process failed without a matching failed result")
	}
	if len(result.Steps) > 500 || len(report.Attachments) > 32 || len(report.EvidenceErrors) > 20 || len(result.Error) > 16384 {
		return errors.New("Coded worker result exceeds its contract limits")
	}
	for _, step := range result.Steps {
		if len(step.Title) > 2048 || len(step.Error) > 16384 || step.DurationMs < 0 || math.IsNaN(step.DurationMs) || math.IsInf(step.DurationMs, 0) {
			return errors.New("Coded worker step is invalid")
		}
	}
	return nil
}
