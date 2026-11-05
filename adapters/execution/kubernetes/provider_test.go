package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestPodProviderUsesTheNormalPrivateObservationAndCleanupPath(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, mode := range []string{"passed", "failed", "rejected"} {
			t.Run(runner+"/"+mode, func(t *testing.T) {
				f := fresh(t, runner)
				f.mode = mode
				report := f.driver.Execute(t.Context(), f.input)
				want := mode
				if mode == "rejected" {
					want = "unknown"
				}
				if report.RecoveryError != nil || report.Result.State != want {
					t.Fatal("provider execution did not preserve the observation", report)
				}
				f.mu.Lock()
				posts, deletes, value := f.posts, f.deletes, f.value
				f.mu.Unlock()
				if posts != 1 || value != nil || mode != "rejected" && deletes != 1 {
					t.Fatal("owned resources were not cleaned", posts, deletes)
				}
				if mode != "rejected" && (report.Result.DurationMs != 1 || len(report.Attachments) != 1 || string(report.Attachments[0].Data) != "private provider fixture evidence") {
					t.Fatal("provider timing or evidence replaced the test observation", report)
				}
				entry, _, err := (workerstate.Ledger{Store: f.store}).Get(t.Context(), f.input.Reference.Run, f.input.Reference.ID)
				if err != nil || entry.State != "cleaned" {
					t.Fatal(entry, err)
				}
				row, err := f.store.Get(t.Context(), Bucket, f.input.Reference.ID)
				if err != nil || strings.Contains(string(row.Data), "api-private-token") {
					t.Fatal("private API data entered the journal", err)
				}
			})
		}
	}
}

func TestPodProviderRecoversLostCreationAcknowledgementsWithoutAnotherLaunch(t *testing.T) {
	f := fresh(t, "playwright")
	f.mode = "lost-ack"
	report := f.driver.Execute(t.Context(), f.input)
	if report.RecoveryError == nil {
		t.Fatal("ambiguous creation passed as completed")
	}
	restarted, err := New(Options{Config: f.provider.config, Store: f.store, Inspector: imageInspector{"playwright"}, Transport: f.server.Client().Transport, Token: f.provider.token})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	driver := *f.driver
	driver.Provider = restarted
	recovered := driver.Recover(t.Context(), *f.input.Reference, f.input)
	if recovered.RecoveryError != nil || recovered.Result.State != "unknown" || len(recovered.Attachments) != 0 {
		t.Fatal("frozen provider observation was lost", recovered)
	}
	if err := f.input.BeforeCleanup(recovered); err != nil {
		t.Fatal(err)
	}
	if err := driver.Cleanup(t.Context(), *f.input.Reference); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 || f.deletes != 1 {
		t.Fatal("recovery dispatched again", f.posts, f.deletes)
	}
}

func TestPodProviderRecoversPrivateReportsAfterACheckpointFailure(t *testing.T) {
	f := fresh(t, "playwright")
	f.checkpointError = true
	report := f.driver.Execute(t.Context(), f.input)
	if report.Result.State != "passed" || len(report.EvidenceErrors) == 0 {
		t.Fatal("checkpoint failure lost its observation", report)
	}
	restarted, err := New(Options{Config: f.provider.config, Store: f.store, Inspector: imageInspector{"playwright"}, Transport: f.server.Client().Transport, Token: f.provider.token})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	driver := *f.driver
	driver.Provider = restarted
	recovered := driver.Recover(t.Context(), *f.input.Reference, f.input)
	if recovered.RecoveryError != nil || recovered.Result.State != "passed" || len(recovered.Attachments) != 1 {
		t.Fatal("accepted private report did not recover", recovered)
	}
	f.checkpointError = false
	if err := f.input.BeforeCleanup(recovered); err != nil {
		t.Fatal(err)
	}
	if err := driver.Cleanup(t.Context(), *f.input.Reference); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 || f.deletes != 1 {
		t.Fatal("report recovery repeated a launch", f.posts, f.deletes)
	}
}

