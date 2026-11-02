package storage_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestRevisionAssertionsProtectTransactionsWithoutChangingRecords(t *testing.T) {
	s, err := testkit.Open(t, filepath.Join(t.TempDir(), "assert.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	entry, _ := storage.Put("objects", "one", "project", 0, "original")
	if err := s.Commit(t.Context(), entry); err != nil {
		t.Fatal(err)
	}
	original, err := s.Get(t.Context(), "objects", "one")
	if err != nil {
		t.Fatal(err)
	}
	guard := storage.Change{Record: original, Assert: true}
	other, _ := storage.Put("objects", "two", "project", 0, "second")
	if err := s.Commit(t.Context(), guard, other); err != nil {
		t.Fatal(err)
	}
	after, err := s.Get(t.Context(), "objects", "one")
	if err != nil || after.Revision != original.Revision || !after.Updated.Equal(original.Updated) {
		t.Fatalf("assert changed the record: %+v %v", after, err)
	}
	changed, _ := storage.Put("objects", "one", "project", original.Revision, "changed")
	if err := s.Commit(t.Context(), changed); err != nil {
		t.Fatal(err)
	}
	third, _ := storage.Put("objects", "three", "project", 0, "third")
	if err := s.Commit(t.Context(), guard, third); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale assert: %v", err)
	}
	if _, err := s.Get(t.Context(), "objects", "three"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("assert failure leaked a write")
	}
}
