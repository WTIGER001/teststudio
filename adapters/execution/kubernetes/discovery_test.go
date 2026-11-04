package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func TestDiscoveryPodUsesFixedOperationAndPreparationCheckpoint(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f := fresh(t, runner)
			f.mode = "running"
			checkpointed := false
			allowCleanup := func(context.Context, execution.Reference) error {
				if !checkpointed {
					return storage.ErrConflict
				}
				return nil
			}
			f.provider.discoveryCleanup = allowCleanup
			f.exchange.AuthorizeDiscoveryCleanup = allowCleanup
			inspector := &hostedexecution.DiscoveryInspector{Name: "kubernetes", Runner: runner}
			ref := inspector.Reference(execution.Reference{ID: "preparation-attempt", Installation: "preparation-test", Project: "project", Run: "source-job", Test: "catalog", Attempt: 1, Worker: "preparer"})
			input := workerprotocol.DiscoveryInput{Project: ref.Project, Suite: "checks", Runner: runner, Image: f.input.Artifact.Image, SourceDigest: "sha256:" + strings.Repeat("a", 64)}
			ticket, err := f.exchange.PrepareDiscovery(t.Context(), ref, input, time.Now().UTC().Add(30*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			job := hostedexecution.Job{Operation: "discover", Reference: ref, Image: input.Image, Runner: runner, ExchangeOrigin: f.provider.config.ExchangeOrigin, Ticket: ticket}
			if err := f.provider.Ensure(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			f.mu.Lock()
			data, _ := json.Marshal(f.value.Spec["containers"])
			f.mu.Unlock()
			var containers []struct{ Args []string }
			json.Unmarshal(data, &containers)
			if len(containers) != 1 || !slices.Contains(containers[0].Args, "--operation=discover") {
				t.Fatal("discovery Pod did not bind its fixed operation")
			}
			wrong := job
			wrong.Operation = ""
			if err := f.provider.Ensure(t.Context(), wrong); !errors.Is(err, hostedexecution.ErrConfiguration) {
				t.Fatal("execution mode reused a discovery reference", err)
			}
			if err := f.provider.Remove(t.Context(), ref); err == nil {
				t.Fatal("provider deleted a discovery resource before its preparation checkpoint")
			}
			_, row, err := f.provider.load(t.Context(), ref)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.store.CommitAll(t.Context(), storage.Change{Record: row, Delete: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := f.provider.Observe(t.Context(), ref); err != nil {
				t.Fatal("owned discovery Pod could not be adopted from its transport reference", err)
			}
			checkpointed = true
			if err := f.exchange.Revoke(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if err := f.provider.Stop(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if err := f.provider.Remove(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if err := f.exchange.Cleanup(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if err := f.provider.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) {
				t.Fatal("removed discovery was dispatched again", err)
			}
		})
	}
}
