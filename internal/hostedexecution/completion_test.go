package hostedexecution_test

import (
	"context"
	"testing"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type completionSource struct {
	hostedexecution.Provider
	f    *fixture
	mode string
}

func (p completionSource) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	s, err := p.Provider.Observe(ctx, ref)
	if err != nil {
		return s, err
	}
	if p.mode == "missing-report" && s.Phase == "running" {
		state, err := p.f.exchange.State(ctx, ref)
		if err == nil && !state.DispatchPending {
			p.f.provider.status.Phase = "succeeded"
			s.Phase = "succeeded"
		}
	}
	if s.Phase == "succeeded" {
		s.NativeCompletion = p.mode != "missing-completion"
		if p.mode != "conflicting-exit" {
			s.ExitCode = nil
		}
	}
	return s, nil
}

func TestHostedNativeCompletionStillRequiresAnAuthenticatedRunnerObservation(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, mode := range []string{"passed", "failed", "provider-failed", "missing-report", "missing-completion", "conflicting-exit"} {
			t.Run(runner+mode, func(t *testing.T) {
				f := fresh(t, runner)
				if mode == "failed" || mode == "provider-failed" {
					f.provider.mode = mode
				}
				if mode == "missing-report" {
					f.provider.mode = "running"
				}
				f.driver.Provider = completionSource{Provider: f.provider, f: f, mode: mode}
				report := f.driver.Execute(t.Context(), f.input)
				want := "unknown"
				if mode == "passed" || mode == "failed" {
					want = mode
				}
				if report.Result.State != want || f.provider.ensureCount != 1 {
					t.Fatal("provider completion replaced the observed test outcome", report.Result.State, report.RecoveryError)
				}
				if want == "unknown" && (!report.Result.DurationUnavailable || report.Result.DurationMs != 0) {
					t.Fatal("provider completion invented a test duration")
				}
				if want != "unknown" && (report.RecoveryError != nil || report.Result.DurationMs != 1 || len(report.Attachments) != 1 || f.provider.removeCount != 1) {
					t.Fatal("native completion lost measured timing, evidence, or cleanup", report.RecoveryError)
				}
			})
		}
	}
}
