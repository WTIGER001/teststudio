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

type holdUpload struct {
	artifacts.Store
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (h *holdUpload) Put(ctx context.Context, key string, source io.Reader, limit int64) (artifacts.Object, error) {
	if h.calls.Add(1) == 1 {
		close(h.started)
	}
	select {
	case <-h.release:
		return h.Store.Put(ctx, key, source, limit)
	case <-ctx.Done():
		return artifacts.Object{}, ctx.Err()
	}
}

func TestUploadLeasePreventsDuplicateWritersAndFencesLateCompletion(t *testing.T) {
	s, files := fixture(t)
	seedFinished(t, s)
	blobs := &holdUpload{Store: s.Blobs, started: make(chan struct{}), release: make(chan struct{})}
	s.Blobs = blobs
	a := evidence.Artifact{ID: "artifact", Project: "project", Run: "run", Kind: "log", Filename: "result.log", ContentType: "text/plain", ExpiresAt: time.Now().Add(-time.Hour)}
	finished := make(chan error, 1)
	go func() { _, err := s.SaveBytes(t.Context(), a, []byte("evidence"), 100); finished <- err }()
	select {
	case <-blobs.started:
	case <-time.After(3 * time.Second):
		t.Fatal("upload did not start")
	}
	duplicate, err := s.SaveBytes(t.Context(), a, []byte("evidence"), 100)
	if err != nil || duplicate.State != "pending" || blobs.calls.Load() != 1 {
		t.Fatalf("duplicate upload: %+v %v", duplicate, err)
	}
	if err := s.Sweep(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	assertQuota(t, s, "artifact-quotas", "run", 8)
	r, err := s.Store.Get(t.Context(), evidence.Bucket("run"), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RequestDelete(t.Context(), "admin", "project", "run", a.ID, "Remove", r.Revision); err == nil {
		t.Fatal("manual deletion raced an active upload")
	}
	current, err := storage.Decode[evidence.Artifact](r)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Second)
	current.UploadUntil = &past
	change, err := storage.Put(r.Bucket, r.ID, "project", r.Revision, current)
	if err == nil {
		err = s.Store.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Sweep(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	close(blobs.release)
	if err := <-finished; !errors.Is(err, evidence.ErrUnavailable) {
		t.Fatalf("late upload revived expired content: %v", err)
	}
	assertQuota(t, s, "artifact-quotas", "run", 0)
	if _, err := files.Open(t.Context(), current.Project+"/"+current.Run+"/"+current.ID+"/"+current.Checksum); !errors.Is(err, artifacts.ErrNotFound) {
		t.Fatalf("late upload leaked content: %v", err)
	}
}
