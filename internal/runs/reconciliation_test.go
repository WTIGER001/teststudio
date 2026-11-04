package runs_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerstate"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type recoveryExecutor struct {
	codedExecutor
	recoveries  atomic.Int32
	cleanups    atomic.Int32
	failCleanup atomic.Bool
	inventory   []execution.Reference
}

func (e *recoveryExecutor) Reference(ref execution.Reference) execution.Reference {
	ref.Profile = e.Profile()
	ref.Container = "container-" + ref.ID
	ref.Volume = "volume-" + ref.ID
	return ref
}
func (e *recoveryExecutor) Cleanup(context.Context, execution.Reference) error {
	e.cleanups.Add(1)
	if e.failCleanup.Load() {
		return errors.New("cleanup failed")
	}
	return nil
}
func (e *recoveryExecutor) Inventory(context.Context, string) ([]execution.Reference, error) {
	return e.inventory, nil
}
func (e *recoveryExecutor) Recover(ctx context.Context, ref execution.Reference, input execution.CodedInput) execution.CodedReport {
	e.recoveries.Add(1)
	return e.codedExecutor.Execute(ctx, input)
}

func lostExecution(t *testing.T, s *runs.Service, driver execution.ManagedExecutor, live bool) (runs.Run, execution.Reference) {
	t.Helper()
	v, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, fmt.Sprintf("lost-%d", time.Now().UnixNano()), "manual")
	if err != nil {
		t.Fatal(err)
	}
	v.Tests[0].Runner = "playwright"
	v.Tests[0].Source = "self-tests"
	v.Tests[0].Selector = "@HEALTH-1"
	v.Tests[0].Check = nil
	v.State = "running"
	v.Worker = "lost-worker"
	until := time.Now().UTC().Add(-time.Minute)
	if live {
		until = time.Now().UTC().Add(time.Minute)
	}
	v.LeaseUntil = &until
	v.CurrentTest = "HEALTH-1"
	v.CurrentAttempt = 1
	_, row, err := s.Get(t.Context(), v.Project, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := s.Store.Get(t.Context(), "run-queue", v.ID)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
	lease, _ := storage.Put("run-leases", v.ID, v.Project, 0, v.ID)
	ledger := workerstate.Ledger{Store: s.Store}
	installation, err := ledger.Installation(t.Context(), true)
	if err != nil {
		t.Fatal(err)
	}
	ref := driver.Reference(execution.Reference{ID: "execution-" + v.ID, Installation: installation, Project: v.Project, Run: v.ID, Test: v.Tests[0].ID, Attempt: 1, Worker: v.Worker})
	_, changes, err := ledger.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	changes = append(changes, c, lease, storage.Change{Record: queue, Delete: true})
	if err := s.Store.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	return v, ref
}
func awaitExecution(t *testing.T, s *runs.Service, ref execution.Reference, state string) workerstate.Record {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	ledger := workerstate.Ledger{Store: s.Store}
	for time.Now().Before(deadline) {
		v, _, err := ledger.Get(t.Context(), ref.Run, ref.ID)
		if err != nil {
			t.Fatal(err)
		}
		if v.State == state {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	v, _, _ := ledger.Get(t.Context(), ref.Run, ref.ID)
	t.Fatalf("worker state did not reach %s: %+v", state, v)
	return v
}
func recoveredDriver(state string) *recoveryExecutor {
	return &recoveryExecutor{codedExecutor: codedExecutor{report: execution.CodedReport{Result: execution.Result{State: state}, Attachments: []execution.Attachment{{Kind: "log", Stream: "playwright.stdout", Filename: "stdout.log", ContentType: "text/plain", Data: []byte("diagnostics")}}}}}
}

func TestLostWorkerRecoveryPreservesUnknownOutcomeAndDurableDiagnostics(t *testing.T) {
	for _, state := range []string{"passed", "failed"} {
		t.Run(state, func(t *testing.T) {
			s, _, _ := setup(t, nil)
			driver := recoveredDriver(state)
			run, ref := lostExecution(t, s, driver, false)
			dbPath := ""
			if s.Store.Dialect == "sqlite" {
				var seq int
				var name string
				if err := s.Store.DB.QueryRowContext(t.Context(), "PRAGMA database_list").Scan(&seq, &name, &dbPath); err != nil {
					t.Fatal(err)
				}
			}
			reopened, err := storage.Open(t.Context(), testkit.ConnectionString(t, s.Store, dbPath))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { reopened.Close() })
			s.Store = reopened
			s.RecoveryDriver = driver
			s.RecoveryInterval = 5 * time.Millisecond
			start(t, s)
			record := awaitExecution(t, s, ref, "cleaned")
			if !record.ResultSaved || !record.Recovered || record.RecoveredResult == nil || record.RecoveredResult.State != state {
				t.Fatalf("recovered observation: %+v", record)
			}
			final, _, err := s.Get(t.Context(), run.Project, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if final.State != "incomplete" || final.Outcome != "unknown" || final.Counts.Passed != 0 {
				t.Fatal("lost worker became a passing logical run")
			}
			attempts, err := s.Store.All(t.Context(), "attempts/"+run.ID, run.Project)
			if err != nil || len(attempts) != 1 {
				t.Fatalf("recovered attempts: %d %v", len(attempts), err)
			}
			attempt, err := storage.Decode[execution.Result](attempts[0])
			if err != nil || !attempt.Recovered || attempt.State != state {
				t.Fatal("recovered attempt was lost")
			}
			artifacts, err := s.Store.All(t.Context(), "artifacts/"+run.ID, run.Project)
			if err != nil || len(artifacts) != 1 {
				t.Fatal("diagnostic artifact was not retained")
			}
			if driver.cleanups.Load() != 1 || driver.recoveries.Load() != 1 {
				t.Fatal("recovery ran twice")
			}
		})
	}
}

func TestLiveWorkerAndOtherInstallationResourcesArePreserved(t *testing.T) {
	s, _, _ := setup(t, nil)
	driver := recoveredDriver("passed")
	run, ref := lostExecution(t, s, driver, true)
	foreign := ref
	foreign.Installation = "another-installation"
	driver.inventory = []execution.Reference{foreign}
	s.RecoveryDriver = driver
	s.RecoveryInterval = 5 * time.Millisecond
	stop := start(t, s)
	time.Sleep(60 * time.Millisecond)
	stop()
	current, _, err := s.Get(t.Context(), run.Project, run.ID)
	if err != nil || current.State != "running" || driver.cleanups.Load() != 0 || driver.recoveries.Load() != 0 {
		t.Fatal("active execution was interrupted")
	}
}

func TestCleanupFailureRetriesWithoutDuplicatingAttemptsOrArtifacts(t *testing.T) {
	s, _, _ := setup(t, nil)
	driver := recoveredDriver("failed")
	driver.failCleanup.Store(true)
	run, ref := lostExecution(t, s, driver, false)
	s.RecoveryDriver = driver
	s.RecoveryInterval = 5 * time.Millisecond
	start(t, s)
	failed := awaitExecution(t, s, ref, "cleanup-failed")
	if failed.Error == "" || driver.recoveries.Load() != 1 {
		t.Fatal("cleanup failure was hidden")
	}
	ledger := workerstate.Ledger{Store: s.Store}
	record, row, err := ledger.Get(t.Context(), run.ID, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	record.NextCheck = time.Now().Add(-time.Second)
	driver.failCleanup.Store(false)
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
	if err == nil {
		err = s.Store.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
	awaitExecution(t, s, ref, "cleaned")
	artifacts, err := s.Store.All(t.Context(), "artifacts/"+run.ID, run.Project)
	if err != nil || len(artifacts) != 1 || driver.recoveries.Load() != 1 || driver.cleanups.Load() != 2 {
		t.Fatal("cleanup retry duplicated observations or evidence")
	}
}

func TestReplicaRecoveryClaimsDoNotDuplicateDiagnostics(t *testing.T) {
	s, _, _ := setup(t, nil)
	driver := recoveredDriver("failed")
	run, ref := lostExecution(t, s, driver, false)
	s.RecoveryDriver = driver
	s.RecoveryInterval = 5 * time.Millisecond
	peer := *s
	start(t, s)
	start(t, &peer)
	awaitExecution(t, s, ref, "cleaned")
	time.Sleep(30 * time.Millisecond)
	attempts, err := s.Store.All(t.Context(), "attempts/"+run.ID, run.Project)
	if err != nil || len(attempts) != 1 {
		t.Fatal("replica recovery duplicated attempts")
	}
	artifacts, err := s.Store.All(t.Context(), "artifacts/"+run.ID, run.Project)
	if err != nil || len(artifacts) != 1 || driver.cleanups.Load() != 1 || driver.recoveries.Load() != 1 {
		t.Fatalf("replica recovery duplicated provider work: cleanups=%d recoveries=%d", driver.cleanups.Load(), driver.recoveries.Load())
	}
}

type interruptedUpload struct{ artifacts.Store }

func (interruptedUpload) Put(context.Context, string, io.Reader, int64) (artifacts.Object, error) {
	return artifacts.Object{}, errors.New("temporary artifact storage outage")
}

func TestRecoveredEvidenceRetriesAfterRestartWithoutQuotaDuplication(t *testing.T) {
	s, _, _ := setup(t, nil)
	driver := recoveredDriver("failed")
	run, ref := lostExecution(t, s, driver, false)
	originalBlobs := s.Evidence.Blobs
	s.Evidence.Blobs = interruptedUpload{originalBlobs}
	s.RecoveryDriver = driver
	s.RecoveryInterval = 5 * time.Millisecond
	stop := start(t, s)
	failed := awaitExecution(t, s, ref, "cleanup-failed")
	stop()
	if !failed.ResultSaved || failed.EvidenceComplete || failed.EvidenceAbandoned || driver.cleanups.Load() != 0 {
		t.Fatalf("transient capture loss was abandoned: %+v", failed)
	}
	firstQuota, err := s.Store.Get(t.Context(), "artifact-quotas", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.Evidence.Blobs = originalBlobs
	ledger := workerstate.Ledger{Store: s.Store}
	record, row, err := ledger.Get(t.Context(), run.ID, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	record.NextCheck = time.Now().Add(-time.Second)
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
	if err == nil {
		err = s.Store.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	complete := awaitExecution(t, s, ref, "cleaned")
	if !complete.EvidenceComplete || complete.EvidenceAbandoned || driver.recoveries.Load() != 2 || driver.cleanups.Load() != 1 {
		t.Fatalf("capture retry failed: %+v", complete)
	}
	files, err := s.Store.All(t.Context(), "artifacts/"+run.ID, run.Project)
	if err != nil || len(files) != 1 {
		t.Fatal("capture retry duplicated evidence")
	}
	quota, err := s.Store.Get(t.Context(), "artifact-quotas", run.ID)
	if err != nil || string(firstQuota.Data) != string(quota.Data) {
		t.Fatal("capture retry counted bytes twice")
	}
	file, err := storage.Decode[evidence.Artifact](files[0])
	if err != nil || file.State != "available" || file.Sequence != 1 {
		t.Fatalf("recovered file metadata: %+v %v", file, err)
	}
}

type pendingEvidenceUpload struct {
	artifacts.Store
	entered chan struct{}
	release chan struct{}
}

func (b *pendingEvidenceUpload) Put(ctx context.Context, key string, source io.Reader, limit int64) (artifacts.Object, error) {
	close(b.entered)
	select {
	case <-b.release:
		return b.Store.Put(ctx, key, source, limit)
	case <-ctx.Done():
		return artifacts.Object{}, ctx.Err()
	}
}

func TestPendingDuplicateEvidenceKeepsCheckpointUntilUploadCommits(t *testing.T) {
	s, _, _ := setup(t, nil)
	driver := recoveredDriver("failed")
	run, ref := lostExecution(t, s, driver, false)
	blobs := s.Evidence.Blobs
	s.Evidence.Blobs = interruptedUpload{blobs}
	s.RecoveryDriver = driver
	s.RecoveryInterval = 5 * time.Millisecond
	stop := start(t, s)
	awaitExecution(t, s, ref, "cleanup-failed")
	stop()
	s.Evidence.Blobs = blobs
	files, err := s.Store.All(t.Context(), "artifacts/"+run.ID, run.Project)
	if err != nil || len(files) != 1 {
		t.Fatal("failed upload manifest missing", err)
	}
	file, err := storage.Decode[evidence.Artifact](files[0])
	if err != nil {
		t.Fatal(err)
	}
	gate := &pendingEvidenceUpload{Store: blobs, entered: make(chan struct{}), release: make(chan struct{})}
	uploader := &evidence.Service{Store: s.Store, Blobs: gate}
	uploadCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	written := make(chan error, 1)
	go func() { _, err := uploader.SaveBytes(uploadCtx, file, []byte("diagnostics"), 100<<20); written <- err }()
	defer func() {
		cancel()
		select {
		case <-written:
		case <-time.After(time.Second):
			t.Error("owned upload did not stop")
		}
	}()
	select {
	case <-gate.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("retry did not claim the attachment")
	}
	ledger := workerstate.Ledger{Store: s.Store}
	advance := func() {
		record, row, err := ledger.Get(t.Context(), run.ID, ref.ID)
		if err != nil {
			t.Fatal(err)
		}
		record.NextCheck = time.Now().Add(-time.Second)
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
		if err == nil {
			err = s.Store.Commit(t.Context(), change)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	advance()
	stop = start(t, s)
	var pending workerstate.Record
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		pending, _, err = ledger.Get(t.Context(), run.ID, ref.ID)
		if err != nil {
			t.Fatal(err)
		}
		if pending.ReconcileAttempts >= 2 && pending.State == "cleanup-failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop()
	if pending.EvidenceComplete || pending.EvidenceAbandoned || driver.cleanups.Load() != 0 || driver.recoveries.Load() != 2 {
		t.Fatal("pending duplicate discarded its source checkpoint", pending)
	}
	close(gate.release)
	select {
	case err := <-written:
		if err != nil {
			t.Fatal(err)
		}
		written <- nil
	case <-time.After(3 * time.Second):
		t.Fatal("owned upload did not finish")
	}
	advance()
	stop = start(t, s)
	complete := awaitExecution(t, s, ref, "cleaned")
	stop()
	if !complete.EvidenceComplete || driver.cleanups.Load() != 1 || driver.recoveries.Load() != 3 {
		t.Fatal("checkpoint did not finish after upload", complete)
	}
	files, err = s.Store.All(t.Context(), "artifacts/"+run.ID, run.Project)
	if err != nil || len(files) != 1 {
		t.Fatal("pending retry duplicated evidence", err)
	}
	quota, err := s.Store.Get(t.Context(), "artifact-quotas", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := storage.Decode[evidence.Quota](quota)
	if err != nil || reserved.Bytes != int64(len("diagnostics")) {
		t.Fatal("pending retry duplicated quota", reserved, err)
	}
	current, _, err := s.Get(t.Context(), run.Project, run.ID)
	if err != nil || current.State != "incomplete" || current.Outcome != "unknown" || current.Counts.Completed != 0 {
		t.Fatal("diagnostic recovery changed the logical outcome", current, err)
	}
	attempts, _ := s.Store.All(t.Context(), "attempts/"+run.ID, run.Project)
	runs, _ := s.Store.All(t.Context(), "runs", run.Project)
	if len(attempts) != 1 || len(runs) != 1 {
		t.Fatal("diagnostic recovery duplicated attempts or runs")
	}
}
