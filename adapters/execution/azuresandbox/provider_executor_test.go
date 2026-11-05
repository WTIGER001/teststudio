package azuresandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type settlingInstances struct{ *instanceControl }

func (a settlingInstances) Stop(ctx context.Context, id string) error {
	if err := a.instanceControl.Stop(ctx, id); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	item := a.items[id]
	item.State = "Stopped"
	a.items[id] = item
	return nil
}

func TestSandboxProviderUsesSharedExecutorReportsEvidenceAndCleanup(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f := newControllerFixture(t, runner, "")
			source := fixtureImagePlan(t, f.plan)
			a := &imageControl{t: t, items: map[string]DiskImage{}}
			preparedImage(t, f, a, source.proof.Image)
			m, err := NewImageManager(ImageManagerOptions{Config: source.config, PullIdentity: source.proof.PullIdentity, Images: a, Exchange: f.exchange})
			if err != nil {
				t.Fatal(err)
			}
			blobs, err := filesystem.New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer blobs.Close()
			f.exchange.Blobs = blobs
			original := f.exchange.Authorize
			f.exchange.Authorize = func(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
				access, err := original(ctx, ref)
				access.ProjectBytes = 1 << 20
				access.RunBytes = 1 << 20
				return access, err
			}
			f.controller.instances = settlingInstances{f.api}
			var calls atomic.Int32
			body := []byte("TestStudio sandbox provider protocol log\n")
			sum := sha256.Sum256(body)
			f.controller.commands = commandCall(func(ctx context.Context, plan *Plan, id string) (CommandResult, error) {
				calls.Add(1)
				packet, err := f.exchange.Input(ctx, plan.proof.Reference.ID, f.plan.token, "protocol-worker")
				if err != nil {
					return CommandResult{}, err
				}
				var input execution.CodedInput
				if err := json.Unmarshal(packet.Input, &input); err != nil {
					return CommandResult{}, err
				}
				at := time.Now().UTC()
				if err := f.exchange.Started(ctx, packet.Execution, f.plan.token, "protocol-worker", at); err != nil {
					return CommandResult{}, err
				}
				report := execution.CodedReport{Schema: "teststudio." + runner + "/v1", TestID: input.Test.ID, Selector: input.Test.Selector, Result: execution.Result{TestID: input.Test.ID, State: "passed", StartedAt: at, FinishedAt: at.Add(4 * time.Millisecond), DurationMs: 4}, Attachments: []execution.Attachment{{Path: "provider.log", Filename: "provider.log", Kind: "log", ContentType: "text/plain"}}}
				if runner == "go" {
					report.Result.Go = &execution.GoObservation{Package: "./parcel", Test: "TestPackage", Toolchain: "go-protocol", OS: "linux", Architecture: "amd64"}
				}
				submission := workerexchange.Submission{Report: report, Files: []workerexchange.File{{Path: "provider.log", Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])}}}
				if err := f.exchange.Submit(ctx, packet.Execution, f.plan.token, "protocol-worker", submission); err != nil {
					return CommandResult{}, err
				}
				if err := f.exchange.Upload(ctx, packet.Execution, f.plan.token, "protocol-worker", "provider.log", bytes.NewReader(body)); err != nil {
					return CommandResult{}, err
				}
				return CommandResult{ExitCode: 0}, nil
			})
			p := fixtureProvider(t, f, m, source.proof.Image)
			row, err := f.store.Get(t.Context(), workerexchange.Bucket, f.plan.proof.Reference.ID)
			if err != nil {
				t.Fatal(err)
			}
			work, err := storage.Decode[workerexchange.Record](row)
			if err != nil {
				t.Fatal(err)
			}
			var input execution.CodedInput
			if err := json.Unmarshal(work.Input, &input); err != nil {
				t.Fatal(err)
			}
			ref := f.plan.proof.Reference
			input.Reference = &ref
			input.BeforeDispatch = func() error { return nil }
			var started atomic.Bool
			input.OnStart = func(time.Time) error { started.Store(true); return nil }
			input.BeforeCleanup = func(report execution.CodedReport) error {
				ledger := workerstate.Ledger{Store: f.store}
				value, row, err := ledger.Get(t.Context(), ref.Run, ref.ID)
				if err != nil {
					return err
				}
				value.RecoveredResult = &report.Result
				change, err := ledger.Checkpoint(row, value, report.EvidenceErrors, false)
				if err != nil {
					return err
				}
				return f.store.Commit(t.Context(), change)
			}
			input.AfterCleanup = func(err error) error {
				return (workerstate.Ledger{Store: f.store}).Finish(t.Context(), ref.Run, ref.ID, "", err)
			}
			driver := hostedexecution.Executor{Provider: p, Exchange: f.exchange, Name: Location, Runner: runner, ExchangeOrigin: source.config.ExchangeOrigin, ApproveArtifact: func(context.Context, catalog.Artifact) error { return nil }, PollInterval: time.Millisecond}
			ctx, cancel := context.WithDeadline(t.Context(), f.plan.proof.Deadline)
			defer cancel()
			report := driver.Execute(ctx, input)
			f.controller.commandGroup.Wait()
			if report.Result.State != "passed" || report.Result.DurationMs != 4 || report.Result.DurationUnavailable || report.RecoveryError != nil || !started.Load() || calls.Load() != 1 || a.imports != 0 || len(report.Attachments) != 1 || !bytes.Equal(report.Attachments[0].Data, body) {
				t.Fatalf("shared execution lost its outcome, evidence, or lifecycle: %+v calls=%d", report, calls.Load())
			}
			refs, err := driver.Inventory(t.Context(), ref.Installation)
			if err != nil || len(refs) != 0 {
				t.Fatal("completed shared execution remains active", err)
			}
			if _, err := f.store.Get(t.Context(), workerexchange.Bucket, ref.ID); err != storage.ErrNotFound {
				t.Fatal("checkpointed exchange was retained", err)
			}
			ledger, _, err := (workerstate.Ledger{Store: f.store}).Get(t.Context(), ref.Run, ref.ID)
			if err != nil || ledger.State != "cleaned" || !ledger.EvidenceComplete {
				t.Fatal("shared result cleanup did not finish", err)
			}
		})
	}
}
