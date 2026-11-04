package evidence_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type interruptedManifest struct {
	artifacts.Store
	cancel  context.CancelFunc
	content string
}

func (b interruptedManifest) Put(ctx context.Context, key string, source io.Reader, limit int64) (artifacts.Object, error) {
	if b.content == "missing" {
		b.cancel()
		return artifacts.Object{}, context.Canceled
	}
	if b.content == "corrupt" {
		source = bytes.NewReader([]byte("wrong"))
	}
	object, err := b.Store.Put(ctx, key, source, limit)
	b.cancel()
	return object, err
}

type countedUploadStore struct {
	artifacts.Store
	puts atomic.Int32
}

func (b *countedUploadStore) Put(ctx context.Context, key string, source io.Reader, limit int64) (artifacts.Object, error) {
	b.puts.Add(1)
	return b.Store.Put(ctx, key, source, limit)
}

func interruptManifest(t *testing.T, s *evidence.Service, id, content string) evidence.Artifact {
	t.Helper()
	original := s.Blobs
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s.Blobs = interruptedManifest{Store: original, cancel: cancel, content: content}
	_, err := s.SaveBytes(ctx, evidence.Artifact{ID: id, Project: "project", Run: "run", Test: "TEST-1", Attempt: 1, Kind: "attachment", Filename: id + ".txt", ContentType: "text/plain"}, []byte("fresh recovery evidence"), 1<<20)
	s.Blobs = original
	if err == nil {
		t.Fatal("interrupted manifest unexpectedly committed")
	}
	row, err := s.Store.Get(t.Context(), evidence.Bucket("run"), id)
	if err != nil {
		t.Fatal(err)
	}
	a, err := storage.Decode[evidence.Artifact](row)
	if err != nil || a.State != "pending" {
		t.Fatal("pending manifest was lost", a, err)
	}
	return a
}

