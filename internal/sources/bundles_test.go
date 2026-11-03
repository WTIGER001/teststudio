package sources_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type bundleBuilder struct {
	origin          suitespec.Origin
	calls, cleanups int
	digest          string
	failure         error
}

func (b *bundleBuilder) Build(_ context.Context, input preparation.BuildRequest) (preparation.Build, error) {
	b.calls++
	b.digest = input.Snapshot.Digest
	b.origin = input.Origin
	now := time.Now().UTC()
	return preparation.Build{Image: "sha256:" + strings.Repeat("d", 64), SourceDigest: b.digest, StartedAt: now, FinishedAt: now, Log: "Retained assembly log\n"}, b.failure
}
func (b *bundleBuilder) CleanupBuild(context.Context, execution.Reference) error {
	b.cleanups++
	return nil
}

type bundleInspector struct {
	*inspector
	builder *bundleBuilder
}

func (p *bundleInspector) Inspect(ctx context.Context, input preparation.InspectionRequest) (preparation.Inspection, error) {
	result, err := p.inspector.Inspect(ctx, input)
	result.Discovery.Origin = p.builder.origin
	return result, err
}
func bundleSetup(t *testing.T) (*sources.Service, *catalog.Service, *bundleBuilder) {
	t.Helper()
	s, c, probe := preparedService(t)
	probe.count = 1
	builder := &bundleBuilder{}
	s.BundleRunners = []string{"go"}
	s.Builders = map[string]preparation.ArtifactBuilder{"go": builder}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"go": &bundleInspector{inspector: probe, builder: builder}}
	s.RecoveryInspectors = s.Inspectors
	return s, c, builder
}
func sourceZIP(t *testing.T) []byte {
	t.Helper()
	manifest := suitespec.Manifest{Schema: 1, Suite: "health", Type: "unit", Runner: "go", ModuleRoot: ".", Tests: suitespec.Tests{Packages: []string{"./checks"}}, Catalog: "catalog.json", BuildProfile: "go-module"}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "CODE-000", Title: "Source title", Category: "Unit", Type: "unit", Runner: "go", State: "enabled", Safety: "safe", Rationale: "Computes in memory", TimeoutSeconds: 5, Source: "checks/fresh_test.go", Selector: "./checks::TestVersion1Case0"}}}
	m, _ := json.Marshal(manifest)
	c, _ := json.Marshal(document)
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range map[string][]byte{"suites/health.json": m, "catalog.json": c, "go.mod": []byte("module source.local/bundle\n\ngo 1.24\n"), "checks/fresh_test.go": []byte("package checks\nimport \"testing\"\nfunc TestVersion1Case0(t *testing.T) {}\n")} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
func enqueueZIP(t *testing.T, s *sources.Service, key string) (storage.Record, []byte) {
	t.Helper()
	data := sourceZIP(t)
	row, err := s.EnqueueBundle(t.Context(), "admin", "alpha", "health", key, "source.zip", "suites/health.json", data)
	if err != nil {
		t.Fatal(err)
	}
	return row, data
}
func processBundle(t *testing.T, s *sources.Service, id string) (storage.Record, sources.Job) {
	t.Helper()
	if processed, err := s.ProcessBuildNext(t.Context(), "builder"); err != nil || !processed {
		t.Fatal("bundle worker", processed, err)
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "health", id)
	if err != nil {
		t.Fatal(err)
	}
	job, err := storage.Decode[sources.Job](row)
	if err != nil {
		t.Fatal(err)
	}
	return row, job
}

