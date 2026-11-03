package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestBoundTransactionReadsWritesAndSequenceRollback(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "bound.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	failure := errors.New("cancel workflow")
	for _, rollback := range []bool{true, false} {
		err := db.Transaction(t.Context(), func(bound *storage.Store, tx *sql.Tx) error {
			c, _ := storage.Put("workflow", "one", "alpha", 0, "pending")
			if err := bound.Commit(t.Context(), c); err != nil {
				return err
			}
			row, err := bound.Get(t.Context(), "workflow", "one")
			if err != nil || row.Revision != 1 {
				t.Fatalf("read own write: %+v %v", row, err)
			}
			c, _ = storage.Put("workflow", "one", "alpha", row.Revision, "finished")
			if err := bound.Commit(t.Context(), c); err != nil {
				return err
			}
			page, err := bound.ListNewest(t.Context(), "workflow", "alpha", "", 1)
			if err != nil || len(page.Items) != 1 || page.Items[0].Revision != 2 {
				t.Fatalf("transaction page: %+v %v", page, err)
			}
			n, err := bound.Next(t.Context(), "workflow")
			if err != nil || n != 1 {
				t.Fatalf("transaction counter: %d %v", n, err)
			}
			var revision int64
			if err := tx.QueryRowContext(t.Context(), "SELECT revision FROM records WHERE bucket=$1 AND id=$2", "workflow", "one").Scan(&revision); err != nil || revision != 2 {
				t.Fatalf("shared transaction: %d %v", revision, err)
			}
			if rollback {
				return failure
			}
			return nil
		})
		if rollback {
			if !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if _, err := db.Get(t.Context(), "workflow", "one"); !errors.Is(err, storage.ErrNotFound) {
				t.Fatal("rolled back row survived", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	row, err := db.Get(t.Context(), "workflow", "one")
	if err != nil || row.Revision != 2 {
		t.Fatal(row, err)
	}
	n, err := db.Next(t.Context(), "workflow")
	if err != nil || n != 2 {
		t.Fatal("counter did not commit with row", n, err)
	}
}

func TestBoundTransactionCannotCommitPartialFailedChanges(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "failed.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c, _ := storage.Put("workflow", "existing", "", 0, "existing")
	if err := db.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(t.Context(), func(bound *storage.Store, _ *sql.Tx) error {
		pending, _ := storage.Put("workflow", "leak", "", 0, "must roll back")
		if err := bound.Commit(t.Context(), pending, c); !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
		if err := bound.Transaction(t.Context(), func(*storage.Store, *sql.Tx) error { t.Fatal("nested transaction ran"); return nil }); err == nil {
			t.Fatal("nested transaction accepted")
		}
		return nil
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatal("swallowed failure committed transaction", err)
	}
	if _, err := db.Get(context.Background(), "workflow", "leak"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("partial write survived", err)
	}
}
