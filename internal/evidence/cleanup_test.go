package evidence_test

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
)

func expiredArtifact(t *testing.T, s *evidence.Service) evidence.Artifact {
	t.Helper()
	a, err := s.SaveBytes(t.Context(), evidence.Artifact{ID: "artifact", Project: "project", Run: "run", Test: "TEST-1", Kind: "attachment", Filename: "result.txt", ContentType: "text/plain", ExpiresAt: time.Now().Add(-time.Hour)}, []byte("evidence"), 100)
	if err != nil {
		t.Fatal(err)
	}
	seedFinished(t, s)
	return a
}

func seedFinished(t *testing.T, s *evidence.Service) {
	t.Helper()
	r, err := storage.Put("runs", "run", "project", 0, map[string]any{"projectId": "project", "finishedAt": time.Now().UTC(), "outcome": "failed", "counts": map[string]int{"failed": 1}})
	if err == nil {
		err = s.Store.Commit(t.Context(), r)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func assertQuota(t *testing.T, s *evidence.Service, bucket, id string, bytes int64) {
	t.Helper()
	r, err := s.Store.Get(t.Context(), bucket, id)
	if err != nil {
		t.Fatal(err)
	}
	q, err := storage.Decode[evidence.Quota](r)
	if err != nil || q.Bytes != bytes {
		t.Fatalf("%s/%s reserved %d, expected %d: %v", bucket, id, q.Bytes, bytes, err)
	}
}

func TestPinsPreserveContentAndExpiryReleasesEveryQuotaOnce(t *testing.T) {
	s, files := fixture(t)
	a := expiredArtifact(t, s)
	if err := s.SavePin(t.Context(), "admin", "project", "run", true, nil, "Investigate the failed run", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Sweep(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	_, reader, err := s.Open(t.Context(), "project", "run", a.ID)
	if err != nil {
		t.Fatal("pinned content became unavailable", err)
	}
	body, _ := io.ReadAll(reader)
	reader.Close()
	if string(body) != "evidence" {
		t.Fatal("pinned content changed")
	}
	if err := s.SavePin(t.Context(), "admin", "project", "run", false, nil, "Investigation complete", 1); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Sweep(t.Context(), 10); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range [][2]string{{"artifact-quotas", "run"}, {"artifact-project-quotas", "project"}, {"artifact-test-quotas", "run/TEST-1"}} {
		assertQuota(t, s, scope[0], scope[1], 0)
	}
	if _, err := files.Open(t.Context(), a.Project+"/"+a.Run+"/"+a.ID+"/"+a.Checksum); !errors.Is(err, artifacts.ErrNotFound) {
		t.Fatalf("expired bytes remain: %v", err)
	}
	r, err := s.Store.Get(t.Context(), evidence.Bucket("run"), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := storage.Decode[evidence.Artifact](r)
	if err != nil || expired.State != "expired" || expired.DeletedAt == nil {
		t.Fatalf("retained manifest: %+v %v", expired, err)
	}
	r, err = s.Store.Get(t.Context(), "runs", "run")
	if err != nil || r.Revision != 1 {
		t.Fatal("cleanup changed compact test results")
	}
}

type failDeletion struct {
	artifacts.Store
	fail atomic.Bool
}

func (f *failDeletion) Delete(ctx context.Context, key string) error {
	if f.fail.Load() {
		return errors.New("simulated deletion outage")
	}
	return f.Store.Delete(ctx, key)
}

func retryDeletionNow(t *testing.T, s *evidence.Service) {
	t.Helper()
	rows, err := s.Store.All(t.Context(), "artifact-expirations", "project")
	if err != nil || len(rows) != 1 {
		t.Fatalf("deletion index: %d %v", len(rows), err)
	}
	e, err := storage.Decode[evidence.Expiration](rows[0])
	if err != nil {
		t.Fatal(err)
	}
	e.NextAttempt = time.Now().Add(-time.Second)
	change, err := storage.Put(rows[0].Bucket, rows[0].ID, "project", rows[0].Revision, e)
	if err == nil {
		err = s.Store.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeletionFailureRemainsVisibleAndRetriesAfterRestart(t *testing.T) {
	s, _ := fixture(t)
	a := expiredArtifact(t, s)
	blobs := &failDeletion{Store: s.Blobs}
	blobs.fail.Store(true)
	s.Blobs = blobs
	if err := s.Sweep(t.Context(), 10); err == nil {
		t.Fatal("deletion outage was not reported")
	}
	assertQuota(t, s, "artifact-quotas", "run", a.Size)
	r, err := s.Store.Get(t.Context(), evidence.Bucket("run"), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := storage.Decode[evidence.Artifact](r)
	if err != nil || failed.State != "cleanup-failed" || failed.Error == "" {
		t.Fatalf("failed cleanup hidden: %+v %v", failed, err)
	}
	p, _, err := s.ReadPin(t.Context(), "project", "run")
	if err != nil || p.InFlight != 0 {
		t.Fatalf("failed deletion retained a lease: %+v %v", p, err)
	}
	blobs.fail.Store(false)
	retryDeletionNow(t, s)
	restarted := &evidence.Service{Store: s.Store, Blobs: blobs}
	if err := restarted.Sweep(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	assertQuota(t, s, "artifact-project-quotas", "project", 0)
}

type holdDeletion struct {
	artifacts.Store
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (h *holdDeletion) Delete(ctx context.Context, key string) error {
	if h.calls.Add(1) == 1 {
		close(h.started)
	}
	select {
	case <-h.release:
		return h.Store.Delete(ctx, key)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestCompetingCleanupClaimsOnceAndBlocksPinsDuringDeletion(t *testing.T) {
	s, _ := fixture(t)
	expiredArtifact(t, s)
	blobs := &holdDeletion{Store: s.Blobs, started: make(chan struct{}), release: make(chan struct{})}
	s.Blobs = blobs
	first := make(chan error, 1)
	go func() { first <- s.Sweep(t.Context(), 10) }()
	select {
	case <-blobs.started:
	case <-time.After(3 * time.Second):
		t.Fatal("deletion did not start")
	}
	other := &evidence.Service{Store: s.Store, Blobs: blobs}
	if err := other.Sweep(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	p, r, err := s.ReadPin(t.Context(), "project", "run")
	if err != nil || p.InFlight != 1 {
		t.Fatalf("claim reservation: %+v %v", p, err)
	}
	if err := s.SavePin(t.Context(), "admin", "project", "run", true, nil, "Too late", r.Revision); err == nil {
		t.Fatal("pin raced an active storage deletion")
	}
	close(blobs.release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if blobs.calls.Load() != 1 {
		t.Fatalf("competing cleaners deleted %d times", blobs.calls.Load())
	}
}
