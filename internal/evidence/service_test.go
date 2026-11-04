package evidence_test

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type failing struct{ artifacts.Store }

func (f failing) Put(context.Context, string, io.Reader, int64) (artifacts.Object, error) {
	return artifacts.Object{}, errors.New("storage unavailable")
}

func TestConcurrentDuplicatesReserveStorageOnlyOnce(t *testing.T) {
	s, _ := fixture(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "same log\n", 100); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	r, err := s.Store.Get(t.Context(), "artifact-quotas", "run")
	if err != nil {
		t.Fatal(err)
	}
	quota, err := storage.Decode[evidence.Quota](r)
	if err != nil || quota.Bytes != 9 {
		t.Fatalf("duplicate quota reservation: %+v %v", quota, err)
	}
}

func TestDiagnosticPolicyCannotRelabelRetainedArtifact(t *testing.T) {
	s, _ := fixture(t)
	a := evidence.Artifact{ID: "diagnostic", Project: "project", Run: "run", Test: "API-1", Worker: "worker", Attempt: 1, Kind: "attachment", Filename: "response.json", ContentType: "application/json", RedactionPolicy: "api-diagnostics/v1"}
	data := []byte(`{"status":"up"}`)
	saved, err := s.SaveBytes(t.Context(), a, data, 1024)
	if err != nil || saved.RedactionPolicy != a.RedactionPolicy {
		t.Fatal("diagnostic policy was not retained", err)
	}
	a.RedactionPolicy = "another-policy"
	if _, err := s.SaveBytes(t.Context(), a, data, 1024); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("retained diagnostic policy can be relabeled", err)
	}
}

func TestMissingContentHasAnExplicitMetadataState(t *testing.T) {
	s, files := fixture(t)
	a, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "log\n", 100)
	if err != nil {
		t.Fatal(err)
	}
	if err := files.Delete(t.Context(), a.Project+"/"+a.Run+"/"+a.ID+"/"+a.Checksum); err != nil {
		t.Fatal(err)
	}
	missing, _, err := s.Open(t.Context(), "project", "run", a.ID)
	if !errors.Is(err, artifacts.ErrNotFound) || missing.State != "missing" {
		t.Fatalf("missing content was not marked: %+v %v", missing, err)
	}
}

func fixture(t *testing.T) (*evidence.Service, *filesystem.Store) {
	t.Helper()
	root := t.TempDir()
	db, err := testkit.Open(t, filepath.Join(root, "evidence.db"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filesystem.New(filepath.Join(root, "files"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close(); db.Close() })
	return &evidence.Service{Store: db, Blobs: files}, files
}

func TestDurableLogSegmentsAndIdempotency(t *testing.T) {
	s, _ := fixture(t)
	a, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "started\n", 100)
	if err != nil || a.State != "available" {
		t.Fatalf("capture: %+v %v", a, err)
	}
	if _, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "started\n", 100); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Store.All(t.Context(), evidence.Bucket("run"), "project")
	if err != nil || len(rows) != 1 {
		t.Fatalf("duplicate segment: %v %v", rows, err)
	}
	if _, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "changed\n", 100); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("segment changed after publication: %v", err)
	}
	_, reader, err := s.Open(t.Context(), "project", "run", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(data) != "started\n" {
		t.Fatalf("stored log: %s %v", data, err)
	}
	if _, _, err := s.Open(t.Context(), "other-project", "run", a.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("artifact crossed project boundary: %v", err)
	}
}

func TestQuotaAndFailedCaptureRemainExplicit(t *testing.T) {
	s, files := fixture(t)
	s.Blobs = failing{Store: files}
	a, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "start\n", 10)
	if err == nil || a.State != "upload-failed" {
		t.Fatalf("failed upload disappeared: %+v %v", a, err)
	}
	s.Blobs = files
	a, err = s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 1, "start\n", 10)
	if err != nil || a.State != "available" {
		t.Fatalf("upload retry: %+v %v", a, err)
	}
	if _, err := s.Log(t.Context(), "project", "run", "worker", "stdout", "TEST-1", 1, 2, "long log\n", 10); !errors.Is(err, evidence.ErrQuota) {
		t.Fatalf("quota was ignored: %v", err)
	}
}