func TestPodProviderDeadlineStopsBeforeDeletingItsWorker(t *testing.T) {
	f := fresh(t, "playwright")
	f.mode = "running"
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	report := f.driver.Execute(ctx, f.input)
	if report.RecoveryError != nil || report.Result.State != "timed-out" || report.Result.FailureKind != "timeout" || !report.Result.DurationUnavailable || report.Result.DurationMs != 0 {
		t.Fatal("deadline did not remain distinct", report)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.patches != 1 || f.deletes != 1 || f.value != nil {
		t.Fatal("deadline did not stop and clean the worker", f.patches, f.deletes)
	}
}

func TestPodProviderRejectsUnsafePolicyAndForeignResources(t *testing.T) {
	f := fresh(t, "go")
	if err := f.provider.Remove(t.Context(), *f.input.Reference); err == nil {
		t.Fatal("cleanup ignored a missing checkpoint")
	}
	f.unsafePolicy = true
	report := f.driver.Execute(t.Context(), f.input)
	if report.RecoveryError != nil || report.Result.State != "unknown" {
		t.Fatal("unsafe policy permitted dispatch")
	}
	f.mu.Lock()
	posts := f.posts
	f.unsafePolicy = false
	f.mu.Unlock()
	if posts != 0 {
		t.Fatal("unsafe namespace received test code")
	}
	if err := f.driver.Cleanup(t.Context(), *f.input.Reference); err != nil {
		t.Fatal("undispatched work could not clean", err)
	}
}

func TestPodProviderCreationReservationIsSharedAcrossControllers(t *testing.T) {
	f := fresh(t, "go")
	f.mode = "running"
	ref := *f.input.Reference
	ticket, err := f.exchange.Prepare(t.Context(), ref, f.input, time.Now().UTC().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	job := hostedexecution.Job{Reference: ref, Image: f.input.Artifact.Image, Runner: "go", ExchangeOrigin: f.provider.config.ExchangeOrigin, Ticket: ticket}
	var group sync.WaitGroup
	results := make(chan error, 6)
	for range 6 {
		group.Add(1)
		go func() { defer group.Done(); results <- f.provider.Ensure(t.Context(), job) }()
	}
	group.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, hostedexecution.ErrUnavailable) {
			t.Fatal(err)
		}
	}
	f.mu.Lock()
	posts := f.posts
	f.mu.Unlock()
	if posts != 1 {
		t.Fatal("controllers created duplicate workers", posts)
	}
	row, err := f.store.Get(t.Context(), Bucket, ref.ID)
	if err != nil || strings.Contains(string(row.Data), ticket.Token) {
		t.Fatal("scoped ticket entered the journal", err)
	}
	if err := f.provider.Stop(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	status, err := f.provider.Observe(t.Context(), ref)
	if err != nil || status.Phase != "stopped" {
		t.Fatal(status, err)
	}
}

func TestPodProviderInventoryValidatesOwnershipAndKeepsCreationTombstones(t *testing.T) {
	f := fresh(t, "playwright")
	f.mode = "running"
	ref := *f.input.Reference
	ticket, err := f.exchange.Prepare(t.Context(), ref, f.input, time.Now().UTC().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	job := hostedexecution.Job{Reference: ref, Image: f.input.Artifact.Image, Runner: "playwright", ExchangeOrigin: f.provider.config.ExchangeOrigin, Ticket: ticket}
	if err := f.provider.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	refs, err := f.provider.Inventory(t.Context(), ref.Installation, ref.Profile)
	if err != nil || len(refs) != 1 || refs[0] != ref {
		t.Fatal(refs, err)
	}
	f.mu.Lock()
	f.value.Spec["containers"].([]any)[0].(map[string]any)["image"] = "registry.example.test/foreign@sha256:" + strings.Repeat("b", 64)
	f.mu.Unlock()
	if _, err := f.provider.Observe(t.Context(), ref); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("foreign image was accepted", err)
	}
	if err := f.provider.Stop(t.Context(), ref); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("foreign Pod was changed", err)
	}
	row, err := f.store.Get(t.Context(), Bucket, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	var record dispatch
	if json.Unmarshal(row.Data, &record) != nil {
		t.Fatal("journal record")
	}
	record.Removed = true
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if err := f.provider.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) {
		t.Fatal("completed reference dispatched again", err)
	}
	if strings.Contains(fmt.Sprintf("%#v", f.value), ticket.Token) {
		t.Fatal("private worker ticket entered a diagnostic")
	}
}