func setUploadDeadline(t *testing.T, s *evidence.Service, id string, deadline time.Time) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), evidence.Bucket("run"), id)
	if err != nil {
		t.Fatal(err)
	}
	a, err := storage.Decode[evidence.Artifact](row)
	if err != nil {
		t.Fatal(err)
	}
	a.UploadUntil = &deadline
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, a)
	if err == nil {
		err = s.Store.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestUploadRecoveryReopensStorageAndCommitsExistingBytesWithoutReupload(t *testing.T) {
	root := t.TempDir()
	database := filepath.Join(root, "restart.db")
	db, err := testkit.Open(t, database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := filesystem.New(filepath.Join(root, "files"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	s := &evidence.Service{Store: db, Blobs: files}
	a := interruptManifest(t, s, "retained", "complete")
	setUploadDeadline(t, s, a.ID, time.Now().Add(-time.Second))
	connection := testkit.ConnectionString(t, db, database)
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = storage.Open(t.Context(), connection)
	if err != nil {
		t.Fatal(err)
	}
	files, err = filesystem.New(filepath.Join(root, "files"))
	if err != nil {
		t.Fatal(err)
	}
	counted := &countedUploadStore{Store: files}
	restarted := &evidence.Service{Store: db, Blobs: counted}
	for range 2 {
		if err := restarted.RecoverUploads(t.Context(), 10); err != nil {
			t.Fatal(err)
		}
	}
	available, reader, err := restarted.Open(t.Context(), a.Project, a.Run, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(data) != "fresh recovery evidence" || available.Checksum != a.Checksum || available.CreatedAt != a.CreatedAt || counted.puts.Load() != 0 {
		t.Fatal("restart lost or rewrote captured bytes", available, string(data), err)
	}
	duplicate, err := restarted.SaveBytes(t.Context(), a, data, 1<<20)
	if err != nil || duplicate.State != "available" || counted.puts.Load() != 0 {
		t.Fatal("duplicate upload rewrote available content", duplicate, err)
	}
	rows, _ := db.All(t.Context(), evidence.Bucket(a.Run), a.Project)
	pending, _ := db.All(t.Context(), "artifact-upload-work", a.Project)
	if len(rows) != 1 || len(pending) != 0 {
		t.Fatal("recovery duplicated or retained completed work", len(rows), len(pending))
	}
	assertQuota(t, restarted, "artifact-quotas", a.Run, a.Size)
	assertQuota(t, restarted, "artifact-test-quotas", a.Run+"/TEST-1", a.Size)
	assertQuota(t, restarted, "artifact-project-quotas", a.Project, a.Size)
}

func TestUploadRecoveryRejectsMissingOrCorruptBytesAndAllowsSameIdentityRetry(t *testing.T) {
	for _, content := range []string{"missing", "corrupt"} {
		t.Run(content, func(t *testing.T) {
			s, _ := fixture(t)
			a := interruptManifest(t, s, "interrupted", content)
			setUploadDeadline(t, s, a.ID, time.Now().Add(-time.Second))
			if err := s.RecoverUploads(t.Context(), 10); err != nil {
				t.Fatal(err)
			}
			row, _ := s.Store.Get(t.Context(), evidence.Bucket(a.Run), a.ID)
			failed, _ := storage.Decode[evidence.Artifact](row)
			if failed.State != "upload-failed" || failed.Error == "" {
				t.Fatal("invalid content became available", failed)
			}
			if _, _, err := s.Open(t.Context(), a.Project, a.Run, a.ID); err == nil {
				t.Fatal("failed content became readable")
			}
			retry, err := s.SaveBytes(t.Context(), a, []byte("fresh recovery evidence"), 1<<20)
			if err != nil || retry.ID != a.ID || retry.State != "available" {
				t.Fatal("same attachment could not recover", retry, err)
			}
			rows, _ := s.Store.All(t.Context(), evidence.Bucket(a.Run), a.Project)
			if len(rows) != 1 {
				t.Fatal("retry duplicated the attachment")
			}
			assertQuota(t, s, "artifact-quotas", a.Run, a.Size)
		})
	}
}

func TestUploadRecoveryPagesPastLiveLeasesAndMigratesOldPendingManifests(t *testing.T) {
	s, _ := fixture(t)
	for _, id := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"} {
		interruptManifest(t, s, id, "complete")
	}
	work, err := s.Store.All(t.Context(), "artifact-upload-work", "project")
	if err != nil || len(work) != 12 {
		t.Fatal("missing durable recovery work", err)
	}
	var last struct {
		Artifact string `json:"artifactId"`
	}
	last, err = storage.Decode[struct {
		Artifact string `json:"artifactId"`
	}](work[len(work)-1])
	if err != nil {
		t.Fatal(err)
	}
	setUploadDeadline(t, s, last.Artifact, time.Now().Add(-time.Second))
	if err := s.RecoverUploads(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	row, _ := s.Store.Get(t.Context(), evidence.Bucket("run"), last.Artifact)
	before, _ := storage.Decode[evidence.Artifact](row)
	if before.State != "pending" {
		t.Fatal("scan skipped its bounded first page")
	}
	if err := s.RecoverUploads(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	row, _ = s.Store.Get(t.Context(), evidence.Bucket("run"), last.Artifact)
	after, _ := storage.Decode[evidence.Artifact](row)
	if after.State != "available" {
		t.Fatal("live first-page uploads starved interrupted work", after)
	}
	for _, index := range work {
		if err := s.Store.Commit(t.Context(), storage.Change{Record: index, Delete: true}); err != nil && !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
	}
	seedFinished(t, s)
	oldStatus, _ := storage.Put("evidence-retention-status", "run", "project", 0, evidence.RetentionStatus{State: "complete", IndexVersion: 1})
	if err := s.Store.Commit(t.Context(), oldStatus); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureRetentionIndex(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	if err := s.ProcessRetention(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	pending, _ := s.Store.All(t.Context(), "artifact-upload-work", "project")
	if len(pending) != 11 {
		t.Fatal("upgrade failed to index interrupted uploads", len(pending))
	}
}

type unavailableUploadRead struct{ artifacts.Store }

func (b unavailableUploadRead) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("temporary storage read outage")
}

type heldUploadRead struct {
	artifacts.Store
	entered chan struct{}
	release chan struct{}
	reads   atomic.Int32
}

func (b *heldUploadRead) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if b.reads.Add(1) == 1 {
		close(b.entered)
	}
	select {
	case <-b.release:
		return b.Store.Open(ctx, key)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestUploadRecoveryRetainsTransientReadFailureAndFencesOtherInstances(t *testing.T) {
	s, files := fixture(t)
	a := interruptManifest(t, s, "shared", "complete")
	setUploadDeadline(t, s, a.ID, time.Now().Add(-time.Second))
	s.Blobs = unavailableUploadRead{files}
	if err := s.RecoverUploads(t.Context(), 10); err == nil {
		t.Fatal("storage outage was hidden")
	}
	row, _ := s.Store.Get(t.Context(), evidence.Bucket(a.Run), a.ID)
	pending, _ := storage.Decode[evidence.Artifact](row)
	work, _ := s.Store.All(t.Context(), "artifact-upload-work", a.Project)
	if pending.State != "pending" || pending.Error == "" || len(work) != 1 {
		t.Fatal("transient storage loss discarded recovery", pending)
	}
	setUploadDeadline(t, s, a.ID, time.Now().Add(-time.Second))
	gate := &heldUploadRead{Store: files, entered: make(chan struct{}), release: make(chan struct{})}
	s.Blobs = gate
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	var recoveryErr error
	go func() { defer close(done); recoveryErr = s.RecoverUploads(ctx, 10) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("owned recovery did not stop")
		}
	}()
	select {
	case <-gate.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("recovery did not claim its upload")
	}
	other := &evidence.Service{Store: s.Store, Blobs: gate}
	if err := other.RecoverUploads(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	if gate.reads.Load() != 1 {
		t.Fatal("competing recovery duplicated content work")
	}
	close(gate.release)
	select {
	case <-done:
		if recoveryErr != nil {
			t.Fatal(recoveryErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("recovery did not finish")
	}
	row, _ = s.Store.Get(t.Context(), evidence.Bucket(a.Run), a.ID)
	available, _ := storage.Decode[evidence.Artifact](row)
	work, _ = s.Store.All(t.Context(), "artifact-upload-work", a.Project)
	if available.State != "available" || len(work) != 0 {
		t.Fatal("recovery did not finalize verified bytes", available)
	}
	assertQuota(t, s, "artifact-quotas", a.Run, a.Size)
}
