package hostedexecution_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type fixture struct {
	t               *testing.T
	driver          *hostedexecution.Executor
	exchange        *workerexchange.Service
	store           *storage.Store
	input           execution.CodedInput
	provider        *provider
	events          []string
	checkpointError bool
	cleanupError    error
}

type provider struct {
	f             *fixture
	job           hostedexecution.Job
	status        *hostedexecution.Status
	mode          string
	ensureCount   int
	removeCount   int
	inspectCount  int
	removeError   bool
	stopError     bool
	startCallback func()
	inventory     []execution.Reference
}

func fresh(t *testing.T, runner string) *fixture {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := filesystem.New(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close(); db.Close() })
	f := &fixture{t: t, store: db}
	p := &provider{f: f, mode: "passed"}
	f.provider = p
	f.exchange = &workerexchange.Service{Store: db, Blobs: blobs, Authorize: func(context.Context, execution.Reference) (workerstate.Access, error) {
		return workerstate.Access{Deadline: time.Now().Add(time.Minute), ProjectBytes: 1 << 30, RunBytes: 1 << 30}, nil
	}}
	f.driver = &hostedexecution.Executor{Provider: p, Exchange: f.exchange, Name: "fixture", Runner: runner, ExchangeOrigin: "https://jobs.example.test", ApproveArtifact: func(context.Context, catalog.Artifact) error { return nil }, PollInterval: time.Millisecond}
	ref := f.driver.Reference(execution.Reference{ID: "attempt-1", Installation: "installation", Project: "project", Run: "run", Test: "TEST-1", Attempt: 1, Worker: "owner"})
	selected := "@TEST-1"
	kind := "browser"
	if runner == "go" {
		selected = "./checks::TestOne"
		kind = "integration"
	}
	f.input = execution.CodedInput{Reference: &ref, RunID: ref.Run, Test: catalog.Test{ID: ref.Test, Runner: runner, Selector: selected, Type: kind}, Artifact: catalog.Artifact{Image: "registry.example.test/suite@sha256:" + strings.Repeat("a", 64), SourceProject: ref.Project, Discovered: map[string]string{ref.Test: selected}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000}
	_, changes, err := (workerstate.Ledger{Store: db}).Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	f.input.BeforeDispatch = func() error { f.events = append(f.events, "authorize"); return nil }
	f.input.OnStart = func(time.Time) error { f.events = append(f.events, "started"); return nil }
	f.input.BeforeCleanup = func(report execution.CodedReport) error {
		f.events = append(f.events, "checkpoint")
		if p.job.Ticket.Token != "" {
			if _, err := f.exchange.Input(context.WithoutCancel(t.Context()), ref.ID, p.job.Ticket.Token, "worker"); !errors.Is(err, workerexchange.ErrDenied) {
				t.Fatal("checkpoint preceded ticket revocation", err)
			}
		}
		if f.checkpointError {
			return errors.New("fixture checkpoint unavailable")
		}
		ledger := workerstate.Ledger{Store: db}
		record, row, err := ledger.Get(t.Context(), ref.Run, ref.ID)
		if err != nil {
			return err
		}
		record.ResultSaved = true
		record.EvidenceComplete = len(report.EvidenceErrors) == 0
		record.EvidenceAbandoned = !record.EvidenceComplete && !report.EvidencePending
		record.RecoveredResult = &report.Result
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
		if err != nil {
			return err
		}
		if err := db.Commit(t.Context(), change); err != nil {
			return err
		}
		if report.EvidencePending {
			return errors.New("private evidence is pending recovery")
		}
		return nil
	}
	f.input.AfterCleanup = func(err error) error {
		f.events = append(f.events, "finished")
		f.cleanupError = err
		return (workerstate.Ledger{Store: db}).Finish(t.Context(), ref.Run, ref.ID, "", err)
	}
	return f
}

func (p *provider) InspectImage(_ context.Context, image string) (hostedexecution.Image, error) {
	p.inspectCount++
	return hostedexecution.Image{Reference: image, Runner: p.f.driver.Runner, Runtime: p.f.driver.Runner + "-fixture-v1", OS: "linux", Architecture: "arm64", Supervisor: "worker-exchange/v1", Project: "project"}, nil
}

