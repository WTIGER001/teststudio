package evidence_test

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestLegacyArtifactsGainIndexesAndProjectAccountingOnce(t *testing.T) {
	s, _ := fixture(t)
	a := expiredArtifact(t, s)
	r, err := s.Store.Get(t.Context(), evidence.Bucket("run"), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	a.QuotaVersion = 0
	a.UploadToken = ""
	a.UploadUntil = nil
	legacy, err := storage.Put(r.Bucket, r.ID, "project", r.Revision, a)
	if err != nil {
		t.Fatal(err)
	}
	changes := []storage.Change{legacy}
	for _, scope := range [][2]string{{"artifact-project-quotas", "project"}, {"artifact-test-quotas", "run/TEST-1"}} {
		row, err := s.Store.Get(t.Context(), scope[0], scope[1])
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, storage.Change{Record: row, Delete: true})
	}
	indexes, err := s.Store.All(t.Context(), "artifact-expirations", "project")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range indexes {
		changes = append(changes, storage.Change{Record: row, Delete: true})
	}
	if err := s.Store.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.EnsureRetentionIndex(t.Context(), 10); err != nil {
			t.Fatal(err)
		}
		if err := s.ProcessRetention(t.Context(), 10); err != nil {
			t.Fatal(err)
		}
	}
	assertQuota(t, s, "artifact-quotas", "run", a.Size)
	assertQuota(t, s, "artifact-project-quotas", "project", a.Size)
	assertQuota(t, s, "artifact-test-quotas", "run/TEST-1", a.Size)
	if err := s.Sweep(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	assertQuota(t, s, "artifact-project-quotas", "project", 0)
	assertQuota(t, s, "artifact-quotas", "run", 0)
}
