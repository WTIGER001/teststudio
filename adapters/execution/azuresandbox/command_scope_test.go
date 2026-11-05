package azuresandbox

import (
	"context"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxForeignStopCannotCancelTheOwnedCommandConnection(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	ref := f.plan.proof.Reference
	entered, release, canceled := make(chan struct{}), make(chan struct{}), make(chan struct{}, 1)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	f.controller.commands = commandCall(func(ctx context.Context, _ *Plan, _ string) (CommandResult, error) {
		close(entered)
		select {
		case <-ctx.Done():
			canceled <- struct{}{}
			return CommandResult{}, hostedexecution.ErrUnavailable
		case <-release:
			if ctx.Err() != nil {
				canceled <- struct{}{}
				return CommandResult{}, hostedexecution.ErrUnavailable
			}
			return CommandResult{ExitCode: 0}, nil
		}
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
	foreign := ref
	foreign.Project = "unrelated-project"
	if err := f.controller.Stop(t.Context(), foreign); err != hostedexecution.ErrOwnership {
		t.Fatal("foreign stop was accepted", err)
	}
	unblock()
	f.controller.commandGroup.Wait()
	select {
	case <-canceled:
		t.Fatal("rejected foreign stop canceled the owned command")
	default:
	}
	d, _, _, err := f.controller.load(t.Context(), ref)
	if err != nil || d.CommandExitCode == nil || *d.CommandExitCode != 0 || d.CommandSuppressed {
		t.Fatal("rejected foreign stop changed completion", err)
	}
}
