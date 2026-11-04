package hostedexecution_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestHostedLifecycleCheckpointsPrivateObservationsBeforeDeletingResources(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, mode := range []string{"passed", "failed", "provider-failed"} {
			t.Run(runner+"/"+mode, func(t *testing.T) {
				f := fresh(t, runner)
				f.provider.mode = mode
				report := f.driver.Execute(t.Context(), f.input)
				want := "passed"
				if mode == "failed" {
					want = "failed"
				}
				if mode == "provider-failed" {
					want = "unknown"
				}
				if report.RecoveryError != nil || report.Result.State != want || f.provider.ensureCount != 1 || f.provider.removeCount != 1 || f.cleanupError != nil {
					t.Fatalf("unexpected outcome: %#v, %v", report, f.events)
				}
				if len(report.Attachments) != 1 || string(report.Attachments[0].Data) != "private fixture evidence" {
					t.Fatal("lost private evidence")
				}
				if mode == "provider-failed" && (!report.Result.DurationUnavailable || report.Result.DurationMs != 0) {
					t.Fatal("provider failure invented a test duration")
				}
				if mode != "provider-failed" && report.Result.DurationMs != 1 {
					t.Fatal("provider duration replaced the test duration")
				}
				if !slices.Contains(f.events, "started") {
					t.Fatal("the acknowledged test start was not forwarded")
				}
				if slices.Index(f.events, "checkpoint") >= slices.Index(f.events, "remove") || slices.Index(f.events, "remove") >= slices.Index(f.events, "finished") {
					t.Fatal(f.events)
				}
				record, _, err := (workerstate.Ledger{Store: f.store}).Get(t.Context(), f.input.Reference.Run, f.input.Reference.ID)
				if err != nil || record.State != "cleaned" {
					t.Fatal(record, err)
				}
				if _, err := f.exchange.State(t.Context(), *f.input.Reference); !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("private staging was not released", err)
				}
			})
		}
	}
}

type readOnceFails struct {
	artifacts.Store
	fail bool
}

func (s *readOnceFails) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.fail {
		s.fail = false
		return nil, errors.New("temporary private storage outage")
	}
	return s.Store.Open(ctx, key)
}

