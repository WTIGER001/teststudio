package azuresandbox

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestSandboxProviderResumesPendingDiscoveryThroughCommandAndCleanup(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f, m, a, job, source := newImageFixture(t, runner)
			var commands atomic.Int32
			command := commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
				commands.Add(1)
				return CommandResult{ExitCode: 0}, nil
			})
			f.controller.commands = command
			p := fixtureProvider(t, f, m, source.proof.Image)
			if err := p.Ensure(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			status, err := p.Observe(t.Context(), job.Reference)
			if err != nil || !status.AdmissionPending || status.Phase != "pending" || commands.Load() != 0 || f.api.createCalls != 0 {
				t.Fatal("pending conversion admitted code", status, err)
			}
			refs, err := p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
			if err != nil || len(refs) != 1 || refs[0] != job.Reference {
				t.Fatal("pending source disappeared from recovery inventory", err)
			}
			f.reopen()
			f.controller.commands = command
			p = fixtureProvider(t, f, m, source.proof.Image)
			a.items["disk-1"] = imageSnapshot(t, source, "disk-1", "Ready")
			status, err = p.Observe(t.Context(), job.Reference)
			if err != nil || status.AdmissionPending || commands.Load() != 0 || f.api.createCalls != 1 || a.imports != 1 {
				t.Fatal("resumed conversion failed admission or repeated work", status, err)
			}
			if err := f.exchange.Activate(t.Context(), job.Reference); err != nil {
				t.Fatal(err)
			}
			if _, err := p.Observe(t.Context(), job.Reference); err != nil {
				t.Fatal(err)
			}
			f.controller.commandGroup.Wait()
			f.instanceState("Stopped")
			status, err = p.Observe(t.Context(), job.Reference)
			if err != nil || status.Phase != "succeeded" || status.ExitCode == nil || *status.ExitCode != 0 || status.NativeCompletion || commands.Load() != 1 {
				t.Fatal("independent completion was lost", status, err)
			}
			f.retainCheckpoint()
			if err := p.Remove(t.Context(), job.Reference); err != nil {
				t.Fatal(err)
			}
			if err := f.exchange.Cleanup(t.Context(), job.Reference); err != nil {
				t.Fatal(err)
			}
			status, err = p.Observe(t.Context(), job.Reference)
			if err != nil || status.Phase != "succeeded" {
				t.Fatal("cleaned completion depends on deleted ticket", err)
			}
			refs, err = p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
			if err != nil || len(refs) != 0 || a.imports != 1 {
				t.Fatal("cleaned attempt remains active or deleted shared source", err)
			}
			for _, bucket := range []string{IntentBucket, DispatchBucket} {
				row, err := f.store.Get(t.Context(), bucket, job.Reference.ID)
				if err != nil || strings.Contains(string(row.Data), job.Ticket.Token) {
					t.Fatal("durable launch contains a credential", err)
				}
			}
		})
	}
}

func TestSandboxProviderCancellationFencesAnInFlightCreate(t *testing.T) {
	f, m, a, job, source := newImageFixture(t, "go")
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
		t.Error("canceled launch started code")
		return CommandResult{}, nil
	})
	p := fixtureProvider(t, f, m, source.proof.Image)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	a.items["disk-1"] = imageSnapshot(t, source, "disk-1", "Ready")
	f.api.createEntered = make(chan struct{})
	f.api.releaseCreate = make(chan struct{})
	var release sync.Once
	unblock := func() { release.Do(func() { close(f.api.releaseCreate) }) }
	done := make(chan error, 1)
	joined := make(chan struct{})
	go func() { defer close(joined); _, err := p.Observe(t.Context(), job.Reference); done <- err }()
	defer func() { unblock(); <-joined }()
	select {
	case <-f.api.createEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("create was not reached")
	}
	if err := p.Stop(t.Context(), job.Reference); err != hostedexecution.ErrUnavailable {
		t.Fatal("unacknowledged creation became absent", err)
	}
	unblock()
	if err := <-done; err != nil && err != hostedexecution.ErrUnavailable {
		t.Fatal(err)
	}
	if _, err := p.Observe(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.instanceState("Stopped")
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || status.Phase != "stopped" || status.ExitCode != nil || status.AdmissionPending || f.api.createCalls != 1 {
		t.Fatal("canceled in-flight resource became executable", status, err)
	}
	f.retainCheckpoint()
	if err := p.Remove(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestSandboxProviderBindsTheImportIdentityAcrossRestart(t *testing.T) {
	f, m, _, job, source := newImageFixture(t, "playwright")
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) { return CommandResult{}, nil })
	p := fixtureProvider(t, f, m, source.proof.Image)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	m.identity = strings.Replace(m.identity, "studio-pull", "other-pull", 1)
	if _, err := p.Observe(t.Context(), job.Reference); err != hostedexecution.ErrOwnership {
		t.Fatal("changed import identity adopted pending work", err)
	}
}

