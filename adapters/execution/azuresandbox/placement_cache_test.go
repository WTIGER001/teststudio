package azuresandbox

import (
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxRunningPlacementBoundsMetadataReuseAndRefreshesAfterFailure(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKPlacement(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	now := time.Now()
	reader.now = func() time.Time { return now }
	if err := reader.Check(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	var replicas sync.WaitGroup
	for range 12 {
		replicas.Go(func() {
			if err := reader.CheckRunning(t.Context(), s.Worker); err != nil {
				t.Error(err)
			}
		})
	}
	replicas.Wait()
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("live polling repeated successful initial metadata")
		}
	}
	now = now.Add(placementRefresh)
	for range 12 {
		replicas.Go(func() {
			if err := reader.CheckRunning(t.Context(), s.Worker); err != nil {
				t.Error(err)
			}
		})
	}
	replicas.Wait()
	if w.calls[s.Worker.groupID()] != 2 || w.calls[s.Placement.PullAssignment] != 1 {
		t.Fatal("expiry did not coalesce one live-only refresh")
	}
	w.status = 503
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("full metadata failure was hidden", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("failed admission left a successful live cache", err)
	}
	w.status = 0
	w.resources[s.Worker.connectionID()]["properties"].(map[string]any)["subnetId"] = "changed-subnet"
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed refresh was cached or changed placement admitted", err)
	}
}
