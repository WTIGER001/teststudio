package hostedexecution_test

import (
	"context"
	"testing"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type boundaryFailureProvider struct {
	hostedexecution.Provider
	f                *fixture
	beforeActivation bool
}

func (p boundaryFailureProvider) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	status, err := p.Provider.Observe(ctx, ref)
	if err != nil {
		return status, err
	}
	state, err := p.f.exchange.State(ctx, ref)
	if err != nil {
		return status, err
	}
	if p.beforeActivation && state.DispatchPending {
		status.InfrastructureFailure = true
	} else if !p.beforeActivation && status.Phase == "succeeded" {
		status.InfrastructureFailure = true
		status.Phase, status.ExitCode, status.NativeCompletion = "failed", nil, false
	}
	return status, nil
}

func TestHostedBoundaryFailureClosesActivationAndCannotVerifyAReportedPass(t *testing.T) {
	for _, before := range []bool{true, false} {
		t.Run(map[bool]string{true: "before activation", false: "after report"}[before], func(t *testing.T) {
			f := fresh(t, "playwright")
			f.driver.Provider = boundaryFailureProvider{Provider: f.provider, f: f, beforeActivation: before}
			report := f.driver.Execute(t.Context(), f.input)
			if report.Result.State != "unknown" || !report.Result.DurationUnavailable || report.Result.DurationMs != 0 || report.RecoveryError != nil {
				t.Fatal("infrastructure boundary failure became a verified result or inferred duration", report.Result, report.RecoveryError)
			}
			if f.provider.ensureCount != 1 || f.provider.removeCount != 1 {
				t.Fatal("boundary failure repeated execution or lost cleanup")
			}
			if !before && len(report.Attachments) != 1 {
				t.Fatal("boundary failure discarded available private evidence")
			}
		})
	}
}
