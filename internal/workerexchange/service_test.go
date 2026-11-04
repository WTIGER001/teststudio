package workerexchange_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type fixture struct {
	service  *workerexchange.Service
	ref      execution.Reference
	input    execution.CodedInput
	deadline time.Time
	store    *storage.Store
	blobs    artifacts.Store
}

func fresh(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	db, err := testkit.Open(t, filepath.Join(root, "exchange.db"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filesystem.New(filepath.Join(root, "private"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close(); db.Close() })
	ref := execution.Reference{ID: "attempt-1", Installation: "installation", Project: "project", Run: "run", Test: "TEST-1", Attempt: 1, Worker: "controller", Profile: "hosted-protocol-v1", Container: "owned-job"}
	ledger := workerstate.Ledger{Store: db}
	_, changes, err := ledger.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := storage.Put("attempt-authority", ref.ID, ref.Project, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), append(changes, authority)...); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().UTC().Add(time.Minute)
	service := &workerexchange.Service{Store: db, Blobs: files, Authorize: func(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
		row, err := db.Get(ctx, "attempt-authority", ref.ID)
		if err != nil {
			return workerstate.Access{}, err
		}
		allowed, err := storage.Decode[bool](row)
		if err != nil || !allowed {
			return workerstate.Access{}, workerexchange.ErrDenied
		}
		return workerstate.Access{Deadline: deadline, ProjectBytes: 2 << 30, RunBytes: 1 << 30, Records: []storage.Record{row}}, nil
	}}
	input := execution.CodedInput{RunID: ref.Run, Test: catalog.Test{ID: ref.Test, Runner: "playwright", Type: "browser", Selector: "@TEST-1", Title: "private-not-required", Check: &catalog.HTTPCheck{Headers: map[string]string{"Authorization": "private-target-value"}}}, Artifact: catalog.Artifact{Image: "registry.example.test/suite@sha256:" + strings.Repeat("a", 64), SourceProject: ref.Project, SourceSnapshot: "snapshot", Discovered: map[string]string{ref.Test: "@TEST-1", "UNRELATED": "private-unrelated-selector"}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000, CredentialReference: "private-binding-reference", RedactionValues: []string{"private-redaction-value"}}
	return fixture{service, ref, input, deadline, db, files}
}

func (f fixture) prepare(t *testing.T) workerexchange.Ticket {
	t.Helper()
	ticket, err := f.service.Prepare(t.Context(), f.ref, f.input, f.deadline)
	if err != nil {
		t.Fatal(err)
	}
	return ticket
}
func (f fixture) revokeAuthority(t *testing.T) {
	t.Helper()
	row, err := f.store.Get(t.Context(), "attempt-authority", f.ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, false)
	if err := f.store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
}
func submission(f fixture, body []byte) workerexchange.Submission {
	now := time.Now().UTC()
	sum := sha256.Sum256(body)
	return workerexchange.Submission{Report: execution.CodedReport{Schema: "teststudio.playwright/v1", TestID: f.ref.Test, Selector: f.input.Test.Selector, Result: execution.Result{TestID: f.ref.Test, State: "passed", StartedAt: now.Add(-time.Millisecond), FinishedAt: now, DurationMs: 1}, Attachments: []execution.Attachment{{Path: "log.txt", Kind: "log", Filename: "log.txt", ContentType: "text/plain"}}}, Files: []workerexchange.File{{Path: "log.txt", Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])}}}
}

func TestScopedWorkSurvivesControllerRestartAndHasOneWorkerInstance(t *testing.T) {
	f := fresh(t)
	ticket := f.prepare(t)
	if strings.Contains(fmt.Sprintf("%+v %#v", ticket, ticket), ticket.Token) {
		t.Fatal("ticket formatting exposed its secret")
	}
	encoded, _ := json.Marshal(ticket)
	if bytes.Contains(encoded, []byte(ticket.Token)) {
		t.Fatal("ticket serialization exposed its secret")
	}
	restarted := &workerexchange.Service{Store: f.store, Blobs: f.blobs, Authorize: f.service.Authorize}
	again, err := restarted.Prepare(t.Context(), f.ref, f.input, f.deadline)
	if err != nil || ticket.Token != again.Token {
		t.Fatal("restart changed the durable ticket", err)
	}
	packet, err := restarted.Input(t.Context(), ticket.Execution, ticket.Token, "worker-one")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"private-target-value", "private-binding-reference", "private-redaction-value", "private-unrelated-selector", "private-not-required", ticket.Token} {
		if bytes.Contains(packet.Input, []byte(value)) {
			t.Fatal("private controller metadata entered worker input")
		}
	}
	if _, err := restarted.Input(t.Context(), ticket.Execution, ticket.Token, "worker-two"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("another instance claimed the same attempt", err)
	}
	if _, err := restarted.Input(t.Context(), "another-attempt", ticket.Token, "worker-one"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("ticket crossed attempt ownership", err)
	}
	f.revokeAuthority(t)
	if _, err := restarted.Input(t.Context(), ticket.Execution, ticket.Token, "worker-one"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("revoked authority retained packet access", err)
	}
}