func (p *provider) Ensure(ctx context.Context, job hostedexecution.Job) error {
	p.ensureCount++
	p.f.events = append(p.f.events, "ensure")
	p.job = job
	if p.mode == "rejected" {
		return hostedexecution.ErrRejected
	}
	p.status = &hostedexecution.Status{Reference: job.Reference, Image: job.Image, Deadline: job.Ticket.Deadline, Phase: "running"}
	if p.mode == "wrong-image" {
		p.status.Image = "registry.example.test/other@sha256:" + strings.Repeat("b", 64)
	}
	if p.startCallback != nil {
		p.startCallback()
	}
	if p.mode == "lost-ack" {
		return errors.New("private provider response with credential-canary")
	}
	return nil
}

func (p *provider) publish(ctx context.Context) {
	t := p.f.t
	input := p.f.input
	ticket := p.job.Ticket
	if _, err := p.f.exchange.Input(ctx, ticket.Execution, ticket.Token, "worker"); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	if err := p.f.exchange.Started(ctx, ticket.Execution, ticket.Token, "worker", at); err != nil {
		t.Fatal(err)
	}
	state := "passed"
	failure := ""
	exit := 0
	if p.mode == "failed" {
		state = "failed"
		failure = "assertion"
		exit = 1
	}
	report := execution.CodedReport{Schema: "teststudio." + input.Test.Runner + "/v1", TestID: input.Test.ID, Selector: input.Test.Selector, Result: execution.Result{TestID: input.Test.ID, State: state, FailureKind: failure, StartedAt: at, FinishedAt: at.Add(time.Millisecond), DurationMs: 1}}
	if input.Test.Runner == "go" {
		report.Result.Go = &execution.GoObservation{Package: "./checks", Test: "TestOne", Toolchain: "go-fixture", OS: "linux", Architecture: "arm64"}
	}
	body := []byte("private fixture evidence")
	hash := sha256.Sum256(body)
	report.Attachments = []execution.Attachment{{Path: "run.log", Filename: "run.log", Kind: "log", ContentType: "text/plain"}}
	submission := workerexchange.Submission{ProcessExitCode: exit, Report: report, Files: []workerexchange.File{{Path: "run.log", Size: int64(len(body)), SHA256: hex.EncodeToString(hash[:])}}}
	if err := p.f.exchange.Submit(ctx, ticket.Execution, ticket.Token, "worker", submission); err != nil {
		t.Fatal(err)
	}
	if p.mode != "missing-upload" {
		if err := p.f.exchange.Upload(ctx, ticket.Execution, ticket.Token, "worker", "run.log", bytes.NewReader(body)); err != nil {
			t.Fatal(err)
		}
	}
	deliveryExit := 0
	p.status.Phase = "succeeded"
	p.status.ExitCode = &deliveryExit
	if p.mode == "provider-failed" {
		deliveryExit = 2
		p.status.Phase = "failed"
	}
}

func (p *provider) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	if ctx.Err() != nil {
		return hostedexecution.Status{}, ctx.Err()
	}
	if p.status == nil {
		return hostedexecution.Status{}, hostedexecution.ErrNotFound
	}
	if p.status.Phase == "running" && p.mode != "running" {
		state, err := p.f.exchange.State(ctx, ref)
		if err == nil && !state.DispatchPending && !state.Revoked {
			p.publish(ctx)
		}
	}
	return *p.status, nil
}
func (p *provider) Stop(ctx context.Context, ref execution.Reference) error {
	p.f.events = append(p.f.events, "stop")
	if p.stopError {
		return errors.New("private stop credential-canary")
	}
	if p.status == nil {
		return hostedexecution.ErrNotFound
	}
	if p.status.Reference != ref {
		return hostedexecution.ErrOwnership
	}
	if p.status.Phase == "running" || p.status.Phase == "pending" {
		p.status.Phase = "stopped"
		exit := 137
		p.status.ExitCode = &exit
	}
	return ctx.Err()
}
func (p *provider) Remove(ctx context.Context, ref execution.Reference) error {
	p.removeCount++
	p.f.events = append(p.f.events, "remove")
	if err := p.f.exchange.CheckCleanup(ctx, ref); err != nil {
		p.f.t.Fatal("provider resource deletion preceded checkpoint", err)
	}
	if p.removeError {
		return errors.New("private delete credential-canary")
	}
	p.status = nil
	return nil
}
func (p *provider) Inventory(context.Context, string, string) ([]execution.Reference, error) {
	return p.inventory, nil
}
