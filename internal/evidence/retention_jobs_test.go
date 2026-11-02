package evidence_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestRetentionJobsResumePagesAndApplyObservedFailureDurations(t *testing.T) {
	s, _ := fixture(t)
	policy := capture.DefaultPolicy()
	policy.RetentionDays = 3
	policy.FailureRetentionDays = 90
	for i := 1; i <= 45; i++ {
		if _, err := s.LogSegment(t.Context(), evidence.LogInput{Project: "project", Run: "run", Worker: "worker", Stream: "stdout", Test: "TEST-1", Attempt: 1, Sequence: int64(i), Text: fmt.Sprintf("line %d", i), Policy: policy, Budget: evidence.Budget{RunBytes: 1 << 20, TestBytes: 1 << 20, FileBytes: 64 << 10, ProjectBytes: 10 << 20}}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	run, _ := storage.Put("runs", "run", "project", 0, map[string]any{"projectId": "project", "finishedAt": now, "outcome": "passed"})
	attempt, _ := storage.Put("attempts/run", "TEST-1.0001", "project", 0, map[string]string{"state": "failed"})
	job, _ := storage.Put("evidence-retention-jobs", "run", "project", 0, evidence.RetentionJob{Project: "project", Run: "run"})
	if err := s.Store.Commit(t.Context(), run, attempt, job); err != nil {
		t.Fatal(err)
	}
	if err := s.ProcessRetention(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	row, err := s.Store.Get(t.Context(), "evidence-retention-jobs", "run")
	if err != nil {
		t.Fatal("retention finished before the second page")
	}
	queued, err := storage.Decode[evidence.RetentionJob](row)
	if err != nil || queued.Cursor == "" {
		t.Fatalf("retention cursor missing: %+v %v", queued, err)
	}
	restarted := &evidence.Service{Store: s.Store, Blobs: s.Blobs}
	if err := restarted.ProcessRetention(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.Get(t.Context(), "evidence-retention-jobs", "run"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("finished retention job remains queued")
	}
	rows, err := s.Store.All(t.Context(), evidence.Bucket("run"), "project")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		a, err := storage.Decode[evidence.Artifact](row)
		if err != nil || !a.ExpiresAt.Equal(a.CreatedAt.Add(90*24*time.Hour)) {
			t.Fatalf("failure retention: %+v %v", a, err)
		}
	}
	statusRow, err := s.Store.Get(t.Context(), "evidence-retention-status", "run")
	if err != nil {
		t.Fatal(err)
	}
	status, err := storage.Decode[evidence.RetentionStatus](statusRow)
	if err != nil || status.State != "ready" {
		t.Fatalf("retention state: %+v %v", status, err)
	}
}

func TestDelayedRetentionJobDoesNotStarveTheNextRun(t *testing.T) {
	s, _ := fixture(t)
	delayed, _ := storage.Put("evidence-retention-jobs", "a", "project", 0, evidence.RetentionJob{Project: "project", Run: "a", NextAttempt: time.Now().Add(time.Hour)})
	due, _ := storage.Put("evidence-retention-jobs", "b", "project", 0, evidence.RetentionJob{Project: "project", Run: "b"})
	run, _ := storage.Put("runs", "b", "project", 0, map[string]any{"projectId": "project", "finishedAt": time.Now().UTC(), "outcome": "passed"})
	if err := s.Store.Commit(t.Context(), delayed, due, run); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.ProcessRetention(t.Context(), 1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Store.Get(t.Context(), "evidence-retention-jobs", "b"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("delayed job starved a ready job")
	}
}
