package storage_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestStoreTransactions(t *testing.T) {
	s, err := testkit.Open(t, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	bucket := t.Name()
	put, _ := storage.Put(bucket, "one", "p1", 0, map[string]string{"name": "first"})
	if err := s.Commit(t.Context(), put); err != nil {
		t.Fatal(err)
	}
	r, err := s.Get(t.Context(), bucket, "one")
	if err != nil || r.Revision != 1 {
		t.Fatalf("created record: %+v %v", r, err)
	}
	put, _ = storage.Put(bucket, "one", "p1", 1, map[string]string{"name": "second"})
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			err := s.Commit(context.Background(), put)
			if err == nil {
				wins.Add(1)
			} else if !errors.Is(err, storage.ErrConflict) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("expected one writer, got %d", wins.Load())
	}
	newRecord, _ := storage.Put(bucket, "two", "p1", 0, "new")
	if err := s.Commit(t.Context(), newRecord, put); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected transaction conflict: %v", err)
	}
	if _, err := s.Get(t.Context(), bucket, "two"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("failed transaction leaked: %v", err)
	}
}

func TestCursorScopeAndDurability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := testkit.Open(t, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b", "c"} {
		put, _ := storage.Put("catalog", id, "project", 0, id)
		if err := s.Commit(t.Context(), put); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.List(t.Context(), "catalog", "project", "", 2)
	if err != nil || len(page.Items) != 2 || page.Cursor == "" {
		t.Fatalf("first page: %+v %v", page, err)
	}
	if _, err := s.List(t.Context(), "catalog", "other", page.Cursor, 2); err == nil {
		t.Fatal("cursor crossed projects")
	}
	path = testkit.ConnectionString(t, s, path)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = storage.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	next, err := s.List(t.Context(), "catalog", "project", page.Cursor, 2)
	if err != nil || len(next.Items) != 1 || next.Items[0].ID != "c" {
		t.Fatalf("resumed page: %+v %v", next, err)
	}
	first, err := s.Next(t.Context(), "grant")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Next(t.Context(), "grant")
	if err != nil || second != first+1 {
		t.Fatalf("sequence: %d %d %v", first, second, err)
	}
}

func TestNewestPaginationAndEmptyDatabaseRejection(t *testing.T) {
	if s, err := storage.Open(t.Context(), ""); err == nil {
		s.Close()
		t.Fatal("empty configuration created an ephemeral database")
	}
	s, err := testkit.Open(t, filepath.Join(t.TempDir(), "newest.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, id := range []string{"a", "b", "c"} {
		c, _ := storage.Put("runs", id, "project", 0, id)
		if err := s.Commit(t.Context(), c); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.ListNewest(t.Context(), "runs", "project", "", 2)
	if err != nil || len(page.Items) != 2 || page.Items[0].ID != "c" || page.Items[1].ID != "b" {
		t.Fatalf("newest page: %+v %v", page, err)
	}
	if _, err := s.List(t.Context(), "runs", "project", page.Cursor, 2); err == nil {
		t.Fatal("cursor crossed sort directions")
	}
	next, err := s.ListNewest(t.Context(), "runs", "project", page.Cursor, 2)
	if err != nil || len(next.Items) != 1 || next.Items[0].ID != "a" {
		t.Fatalf("newest continuation: %+v %v", next, err)
	}
}

func TestConcurrentSchemaStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup.db")
	initial, err := testkit.Open(t, path)
	if err != nil {
		t.Fatal(err)
	}
	path = testkit.ConnectionString(t, initial, path)
	for _, table := range []string{"records", "schema_migrations", "counters"} {
		if _, err := initial.DB.ExecContext(t.Context(), "DROP TABLE "+table); err != nil {
			t.Fatal(err)
		}
	}
	initial.Close()
	var workers sync.WaitGroup
	for i := range 4 {
		workers.Go(func() {
			db, err := storage.Open(t.Context(), path)
			if err != nil {
				t.Error(err)
				return
			}
			defer db.Close()
			change, err := storage.Put("startup", fmt.Sprintf("writer-%d", i), "project", 0, i)
			if err == nil {
				err = db.Commit(t.Context(), change)
			}
			if err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
}