func TestConcurrentControllersIssueOneTicketAndOneInstanceWins(t *testing.T) {
	f := fresh(t)
	var group sync.WaitGroup
	results := make(chan workerexchange.Ticket, 8)
	failures := make(chan error, 8)
	for range 8 {
		group.Go(func() {
			ticket, err := f.service.Prepare(t.Context(), f.ref, f.input, f.deadline)
			if err != nil {
				failures <- err
			} else {
				results <- ticket
			}
		})
	}
	group.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	var token string
	for ticket := range results {
		if token != "" && token != ticket.Token {
			t.Fatal("controllers issued different attempt tickets")
		}
		token = ticket.Token
	}
	wins := make(chan string, 2)
	for _, instance := range []string{"worker-one", "worker-two"} {
		group.Go(func() {
			if _, err := f.service.Input(t.Context(), f.ref.ID, token, instance); err == nil {
				wins <- instance
			} else if !errors.Is(err, workerexchange.ErrDenied) {
				t.Error(err)
			}
		})
	}
	group.Wait()
	close(wins)
	if len(wins) != 1 {
		t.Fatal("attempt had multiple worker instances", len(wins))
	}
}

func TestFrozenReportsRequireCorrectBytesAndRemainRecoverableUntilCheckpoint(t *testing.T) {
	f := fresh(t)
	ticket := f.prepare(t)
	if _, err := f.service.Input(t.Context(), ticket.Execution, ticket.Token, "worker"); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	if err := f.service.Started(t.Context(), ticket.Execution, ticket.Token, "worker", at); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Started(t.Context(), ticket.Execution, ticket.Token, "worker", at); err != nil {
		t.Fatal("duplicate start failed", err)
	}
	body := []byte("owned retained log")
	report := submission(f, body)
	for range 2 {
		if err := f.service.Submit(t.Context(), ticket.Execution, ticket.Token, "worker", report); err != nil {
			t.Fatal(err)
		}
	}
	changed := report
	changed.Report.Result.Error = "different"
	if err := f.service.Submit(t.Context(), ticket.Execution, ticket.Token, "worker", changed); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("accepted report changed", err)
	}
	if err := f.service.Upload(t.Context(), ticket.Execution, ticket.Token, "worker", "log.txt", bytes.NewReader([]byte("wrong"))); !errors.Is(err, workerexchange.ErrInvalid) {
		t.Fatal("incorrect attachment bytes were accepted", err)
	}
	_, started, complete, err := f.service.Observation(t.Context(), f.ref, f.input, nil)
	if err != nil || complete || started == nil || !started.Equal(at) {
		t.Fatal("incomplete evidence or progress was lost", err)
	}
	if err := f.service.Cleanup(t.Context(), f.ref); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("cleanup preceded the result checkpoint", err)
	}
	for range 2 {
		if err := f.service.Upload(t.Context(), ticket.Execution, ticket.Token, "worker", "log.txt", bytes.NewReader(body)); err != nil {
			t.Fatal(err)
		}
	}
	restarted := &workerexchange.Service{Store: f.store, Blobs: f.blobs, Authorize: f.service.Authorize}
	observed, _, complete, err := restarted.Observation(t.Context(), f.ref, f.input, nil)
	if err != nil || !complete || len(observed.Attachments) != 1 || !bytes.Equal(observed.Attachments[0].Data, body) {
		t.Fatal("restart lost the report or owned evidence", err)
	}
	if _, _, _, err := restarted.Observation(t.Context(), f.ref, f.input, errors.New("provider process failed")); !errors.Is(err, workerexchange.ErrInvalid) {
		t.Fatal("infrastructure process failure became a passing result", err)
	}
	if err := restarted.Revoke(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Upload(t.Context(), ticket.Execution, ticket.Token, "worker", "log.txt", bytes.NewReader(body)); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("revoked ticket retained upload access", err)
	}
	ledger := workerstate.Ledger{Store: f.store}
	entry, row, err := ledger.Get(t.Context(), f.ref.Run, f.ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	entry.ResultSaved, entry.EvidenceComplete = true, true
	c, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
	if err := f.store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Cleanup(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Cleanup(t.Context(), f.ref); err != nil {
		t.Fatal("lost cleanup acknowledgement was not retryable", err)
	}
	if _, err := f.store.Get(t.Context(), workerexchange.Bucket, f.ref.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("checkpointed exchange was not removed", err)
	}
}

func TestInvalidOwnershipReportAndCaptureLimitsFailClosed(t *testing.T) {
	f := fresh(t)
	ticket := f.prepare(t)
	f.service.Input(t.Context(), ticket.Execution, ticket.Token, "worker")
	for _, mutate := range []func(*workerexchange.Submission){
		func(s *workerexchange.Submission) { s.Report.TestID = "different" },
		func(s *workerexchange.Submission) { s.Report.Attachments[0].Kind = "screenshot" },
		func(s *workerexchange.Submission) { s.Report.Result.State = "running" },
		func(s *workerexchange.Submission) { s.Report.Result.DurationMs = -1 },
		func(s *workerexchange.Submission) { s.Files[0].Path = "../escape" },
		func(s *workerexchange.Submission) { s.Files[0].Size = f.input.Capture.MaxFileBytes + 1 },
		func(s *workerexchange.Submission) { s.Files[0].SHA256 = "invalid" },
		func(s *workerexchange.Submission) {
			s.Report.Attachments[0].ContentType = "text/plain\r\nInjected: value"
		},
	} {
		report := submission(f, []byte("owned"))
		mutate(&report)
		if err := f.service.Submit(t.Context(), ticket.Execution, ticket.Token, "worker", report); !errors.Is(err, workerexchange.ErrInvalid) {
			t.Fatal("invalid report was accepted", err)
		}
	}
	input := f.input
	input.Artifact.SourceSnapshot = strings.Repeat("x", execution.MaxWorkItemBytes+1)
	if _, err := f.service.Prepare(t.Context(), f.ref, input, f.deadline); !errors.Is(err, workerexchange.ErrInvalid) {
		t.Fatal("oversized input was accepted", err)
	}
	other := f.ref
	other.Project = "foreign"
	if _, err := f.service.Prepare(t.Context(), other, f.input, f.deadline); !errors.Is(err, workerexchange.ErrInvalid) {
		t.Fatal("artifact crossed project ownership", err)
	}
}

type blockedStore struct {
	artifacts.Store
	started, release chan struct{}
}

func (b *blockedStore) Put(ctx context.Context, key string, reader io.Reader, limit int64) (artifacts.Object, error) {
	close(b.started)
	select {
	case <-b.release:
	case <-ctx.Done():
		return artifacts.Object{}, ctx.Err()
	}
	return b.Store.Put(ctx, key, reader, limit)
}

func TestCleanupWaitsForAnOwnedUploadAndCannotLeaveLateOrphanBytes(t *testing.T) {
	f := fresh(t)
	blocked := &blockedStore{f.blobs, make(chan struct{}), make(chan struct{})}
	f.service.Blobs = blocked
	ticket := f.prepare(t)
	f.service.Input(t.Context(), ticket.Execution, ticket.Token, "worker")
	body := []byte("owned upload")
	if err := f.service.Submit(t.Context(), ticket.Execution, ticket.Token, "worker", submission(f, body)); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- f.service.Upload(t.Context(), ticket.Execution, ticket.Token, "worker", "log.txt", bytes.NewReader(body))
	}()
	select {
	case <-blocked.started:
	case <-time.After(time.Second):
		t.Fatal("upload did not start")
	}
	t.Cleanup(func() {
		select {
		case <-blocked.release:
		default:
			close(blocked.release)
		}
	})
	ledger := workerstate.Ledger{Store: f.store}
	entry, row, err := ledger.Get(t.Context(), f.ref.Run, f.ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	entry.ResultSaved, entry.EvidenceComplete = true, true
	c, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
	if err := f.store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Cleanup(t.Context(), f.ref); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("cleanup raced a live upload", err)
	}
	close(blocked.release)
	select {
	case err := <-done:
		if !errors.Is(err, workerexchange.ErrDenied) {
			t.Fatal("revoked in-flight upload outcome", err)
		}
	case <-time.After(time.Second):
		t.Fatal("upload did not finish")
	}
	if err := f.service.Cleanup(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
}

func TestExpiredTicketsAndChangedWorkCannotReactivateAnAttempt(t *testing.T) {
	f := fresh(t)
	ticket := f.prepare(t)
	changed := f.input
	changed.TimeoutMs++
	if _, err := f.service.Prepare(t.Context(), f.ref, changed, f.deadline); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("attempt input was replaced", err)
	}
	row, err := f.store.Get(t.Context(), workerexchange.Bucket, f.ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := storage.Decode[workerexchange.Record](row)
	if err != nil {
		t.Fatal(err)
	}
	record.Deadline = time.Now().Add(-time.Second)
	update, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
	if err := f.store.Commit(t.Context(), update); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Input(t.Context(), ticket.Execution, ticket.Token, "worker"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("expired ticket retained input access", err)
	}
}

func TestStagingQuotaIsSharedAcrossAttemptsAndReleasedOnlyAfterCleanup(t *testing.T) {
	f := fresh(t)
	original := f.service.Authorize
	f.service.Authorize = func(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
		access, err := original(ctx, ref)
		access.ProjectBytes = 4
		access.RunBytes = 4
		return access, err
	}
	first := f.prepare(t)
	if _, err := f.service.Input(t.Context(), first.Execution, first.Token, "first"); err != nil {
		t.Fatal(err)
	}
	body := []byte("four")
	firstReport := submission(f, body)
	for range 2 {
		if err := f.service.Submit(t.Context(), first.Execution, first.Token, "first", firstReport); err != nil {
			t.Fatal(err)
		}
	}
	other := f
	other.ref.ID = "attempt-2"
	other.ref.Run = "another-run"
	other.input.RunID = other.ref.Run
	ledger := workerstate.Ledger{Store: f.store}
	_, changes, err := ledger.Begin(other.ref)
	if err != nil {
		t.Fatal(err)
	}
	authority, _ := storage.Put("attempt-authority", other.ref.ID, other.ref.Project, 0, true)
	if err := f.store.Commit(t.Context(), append(changes, authority)...); err != nil {
		t.Fatal(err)
	}
	second := other.prepare(t)
	if _, err := f.service.Input(t.Context(), second.Execution, second.Token, "second"); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Submit(t.Context(), second.Execution, second.Token, "second", submission(other, body)); !errors.Is(err, workerexchange.ErrQuota) {
		t.Fatal("simultaneous attempts exceeded their shared project staging quota", err)
	}
	finish := func(ref execution.Reference) {
		entry, row, err := ledger.Get(t.Context(), ref.Run, ref.ID)
		if err != nil {
			t.Fatal(err)
		}
		entry.ResultSaved, entry.EvidenceAbandoned = true, true
		update, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
		if err := f.store.Commit(t.Context(), update); err != nil {
			t.Fatal(err)
		}
		if err := f.service.Cleanup(t.Context(), ref); err != nil {
			t.Fatal(err)
		}
	}
	finish(f.ref)
	if err := f.service.Submit(t.Context(), second.Execution, second.Token, "second", submission(other, body)); err != nil {
		t.Fatal("cleanup did not release the project staging reservation", err)
	}
	finish(other.ref)
	if _, err := f.store.Get(t.Context(), "worker-exchange-project-quota", f.ref.Project); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("completed attempts leaked their project reservation", err)
	}
}

