package localauth

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestPasswordLimiterPersistsAndSerializesUnknownNamesAcrossInstances(t *testing.T) {
	local := filepath.Join(t.TempDir(), "limiter.db")
	db, err := testkit.Open(t, local)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first, err := New(t.Context(), &sessions.Service{Store: db, Directory: &identity.Directory{Store: db}, LocalAccounts: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := storage.Open(t.Context(), testkit.ConnectionString(t, db, local))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	second, err := New(t.Context(), &sessions.Service{Store: other, Directory: &identity.Directory{Store: other}, LocalAccounts: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			service := first
			if i%2 == 1 {
				service = second
			}
			err := service.reserve(ctx, "unknown-account", "127.0.0.1")
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrLimited) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if successes.Load() != 10 {
		t.Fatal("limiter exceeded or lost reservations", successes.Load())
	}
	if err := second.reserve(t.Context(), "unknown-account", "127.0.0.1"); !errors.Is(err, ErrLimited) {
		t.Fatal("limiter state did not persist", err)
	}
	rows, err := db.All(t.Context(), "local-auth-attempts", "")
	if err != nil || len(rows) != 2 {
		t.Fatal("limiter stored user-supplied cardinality", len(rows), err)
	}
	for _, row := range rows {
		value, err := storage.Decode[counter](row)
		if err != nil {
			t.Fatal(err)
		}
		value.Until = time.Now().Add(-time.Second)
		c, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, value)
		if err := db.Commit(t.Context(), c); err != nil {
			t.Fatal(err)
		}
	}
	if err := first.reserve(t.Context(), "unknown-account", "127.0.0.1"); err != nil {
		t.Fatal("expired limiter did not reopen", err)
	}
}