func TestBundleQueuePublicationRetainsExactSourceWithoutRunningTests(t *testing.T) {
	s, c, builder := bundleSetup(t)
	queued, data := enqueueZIP(t, s, "upload-one")
	replay, err := s.EnqueueBundle(t.Context(), "admin", "alpha", "health", "upload-one", "source.zip", "suites/health.json", data)
	if err != nil || replay.ID != queued.ID || builder.calls != 0 {
		t.Fatal("upload built or duplicated work", err)
	}
	if processed, err := s.ProcessNext(t.Context(), "api"); processed || err != nil {
		t.Fatal("API preparation consumed a build", err)
	}
	_, ready := processBundle(t, s, queued.ID)
	if ready.State != "ready" || ready.Image == "" || ready.Build == nil || !strings.Contains(ready.Log, "assembly log") {
		t.Fatalf("build result: %+v", ready)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	original, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", queued.ID)
	if err != nil || !bytes.Equal(original.Content, data) || release.Draft.Artifact.Provenance.SnapshotDigest != original.Digest {
		t.Fatal("source bytes or provenance lost", err)
	}
	if err := s.ApproveArtifact(t.Context(), release.Draft.Artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SweepBundles(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.ID != release.ID {
		t.Fatal("active release changed", err)
	}
	runs, err := s.Store.All(t.Context(), "runs", "alpha")
	if err != nil || len(runs) != 0 {
		t.Fatal("publication started tests")
	}
	if _, err := s.ReadBundle(t.Context(), "admin", "beta", "health", queued.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("bundle crossed project boundary", err)
	}
}
func TestBundleWithdrawalAndBuildFailureCleanOwnedImages(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "withdraw", true: "failed-build"}[failure], func(t *testing.T) {
			s, _, builder := bundleSetup(t)
			queued, _ := enqueueZIP(t, s, "one")
			if failure {
				builder.failure = errors.New("isolated assembly failed")
			}
			row, job := processBundle(t, s, queued.ID)
			if failure {
				if job.State != "failed" || !job.CleanupPending {
					t.Fatal("failed build became usable")
				}
			} else {
				if err := s.CancelJob(t.Context(), "admin", "alpha", "health", job.ID, row.Revision); err != nil {
					t.Fatal(err)
				}
			}
			stored, _ := s.ReadJob(t.Context(), "admin", "alpha", "health", job.ID)
			job, _ = storage.Decode[sources.Job](stored)
			deadline := time.Now().Add(-time.Minute)
			job.LeaseUntil = &deadline
			change, _ := storage.Put(stored.Bucket, stored.ID, stored.Project, stored.Revision, job)
			if err := s.Store.Commit(t.Context(), change); err != nil {
				t.Fatal(err)
			}
			s.Builders = nil
			s.Inspectors = nil
			if err := s.RecoverPreparations(t.Context(), "recovery"); err != nil {
				t.Fatal(err)
			}
			after, _ := s.ReadJob(t.Context(), "admin", "alpha", "health", job.ID)
			cleaned, _ := storage.Decode[sources.Job](after)
			if cleaned.CleanupPending || builder.cleanups != 1 || builder.calls != 1 {
				t.Fatal("recovery repeated work or skipped cleanup")
			}
			if failure && (!strings.Contains(cleaned.Error, "isolated assembly failed") || cleaned.State != "failed") {
				t.Fatal("cleanup replaced the recorded failure")
			}
			if !failure && cleaned.State != "cancelled" {
				t.Fatal("withdrawn review regained readiness")
			}
		})
	}
}
func TestBundleExpiredQueueRemovesSourceAndCannotDispatch(t *testing.T) {
	s, _, builder := bundleSetup(t)
	queued, _ := enqueueZIP(t, s, "expiry")
	source, _ := s.Store.Get(t.Context(), "source-bundles/alpha", queued.ID)
	bundle, _ := storage.Decode[sources.Bundle](source)
	bundle.ExpiresAt = time.Now().Add(-time.Hour)
	index, _ := s.Store.Get(t.Context(), "source-bundle-staging", queued.ID)
	saved, _ := storage.Put(source.Bucket, source.ID, source.Project, source.Revision, bundle)
	indexed, _ := storage.Put(index.Bucket, index.ID, index.Project, index.Revision, map[string]any{"jobId": queued.ID, "suiteId": "health", "expiresAt": bundle.ExpiresAt})
	if err := s.Store.Commit(t.Context(), saved, indexed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SweepBundles(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if processed, err := s.ProcessBuildNext(t.Context(), "builder"); err != nil || processed || builder.calls != 0 {
		t.Fatal("expired bundle dispatched")
	}
	if _, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", queued.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("expired bytes survived", err)
	}
	row, _ := s.ReadJob(t.Context(), "admin", "alpha", "health", queued.ID)
	job, _ := storage.Decode[sources.Job](row)
	if job.State != "expired" {
		t.Fatal("expired summary missing")
	}
}

func TestBundleExpiryPreservesRecordedBuildFailure(t *testing.T) {
	s, _, builder := bundleSetup(t)
	queued, _ := enqueueZIP(t, s, "failed-expiry")
	builder.failure = errors.New("retained assembly failure")
	_, job := processBundle(t, s, queued.ID)
	source, _ := s.Store.Get(t.Context(), "source-bundles/alpha", queued.ID)
	bundle, _ := storage.Decode[sources.Bundle](source)
	bundle.ExpiresAt = time.Now().Add(-time.Hour)
	index, _ := s.Store.Get(t.Context(), "source-bundle-staging", queued.ID)
	saved, _ := storage.Put(source.Bucket, source.ID, source.Project, source.Revision, bundle)
	indexed, _ := storage.Put(index.Bucket, index.ID, index.Project, index.Revision, map[string]any{"jobId": queued.ID, "suiteId": "health", "expiresAt": bundle.ExpiresAt})
	if err := s.Store.Commit(t.Context(), saved, indexed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SweepBundles(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	row, _ := s.ReadJob(t.Context(), "admin", "alpha", "health", job.ID)
	retained, _ := storage.Decode[sources.Job](row)
	if retained.State != "failed" || retained.Error != job.Error || !retained.SourceExpired || !retained.FinishedAt.Equal(*job.FinishedAt) {
		t.Fatal("source expiry erased preparation history")
	}
}

func TestBundleWorkerLeavesOtherRuntimeQueuedAndScansLaterPages(t *testing.T) {
	s, _, builder := bundleSetup(t)
	first, _ := enqueueZIP(t, s, "original")
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "health", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := storage.Decode[sources.Job](row)
	for index := range 26 {
		other := original
		other.ID = fmt.Sprintf("other-%02d", index)
		other.Runner = "playwright"
		change, _ := storage.Put(row.Bucket, other.ID, "alpha", 0, other)
		queued, _ := storage.Put("preparation-build-queue/local", fmt.Sprintf("000-%02d", index), "alpha", 0, map[string]string{"jobId": other.ID, "suiteId": "health", "executionLocation": "local"})
		if err := s.Store.Commit(t.Context(), change, queued); err != nil {
			t.Fatal(err)
		}
	}
	if processed, err := s.ProcessBuildNext(t.Context(), "go-worker"); err != nil || !processed {
		t.Fatal(processed, err)
	}
	if builder.calls != 1 {
		t.Fatal("Go build was starved by another runtime queue")
	}
	other, err := s.ReadJob(t.Context(), "admin", "alpha", "health", "other-00")
	if err != nil {
		t.Fatal(err)
	}
	waiting, _ := storage.Decode[sources.Job](other)
	if waiting.State != "received" || waiting.Error != "" {
		t.Fatal("unavailable runtime failed a queued build")
	}
}