func TestPrepareRefreshesAuthorityAfterAConcurrentRecordUpdate(t *testing.T) {
	f := fresh(t)
	original := f.service.Authorize
	updated := false
	f.service.Authorize = func(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
		access, err := original(ctx, ref)
		if err != nil {
			return access, err
		}
		if !updated {
			updated = true
			row := access.Records[0]
			change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, true)
			if err != nil {
				return access, err
			}
			if err := f.store.Commit(ctx, change); err != nil {
				return access, err
			}
		}
		return access, nil
	}
	ticket := f.prepare(t)
	if _, err := f.service.Input(t.Context(), ticket.Execution, ticket.Token, "worker"); err != nil {
		t.Fatal("concurrent authority update stranded the prepared attempt", err)
	}
}

func TestTimingFailureKeepsRequestedFailureOnlyEvidence(t *testing.T) {
	f := fresh(t)
	threshold := float64(1)
	f.input.TimingMode = "fail"
	f.input.SlowThresholdMs = &threshold
	ticket := f.prepare(t)
	if _, err := f.service.Input(t.Context(), ticket.Execution, ticket.Token, "worker"); err != nil {
		t.Fatal(err)
	}
	report := submission(f, []byte("image bytes"))
	report.Report.Attachments[0].Kind = "screenshot"
	report.Report.Attachments[0].ContentType = "image/png"
	if err := f.service.Submit(t.Context(), ticket.Execution, ticket.Token, "worker", report); err != nil {
		t.Fatal("timing failure lost failure-only evidence", err)
	}
}
