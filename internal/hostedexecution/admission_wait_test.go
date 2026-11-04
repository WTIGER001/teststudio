package hostedexecution_test

import (
	"slices"
	"testing"
)

func TestHostedExecutionWaitsForProviderAdmissionBeforeReleasingWork(t *testing.T) {
	f := fresh(t, "go")
	f.provider.admissionPolls = 3
	report := f.driver.Execute(t.Context(), f.input)
	if report.Result.State != "passed" || report.RecoveryError != nil || f.provider.admissionPolls != 0 || f.provider.ensureCount != 1 || f.provider.removeCount != 1 {
		t.Fatal("pending provider admission did not preserve normal execution", report.Result.State, report.RecoveryError)
	}
	if !slices.Contains(f.events, "admission-pending") {
		t.Fatal("provider admission was not observed")
	}
}

func TestHostedExecutionStopsAfterAdmissionIsLostFollowingActivation(t *testing.T) {
	f := fresh(t, "playwright")
	f.provider.mode, f.provider.loseAdmission = "running", true
	report := f.driver.Execute(t.Context(), f.input)
	if report.Result.State != "unknown" || report.RecoveryError != nil || !slices.Contains(f.events, "stop") || f.provider.removeCount != 1 {
		t.Fatal("lost admission did not revoke and stop the owned worker", report.Result.State, report.RecoveryError, f.events)
	}
}
