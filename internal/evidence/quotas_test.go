package evidence_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestProjectQuotaSerializesCompetingRunsAndPreservesProjectRevision(t *testing.T) {
	s, _ := fixture(t)
	limits := capture.DefaultLimits()
	limits.ProjectBytes = 12
	p, err := storage.Put("projects", "project", "project", 0, projects.Project{ID: "project", Name: "Project", CaptureLimits: &limits})
	if err == nil {
		err = s.Store.Commit(t.Context(), p)
	}
	if err != nil {
		t.Fatal(err)
	}
	var admitted atomic.Int32
	var workers sync.WaitGroup
	for _, run := range []string{"run-a", "run-b"} {
		workers.Go(func() {
			_, err := s.Save(t.Context(), evidence.Artifact{ID: "log", Project: "project", Run: run, Kind: "log", Filename: "result.log", ContentType: "text/plain"}, []byte("12345678"), evidence.Budget{RunBytes: 100, TestBytes: 100, FileBytes: 100, ProjectBytes: 100})
			if err == nil {
				admitted.Add(1)
			} else if !errors.Is(err, evidence.ErrQuota) {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if admitted.Load() != 1 {
		t.Fatalf("project quota admitted %d competing runs", admitted.Load())
	}
	r, err := s.Store.Get(t.Context(), "artifact-project-quotas", "project")
	if err != nil {
		t.Fatal(err)
	}
	q, err := storage.Decode[evidence.Quota](r)
	if err != nil || q.Bytes != 8 || q.Limit != 12 {
		t.Fatalf("project accounting: %+v %v", q, err)
	}
	r, err = s.Store.Get(t.Context(), "projects", "project")
	if err != nil || r.Revision != 1 {
		t.Fatalf("quota guard changed project revision: %+v %v", r, err)
	}
}

func TestFileAndTestBudgetsDoNotReserveRejectedBytes(t *testing.T) {
	s, _ := fixture(t)
	budget := evidence.Budget{RunBytes: 100, TestBytes: 10, FileBytes: 8, ProjectBytes: 1000}
	a := evidence.Artifact{ID: "one", Project: "project", Run: "run", Test: "TEST-1", Kind: "attachment", Filename: "data.txt", ContentType: "text/plain"}
	if _, err := s.Save(t.Context(), a, []byte("123456789"), budget); !errors.Is(err, evidence.ErrQuota) {
		t.Fatalf("file cap: %v", err)
	}
	if _, err := s.Save(t.Context(), a, []byte("12345678"), budget); err != nil {
		t.Fatal(err)
	}
	a.ID = "two"
	if _, err := s.Save(t.Context(), a, []byte("1234"), budget); !errors.Is(err, evidence.ErrQuota) {
		t.Fatalf("test cap: %v", err)
	}
	a.Test = "TEST-2"
	if _, err := s.Save(t.Context(), a, []byte("1234"), budget); err != nil {
		t.Fatal(err)
	}
	r, err := s.Store.Get(t.Context(), "artifact-quotas", "run")
	if err != nil {
		t.Fatal(err)
	}
	q, err := storage.Decode[evidence.Quota](r)
	if err != nil || q.Bytes != 12 {
		t.Fatalf("rejected bytes were reserved: %+v %v", q, err)
	}
}