func TestHostedEvidenceOutageRecoversWithoutRerunningTheTest(t *testing.T) {
	f := fresh(t, "playwright")
	f.exchange.Blobs = &readOnceFails{Store: f.exchange.Blobs, fail: true}
	report := f.driver.Execute(t.Context(), f.input)
	if report.Result.State != "passed" || !report.EvidencePending || f.provider.removeCount != 0 || len(report.EvidenceErrors) == 0 {
		t.Fatal("pending evidence was discarded", report)
	}
	if err := f.driver.Cleanup(t.Context(), *f.input.Reference); err == nil || f.provider.removeCount != 0 {
		t.Fatal("pending evidence allowed deletion")
	}
	recovered := f.driver.Recover(t.Context(), *f.input.Reference, f.input)
	if recovered.RecoveryError != nil || recovered.EvidencePending || len(recovered.Attachments) != 1 || recovered.Result.State != "passed" || f.provider.ensureCount != 1 {
		t.Fatal("evidence did not recover", recovered)
	}
	if err := f.input.BeforeCleanup(recovered); err != nil {
		t.Fatal(err)
	}
	if err := f.driver.Cleanup(t.Context(), *f.input.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestHostedUncertainStopRetainsItsDurableReference(t *testing.T) {
	f := fresh(t, "playwright")
	f.provider.mode = "running"
	f.provider.stopError = true
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f.provider.startCallback = cancel
	report := f.driver.Execute(ctx, f.input)
	if report.RecoveryError == nil || f.provider.removeCount != 0 || slices.Contains(f.events, "checkpoint") {
		t.Fatal("uncertain running job was discarded", report)
	}
	f.provider.stopError = false
	recovered := f.driver.Recover(t.Context(), *f.input.Reference, f.input)
	if recovered.RecoveryError != nil || recovered.Result.State != "unknown" || f.provider.status.Phase != "stopped" || f.provider.ensureCount != 1 {
		t.Fatal("owned running job did not stop", recovered)
	}
	if err := f.input.BeforeCleanup(recovered); err != nil {
		t.Fatal(err)
	}
	if err := f.driver.Cleanup(t.Context(), *f.input.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestHostedLostCreateAcknowledgementRecoversWithoutDispatchingAgain(t *testing.T) {
	f := fresh(t, "playwright")
	f.provider.mode = "lost-ack"
	report := f.driver.Execute(t.Context(), f.input)
	if report.RecoveryError == nil || f.provider.removeCount != 0 || slices.Contains(f.events, "checkpoint") {
		t.Fatal("ambiguous dispatch was discarded", f.events)
	}
	if strings.Contains(fmt.Sprintf("%v %#v", report.RecoveryError, f.provider.job), "credential-canary") || strings.Contains(fmt.Sprintf("%#v", f.provider.job), f.provider.job.Ticket.Token) {
		t.Fatal("private job details escaped")
	}
	restarted := *f.driver
	recovered := restarted.Recover(t.Context(), *f.input.Reference, f.input)
	if recovered.RecoveryError != nil || recovered.Result.State != "unknown" || len(recovered.Attachments) != 0 || f.provider.ensureCount != 1 {
		t.Fatal("frozen observation did not recover", recovered.RecoveryError)
	}
	if err := f.input.BeforeCleanup(recovered); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Cleanup(t.Context(), *f.input.Reference); err != nil {
		t.Fatal(err)
	}
	if err := f.input.AfterCleanup(nil); err != nil {
		t.Fatal(err)
	}
}

func TestHostedCancellationAndDeadlineStopTheOwnedJob(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			f := fresh(t, "playwright")
			f.provider.mode = "running"
			ctx, cancel := context.WithCancel(t.Context())
			if deadline {
				ctx, cancel = context.WithTimeout(t.Context(), 150*time.Millisecond)
			} else {
				f.provider.startCallback = cancel
			}
			defer cancel()
			report := f.driver.Execute(ctx, f.input)
			want := "cancelled"
			state := "unknown"
			if deadline {
				want = "timeout"
				state = "timed-out"
			}
			if report.RecoveryError != nil || report.Result.State != state || report.Result.FailureKind != want || !report.Result.DurationUnavailable || report.Result.DurationMs != 0 || f.provider.removeCount != 1 {
				t.Fatal("cancellation was not controlled", report, f.events)
			}
		})
	}
}

func TestHostedCheckpointAndCleanupFailuresPreserveTheObservation(t *testing.T) {
	for _, mode := range []string{"checkpoint", "cleanup"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t, "playwright")
			f.checkpointError = mode == "checkpoint"
			f.provider.removeError = mode == "cleanup"
			report := f.driver.Execute(t.Context(), f.input)
			if report.Result.State != "passed" || len(report.EvidenceErrors) == 0 {
				t.Fatal("cleanup replaced or hid the observation", report)
			}
			if mode == "checkpoint" && (f.provider.removeCount != 0 || slices.Contains(f.events, "finished")) {
				t.Fatal("cleanup crossed a failed checkpoint", f.events)
			}
			if mode == "cleanup" && f.cleanupError == nil {
				t.Fatal("cleanup retry was not recorded")
			}
			if _, err := f.exchange.State(t.Context(), *f.input.Reference); err != nil {
				t.Fatal("staging was lost before recovery", err)
			}
			f.checkpointError = false
			f.provider.removeError = false
			if mode == "checkpoint" {
				if err := f.input.BeforeCleanup(report); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.driver.Cleanup(t.Context(), *f.input.Reference); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHostedControlsRejectForeignResourcesAndPrematureCleanup(t *testing.T) {
	f := fresh(t, "playwright")
	if err := f.driver.Cleanup(t.Context(), *f.input.Reference); err == nil || f.provider.removeCount != 0 {
		t.Fatal("cleanup bypassed checkpoint")
	}
	f.provider.mode = "wrong-image"
	report := f.driver.Execute(t.Context(), f.input)
	if !errors.Is(report.RecoveryError, hostedexecution.ErrOwnership) || f.provider.removeCount != 0 {
		t.Fatal("foreign image was accepted", report)
	}
	f.provider.inventory = []execution.Reference{*f.input.Reference}
	refs, err := f.driver.Inventory(t.Context(), "installation")
	if err != nil || len(refs) != 1 {
		t.Fatal(refs, err)
	}
	f.provider.inventory[0].Container = "foreign"
	refs, err = f.driver.Inventory(t.Context(), "installation")
	if err == nil || len(refs) != 0 {
		t.Fatal("inventory accepted a changed reference")
	}
}

func TestHostedAdmissionAndTimingRequireAReviewedImmutableSupervisedRuntime(t *testing.T) {
	mutations := []func(*fixture){
		func(f *fixture) { f.input.Artifact.Image = "sha256:" + strings.Repeat("a", 64) },
		func(f *fixture) { f.input.Test.RequiredBindings = []string{"credential"} },
		func(f *fixture) { f.input.Artifact.SourceProject = "another-project" },
		func(f *fixture) {
			f.input.Artifact.Provenance = &catalog.Provenance{SnapshotDigest: "sha256:" + strings.Repeat("b", 64)}
		},
		func(f *fixture) { f.driver.ExchangeOrigin = "https://jobs.example.test/" },
		func(f *fixture) {
			f.driver.ApproveArtifact = func(context.Context, catalog.Artifact) error { return errors.New("private review detail") }
		},
	}
	for i, mutation := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			f := fresh(t, "go")
			profile, err := f.driver.TimingProfile(t.Context(), f.input.Artifact)
			if err != nil || profile != "go-fixture-isolated-v1:go-fixture-v1:linux-arm64" {
				t.Fatal(profile, err)
			}
			mutation(f)
			if err := f.driver.Admit(t.Context(), f.input.Test, f.input.Artifact); err == nil || strings.Contains(err.Error(), "private review detail") {
				t.Fatal("unsupported admission escaped", err)
			}
			if f.provider.ensureCount != 0 {
				t.Fatal("admission dispatched work")
			}
		})
	}
}
