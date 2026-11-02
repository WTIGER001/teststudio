package workerstate

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"path/filepath"
	"sync"
	"testing"
)

func TestInstallationIdentityPersistsAndConcurrentCreationAgrees(t *testing.T) {
	path := filepath.Join(t.TempDir(), "worker.db")
	db, err := testkit.Open(t, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ledger := Ledger{Store: db}
	if _, err := ledger.Installation(t.Context(), false); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("absent installation was created by a read")
	}
	ids := make(chan string, 8)
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			id, err := ledger.Installation(t.Context(), true)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- id
		})
	}
	group.Wait()
	close(ids)
	identity := ""
	for id := range ids {
		if identity == "" {
			identity = id
		}
		if id != identity {
			t.Fatal("replicas created different installation identities")
		}
	}
	if identity == "" {
		t.Fatal("no installation identity")
	}
	other, err := storage.Open(t.Context(), testkit.ConnectionString(t, db, path))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	reopened, err := (Ledger{Store: other}).Installation(t.Context(), false)
	if err != nil || reopened != identity {
		t.Fatal("installation identity changed on reopen")
	}
}

func TestCheckpointClaimAndRetryStateAreDurable(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "worker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ledger := Ledger{Store: db}
	ref := execution.Reference{ID: "execution", Project: "project", Run: "run", Test: "TEST-1", Worker: "worker", Installation: "installation", Attempt: 1}
	record, changes, err := ledger.Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	_, row, err := ledger.Get(t.Context(), ref.Run, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	record.Claim = "claim-token"
	change, err := ledger.Checkpoint(row, record, []string{"capture pending"}, true)
	if err == nil {
		err = db.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := ledger.Get(t.Context(), ref.Run, ref.ID)
	if err != nil || loaded.Claim != "claim-token" || loaded.EvidenceComplete || !loaded.ResultSaved {
		t.Fatalf("checkpoint: %+v %v", loaded, err)
	}
	if err := ledger.Finish(t.Context(), ref.Run, ref.ID, "stale", nil); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale claimant finalized worker state")
	}
	if err := ledger.Finish(t.Context(), ref.Run, ref.ID, "claim-token", errors.New("Docker unavailable")); err != nil {
		t.Fatal(err)
	}
	loaded, _, err = ledger.Get(t.Context(), ref.Run, ref.ID)
	if err != nil || loaded.State != "cleanup-failed" || loaded.Claim != "" || loaded.CleanupAttempts != 1 {
		t.Fatal("cleanup failure was not retained")
	}
	if _, err := db.Get(t.Context(), "execution-work", ref.ID); err != nil {
		t.Fatal("failed cleanup lost its queue entry")
	}
	if err := ledger.Finish(t.Context(), ref.Run, ref.ID, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Get(t.Context(), "execution-work", ref.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("completed cleanup kept its queue entry")
	}
}