func TestSandboxProviderCancelsPendingConversionAndPreservesScope(t *testing.T) {
	f, m, a, job, source := newImageFixture(t, "go")
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
		t.Error("canceled conversion executed")
		return CommandResult{}, nil
	})
	p := fixtureProvider(t, f, m, source.proof.Image)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	foreign := job.Reference
	foreign.Project = "another-project"
	if err := p.Stop(t.Context(), foreign); err != hostedexecution.ErrOwnership {
		t.Fatal("foreign project changed launch", err)
	}
	state, err := f.exchange.State(t.Context(), job.Reference)
	if err != nil || state.Revoked {
		t.Fatal("foreign stop revoked rightful work", err)
	}
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.reopen()
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
		t.Error("restarted cancellation executed")
		return CommandResult{}, nil
	})
	p = fixtureProvider(t, f, m, source.proof.Image)
	a.items["disk-1"] = imageSnapshot(t, source, "disk-1", "Ready")
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || status.Phase != "stopped" || status.AdmissionPending || status.ExitCode != nil || f.api.createCalls != 0 || a.imports != 1 {
		t.Fatal("canceled pending launch reappeared", status, err)
	}
	if err := p.Remove(t.Context(), job.Reference); err != hostedexecution.ErrUnavailable {
		t.Fatal("cleanup bypassed canonical checkpoint", err)
	}
	f.retainCheckpoint()
	if err := p.Remove(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestSandboxProviderRecoversLostSandboxReceiptWithoutDuplicateLaunch(t *testing.T) {
	f, m, a, job, source := newImageFixture(t, "playwright")
	f.api.lostCreate = true
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) { return CommandResult{ExitCode: 0}, nil })
	p := fixtureProvider(t, f, m, source.proof.Image)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	a.items["disk-1"] = imageSnapshot(t, source, "disk-1", "Ready")
	var replicas sync.WaitGroup
	for range 8 {
		replicas.Go(func() {
			_, err := p.Observe(t.Context(), job.Reference)
			if err != nil && err != hostedexecution.ErrUnavailable {
				t.Errorf("reconciliation failed: %v", err)
			}
		})
	}
	replicas.Wait()
	if f.api.createCalls != 1 || a.imports != 1 {
		t.Fatal("replicas duplicated resource creation")
	}
	f.reopen()
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) { return CommandResult{ExitCode: 0}, nil })
	p = fixtureProvider(t, f, m, source.proof.Image)
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || status.AdmissionPending || f.api.createCalls != 1 {
		t.Fatal("lost receipt cannot recover exact resource", err)
	}
	row, err := f.store.Get(t.Context(), IntentBucket, job.Reference.ID)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := storage.Decode[launchIntent](row)
	d.Proof.InputHash = hash("changed immutable work")
	d.Proof.Fingerprint = p.fingerprint(d.Proof)
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, d)
	if err := f.store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Observe(t.Context(), job.Reference); err != hostedexecution.ErrOwnership {
		t.Fatal("changed input was accepted", err)
	}
}