func TestPodProviderKeepsASeenWorkerPendingWhenItsPodDisappears(t *testing.T) {
	f := fresh(t, "go")
	f.mode = "running"
	ref := *f.input.Reference
	ticket, err := f.exchange.Prepare(t.Context(), ref, f.input, time.Now().UTC().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	job := hostedexecution.Job{Reference: ref, Image: f.input.Artifact.Image, Runner: "go", ExchangeOrigin: f.provider.config.ExchangeOrigin, Ticket: ticket}
	if err := f.provider.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	first, err := f.driver.TimingProfile(t.Context(), f.input.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.value.Status.Phase = "Failed"
	f.mu.Unlock()
	if _, err := f.provider.Observe(t.Context(), ref); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("phase without a terminated container proved worker exit", err)
	}
	f.mu.Lock()
	f.value = nil
	f.mu.Unlock()
	if _, err := f.provider.Observe(t.Context(), ref); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("API absence was treated as worker termination", err)
	}
	if err := f.provider.Stop(t.Context(), ref); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("lost worker was treated as stopped", err)
	}
	f.provider.config.RuntimeClass = "another-runtime"
	second, err := f.driver.TimingProfile(t.Context(), f.input.Artifact)
	if err != nil || first == second {
		t.Fatal("runtime class change reused a timing baseline", err)
	}
}

func TestPodAdmissionMutationCannotReleaseExecutableWork(t *testing.T) {
	f := fresh(t, "playwright")
	f.mode = "injected"
	report := f.driver.Execute(t.Context(), f.input)
	if report.RecoveryError == nil {
		t.Fatal("unexpected Pod mutation was admitted")
	}
	state, err := f.exchange.State(t.Context(), *f.input.Reference)
	if err != nil || !state.DispatchPending || !state.Revoked || state.Submitted || state.StartedAt != nil {
		t.Fatal("unverified Pod received work", state, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 || f.deletes != 0 {
		t.Fatal("foreign Pod was recreated or removed", f.posts, f.deletes)
	}
}

func TestPodRemovalWaitsForAcknowledgedAsynchronousDeletion(t *testing.T) {
	f := fresh(t, "go")
	f.deleteReads = 2
	report := f.driver.Execute(t.Context(), f.input)
	entry, _, err := (workerstate.Ledger{Store: f.store}).Get(t.Context(), f.input.Reference.Run, f.input.Reference.ID)
	if err != nil || report.Result.State != "passed" || entry.State != "cleaned" || entry.CleanupAttempts != 1 {
		t.Fatal("acknowledged asynchronous deletion required recovery", report, entry, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 || f.deletes != 1 || f.value != nil {
		t.Fatal("cleanup repeated dispatch or deletion", f.posts, f.deletes)
	}
}

func TestPodRemovalRetainsForeignReplacementAfterAcknowledgedDeletion(t *testing.T) {
	f := fresh(t, "go")
	f.replaceAfterDelete = true
	f.driver.Execute(t.Context(), f.input)
	entry, _, err := (workerstate.Ledger{Store: f.store}).Get(t.Context(), f.input.Reference.Run, f.input.Reference.ID)
	if err != nil || entry.State != "cleanup-failed" {
		t.Fatal("a foreign replacement passed cleanup", entry, err)
	}
	record, _, err := f.provider.load(t.Context(), *f.input.Reference)
	if err != nil || record.Removed {
		t.Fatal("uncertain deletion entered the removal journal", record, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.value == nil || f.value.Metadata.UID != "foreign-replacement" || f.deletes != 1 {
		t.Fatal("cleanup removed or retried deletion of a foreign Pod")
	}
}
