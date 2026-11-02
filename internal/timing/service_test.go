package timing

import (
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConcurrentSamplesAreAtomicAndResetFencesOldPredictions(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "timing.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := Service{Store: db}
	w, now := fixtureWindow()
	p, err := service.Prediction(t.Context(), w.Context, DefaultPolicy(), 10000, now)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := range 12 {
		wg.Go(func() {
			sample := Sample{Run: fmt.Sprintf("run-%d", i), DurationMs: 1000, At: now.Add(-time.Second)}
			for range 40 {
				changes, err := service.RecordChanges(t.Context(), p, sample)
				if err != nil {
					failures <- err
					return
				}
				err = db.Commit(t.Context(), changes...)
				if errors.Is(err, storage.ErrConflict) {
					continue
				}
				if err != nil {
					failures <- err
				}
				return
			}
			failures <- errors.New("too many conflicts")
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	current, err := service.Prediction(t.Context(), w.Context, DefaultPolicy(), 10000, now)
	if err != nil || current.Samples != 12 {
		t.Fatalf("lost samples: %+v %v", current, err)
	}
	changes, err := service.RecordChanges(t.Context(), p, Sample{Run: "run-0", DurationMs: 1000, At: now})
	if err != nil || len(changes) != 0 {
		t.Fatal("duplicate sample")
	}
	if err := service.Reset(t.Context(), "admin", "other", current.Key, "reason", current.Revision); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("cross project reset allowed")
	}
	if err := service.Reset(t.Context(), "admin", "p", current.Key, "  ", current.Revision); err == nil {
		t.Fatal("empty reset reason accepted")
	}
	if err := service.Reset(t.Context(), "admin", "p", current.Key, "New performance target", current.Revision); err != nil {
		t.Fatal(err)
	}
	if err := service.Reset(t.Context(), "admin", "p", current.Key, "stale", current.Revision); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale reset accepted")
	}
	changes, err = service.RecordChanges(t.Context(), p, Sample{Run: "queued-before-reset", DurationMs: 1000, At: now})
	if err != nil || len(changes) != 0 {
		t.Fatal("old queued run repopulated baseline")
	}
	reset, err := service.Prediction(t.Context(), w.Context, DefaultPolicy(), 10000, now)
	if err != nil || reset.Samples != 0 || reset.Generation != 2 {
		t.Fatalf("reset: %+v %v", reset, err)
	}
	audits, err := db.All(t.Context(), "audit", "p")
	if err != nil || len(audits) != 1 {
		t.Fatalf("reset audit: %d %v", len(audits), err)
	}
}
