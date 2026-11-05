package azuresandbox

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type commandCall func(context.Context, *Plan, string) (CommandResult, error)

func (f commandCall) Execute(ctx context.Context, plan *Plan, id string) (CommandResult, error) {
	return f(ctx, plan, id)
}

func TestSandboxCommandsRequireActivationAndPersistOneIndependentResult(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				f := newControllerFixture(t, runner, operation)
				var calls atomic.Int32
				entered, release := make(chan struct{}), make(chan struct{})
				var once sync.Once
				unblock := func() { once.Do(func() { close(release) }) }
				defer unblock()
				f.controller.commands = commandCall(func(ctx context.Context, plan *Plan, id string) (CommandResult, error) {
					calls.Add(1)
					close(entered)
					if id != "native-sandbox" || plan.commandHash() == "" {
						return CommandResult{}, hostedexecution.ErrOwnership
					}
					select {
					case <-release:
						return CommandResult{ExitCode: 0}, nil
					case <-ctx.Done():
						return CommandResult{}, hostedexecution.ErrUnavailable
					}
				})
				ref := f.plan.proof.Reference
				if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
					t.Fatal(err)
				}
				if err := f.controller.StartCommand(t.Context(), ref); err != hostedexecution.ErrUnavailable || calls.Load() != 0 {
					t.Fatal("unactivated work executed", err)
				}
				if err := f.exchange.Activate(t.Context(), ref); err != nil {
					t.Fatal(err)
				}
				var replicas sync.WaitGroup
				for range 8 {
					replicas.Go(func() {
						err := f.controller.StartCommand(t.Context(), ref)
						if err != nil && err != hostedexecution.ErrUnavailable {
							t.Errorf("unexpected command reservation error: %v", err)
						}
					})
				}
				replicas.Wait()
				select {
				case <-entered:
				case <-t.Context().Done():
					t.Fatal("reserved command never started")
				}
				unblock()
				f.controller.commandGroup.Wait()
				if calls.Load() != 1 {
					t.Fatal("replicas repeated the command")
				}
				status, err := f.controller.Status(t.Context(), ref)
				if err != nil || status.Phase == "succeeded" || status.ExitCode != nil {
					t.Fatal("exit receipt bypassed instance quiescence", err)
				}
				f.instanceState("Stopped")
				status, err = f.controller.Status(t.Context(), ref)
				if err != nil || status.Phase != "succeeded" || status.ExitCode == nil || *status.ExitCode != 0 || status.NativeCompletion || status.AdmissionPending {
					t.Fatal("independent exit and quiescence did not produce native completion", err)
				}
				row, err := f.store.Get(t.Context(), DispatchBucket, ref.ID)
				if err != nil || strings.Contains(string(row.Data), f.plan.token) {
					t.Fatal("completion journal exposed a ticket", err)
				}
				f.reopen()
				f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
					t.Fatal("recovery repeated an acknowledged command")
					return CommandResult{}, nil
				})
				status, err = f.controller.Status(t.Context(), ref)
				if err != nil || status.Phase != "succeeded" {
					t.Fatal("acknowledged completion did not survive restart", err)
				}
			})
		}
	}
}

func TestSandboxCommandUncertaintyAndCancellationCannotBecomePassingResults(t *testing.T) {
	for _, mode := range []string{"lost", "cancel", "nonzero"} {
		t.Run(mode, func(t *testing.T) {
			f := newControllerFixture(t, "go", "")
			ref := f.plan.proof.Reference
			var calls atomic.Int32
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			f.controller.commands = commandCall(func(ctx context.Context, _ *Plan, _ string) (CommandResult, error) {
				calls.Add(1)
				close(entered)
				if mode == "cancel" {
					<-ctx.Done()
					return CommandResult{ExitCode: 0}, nil
				}
				<-release
				if mode == "lost" {
					return CommandResult{}, hostedexecution.ErrUnavailable
				}
				return CommandResult{ExitCode: 2}, nil
			})
			if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
				t.Fatal(err)
			}
			if err := f.exchange.Activate(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if err := f.controller.StartCommand(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			<-entered
			var stopErr error
			if mode == "cancel" {
				stopErr = f.controller.Stop(t.Context(), ref)
				if stopErr != nil && stopErr != hostedexecution.ErrUnavailable {
					t.Fatal(stopErr)
				}
			} else {
				unblock()
			}
			f.controller.commandGroup.Wait()
			if stopErr != nil {
				if err := f.controller.Stop(t.Context(), ref); err != nil {
					t.Fatal("pending cancellation did not remain recoverable", err)
				}
			}
			f.instanceState("Stopped")
			status, err := f.controller.Status(t.Context(), ref)
			if err != nil || status.Phase == "succeeded" {
				t.Fatal("unconfirmed or canceled command became passing", err)
			}
			if mode == "nonzero" && (status.Phase != "failed" || status.ExitCode == nil || *status.ExitCode != 2) {
				t.Fatal("known nonzero command result disappeared")
			}
			f.reopen()
			f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) { calls.Add(1); return CommandResult{}, nil })
			if err := f.controller.StartCommand(t.Context(), ref); err != nil || calls.Load() != 1 {
				t.Fatal("recovery replayed an uncertain command", err)
			}
		})
	}
}

func TestSandboxCommandAuthorizationLossPreventsTheReservedMutation(t *testing.T) {
	f := newControllerFixture(t, "playwright", "")
	ref := f.plan.proof.Reference
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
		t.Fatal("revoked authority executed code")
		return CommandResult{}, nil
	})
	if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
		t.Fatal(err)
	}
	if err := f.exchange.Activate(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	f.denied.Store(true)
	if err := f.controller.StartCommand(t.Context(), ref); err != hostedexecution.ErrUnavailable {
		t.Fatal("fresh command authorization was bypassed", err)
	}
	d, _, _, err := f.controller.load(t.Context(), ref)
	if err != nil || d.CommandAttempted {
		t.Fatal("rejected authority consumed a command reservation", err)
	}
}

func TestSandboxStoppedIdleInstanceCannotStartCodeOrBecomeSuccessful(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	ref := f.plan.proof.Reference
	f.controller.commands = commandCall(func(context.Context, *Plan, string) (CommandResult, error) {
		t.Fatal("stopped idle guest executed code")
		return CommandResult{}, nil
	})
	if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
		t.Fatal(err)
	}
	if err := f.exchange.Activate(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	f.instanceState("Stopped")
	status, err := f.controller.Status(t.Context(), ref)
	if err != nil || status.Phase != "stopped" || status.ExitCode != nil || status.AdmissionPending {
		t.Fatal("idle termination became an executable or successful command", err)
	}
}
