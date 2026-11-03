package sources_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type inspector struct {
	count    int
	version  int
	failure  error
	entered  chan struct{}
	calls    atomic.Int32
	cleanups atomic.Int32
}

func (i *inspector) Reference(ref execution.Reference) execution.Reference {
	ref.Profile = "go-local-isolated-v1"
	ref.Container = "inspection-" + ref.ID
	ref.Volume = "inspection-output-" + ref.ID
	return ref
}
func (i *inspector) CleanupInspection(context.Context, preparation.InspectionRequest) error {
	i.cleanups.Add(1)
	return nil
}
func (i *inspector) Inspect(ctx context.Context, input preparation.InspectionRequest) (preparation.Inspection, error) {
	i.calls.Add(1)
	if i.entered != nil {
		close(i.entered)
		<-ctx.Done()
		return preparation.Inspection{}, ctx.Err()
	}
	if i.failure != nil {
		return preparation.Inspection{}, i.failure
	}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{}}
	found := []suitespec.DiscoveredTest{}
	for n := range i.count {
		id := fmt.Sprintf("CODE-%03d", n)
		selector := fmt.Sprintf("./checks::TestVersion%dCase%d", i.version, n)
		document.Tests = append(document.Tests, catalog.Test{ID: id, Title: "Source title", Category: "Unit", Description: "Source description", Runner: "go", Type: "unit", State: "enabled", Safety: "safe", Rationale: "Computes only in memory", TimeoutSeconds: 5, Source: "checks/fresh_test.go", Selector: selector})
		found = append(found, suitespec.DiscoveredTest{ID: id, Selector: selector, File: "checks/fresh_test.go", Line: n + 1})
	}
	return preparation.Inspection{Project: input.Reference.Project, Image: input.Image, ImageID: input.Image, Runtime: "go-local", Discovery: suitespec.Discovery{Schema: suitespec.DiscoverySchema, Manifest: suitespec.Manifest{Schema: 1, Suite: input.Suite, Runner: "go", Type: "unit", ModuleRoot: ".", Catalog: "catalog.json", BuildProfile: "go-module", Tests: suitespec.Tests{Packages: []string{"./checks"}}}, Origin: suitespec.Origin{SnapshotDigest: "sha256:" + strings.Repeat("a", 64)}, Catalog: document, Tests: found}}, nil
}
func preparedService(t *testing.T) (*sources.Service, *catalog.Service, *inspector) {
	t.Helper()
	s, c := setup(t)
	probe := &inspector{count: 40, version: 1}
	s.Location = "local"
	s.Inspectors = map[string]preparation.ArtifactInspector{"go": probe}
	row, err := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, _ := storage.Decode[projects.Suite](row)
	suite.RequiresTarget = false
	if err := (&projects.Service{Store: s.Store, Auth: s.Auth}).SaveSuite(t.Context(), "admin", "alpha", suite, row.Revision); err != nil {
		t.Fatal(err)
	}
	return s, c, probe
}
func queue(t *testing.T, s *sources.Service, key string) (storage.Record, sources.Job) {
	t.Helper()
	row, err := s.EnqueuePrebuilt(t.Context(), "admin", "alpha", "health", key, sources.PrebuiltInput{Runner: "go", Image: "sha256:" + strings.Repeat(key[:1], 64)})
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	return row, job
}
func process(t *testing.T, s *sources.Service, id string) sources.Job {
	t.Helper()
	processed, err := s.ProcessNext(t.Context(), "worker")
	if err != nil || !processed {
		t.Fatalf("worker: %v %v", processed, err)
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "health", id)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	return job
}

func TestPrebuiltPreparationPublicationPreservesMetadataAndHistory(t *testing.T) {
	s, c, probe := preparedService(t)
	firstRow, first := queue(t, s, "a-first")
	replay, err := s.EnqueuePrebuilt(t.Context(), "admin", "alpha", "health", "a-first", sources.PrebuiltInput{Runner: "go", Image: first.Image})
	if err != nil || replay.ID != firstRow.ID {
		t.Fatal("duplicate event created another preparation")
	}
	if probe.calls.Load() != 0 {
		t.Fatal("enqueue ran image inspection in the request")
	}
	ready := process(t, s, first.ID)
	if ready.State != "ready" || probe.calls.Load() != 1 {
		t.Fatalf("preparation: %+v", ready)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.Draft.Tests) != 40 {
		t.Fatal("complete prepared catalog was truncated")
	}
	if err := s.ApproveArtifact(t.Context(), release.Draft.Artifact); err != nil {
		t.Fatal(err)
	}
	foreign := release.Draft.Artifact
	foreign.SourceProject = "beta"
	if err := s.ApproveArtifact(t.Context(), foreign); err == nil {
		t.Fatal("artifact admission crossed projects")
	}
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "CODE-000")
	if err != nil {
		t.Fatal(err)
	}
	test, _ := storage.Decode[catalog.Test](row)
	test.Title = "Managed title"
	test.Tags = []string{"managed"}
	test.Owner = "QA"
	if err := c.SaveTest(t.Context(), "admin", "alpha", "health", test, row.Revision); err != nil {
		t.Fatal(err)
	}
	test.Selector = "./checks::TestReplaced"
	if err := c.SaveTest(t.Context(), "admin", "alpha", "health", test, row.Revision+1); err == nil {
		t.Fatal("manual editor changed source-owned selector")
	}
	probe.count, probe.version = 39, 2
	_, next := queue(t, s, "b-second")
	ready = process(t, s, next.ID)
	if ready.State != "ready" {
		t.Fatalf("second review: %+v", ready)
	}
	second, err := s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Draft.Tests[0].Title != "Managed title" || second.Draft.Tests[0].Tags[0] != "managed" || second.Draft.Tests[0].Owner != "QA" || !strings.Contains(second.Draft.Tests[0].Selector, "Version2") {
		t.Fatal("source synchronization overwrote metadata or lost discovered facts")
	}
	removed, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "CODE-039")
	if err != nil {
		t.Fatal(err)
	}
	archived, _ := storage.Decode[catalog.Test](removed)
	if archived.State != "archived" {
		t.Fatal("removed source test lost its archive state")
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.ID != release.ID || !strings.Contains(active.Draft.Tests[0].Selector, "Version1") {
		t.Fatal("preparation or inactive publication replaced the active snapshot")
	}
	probe.failure = errors.New("image preparation failed")
	_, failed := queue(t, s, "c-failed")
	failed = process(t, s, failed.ID)
	if failed.State != "failed" || failed.Upload != "" {
		t.Fatal("failed image became publishable")
	}
	active, _ = c.ActiveRelease(t.Context(), "alpha", "health")
	if active.ID != release.ID {
		t.Fatal("failed preparation replaced the active release")
	}
	runs, err := s.Store.All(t.Context(), "runs", "alpha")
	if err != nil || len(runs) != 0 {
		t.Fatal("publication or preparation started tests")
	}
}

func TestPrebuiltCancellationDuringInspectionDoesNotPublish(t *testing.T) {
	s, _, probe := preparedService(t)
	probe.entered = make(chan struct{})
	_, job := queue(t, s, "a-cancel")
	done := make(chan error, 1)
	go func() { _, err := s.ProcessNext(t.Context(), "worker"); done <- err }()
	select {
	case <-probe.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("inspection did not start")
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "health", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CancelJob(t.Context(), "admin", "alpha", "health", job.ID, row.Revision); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preparation did not observe cancellation")
	}
	row, err = s.ReadJob(t.Context(), "admin", "alpha", "health", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, _ := storage.Decode[sources.Job](row)
	if cancelled.State != "cancelled" || cancelled.Upload != "" {
		t.Fatal("cancelled inspection produced a source review")
	}
}

func TestInterruptedPrebuiltPreparationCleansWithoutReinspection(t *testing.T) {
	s, _, probe := preparedService(t)
	row, job := queue(t, s, "a-recover")
	ref := probe.Reference(execution.Reference{ID: "attempt", Run: job.ID, Installation: "preparation-local", Project: "alpha", Test: "catalog", Attempt: 1, Worker: "old-worker"})
	past := time.Now().Add(-time.Minute)
	job.State, job.Phase, job.Worker = "discovering", "discovering", "old-worker"
	job.Reference, job.LeaseUntil = &ref, &past
	saved, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
	pending, err := s.Store.All(t.Context(), "preparation-queue/local", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	lease, _ := storage.Put("preparation-leases/local", job.ID, "alpha", 0, map[string]string{"jobId": job.ID, "suiteId": "health", "executionLocation": "local"})
	if err := s.Store.Commit(t.Context(), saved, storage.Change{Record: pending[0], Delete: true}, lease); err != nil {
		t.Fatal(err)
	}
	s.RecoveryInspectors = s.Inspectors
	s.Inspectors = nil
	if err := s.RecoverPreparations(t.Context(), "new-worker"); err != nil {
		t.Fatal(err)
	}
	row, err = s.ReadJob(t.Context(), "admin", "alpha", "health", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	recovered, _ := storage.Decode[sources.Job](row)
	if recovered.State != "failed" || recovered.Phase != "finished" || probe.cleanups.Load() != 1 || probe.calls.Load() != 0 {
		t.Fatalf("interrupted preparation was rerun or not recovered: %+v", recovered)
	}
}

func TestPrebuiltOwnershipCannotBeAssignedByEditingOrDuplication(t *testing.T) {
	s, c, probe := preparedService(t)
	probe.count = 1
	_, job := queue(t, s, "a-owned")
	ready := process(t, s, job.ID)
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, false); err != nil {
		t.Fatal(err)
	}
	if err := c.Duplicate(t.Context(), "admin", "alpha", "health", "CODE-000", "DETACHED-001", 1); err != nil {
		t.Fatal(err)
	}
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "DETACHED-001")
	if err != nil {
		t.Fatal(err)
	}
	detached, err := storage.Decode[catalog.Test](row)
	if err != nil || detached.DefinitionSource != "" || detached.State != "draft" {
		t.Fatal("duplicate inherited source ownership", err)
	}
	original, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "CODE-000")
	if err != nil {
		t.Fatal(err)
	}
	owned, _ := storage.Decode[catalog.Test](original)
	detached.DefinitionSource = owned.DefinitionSource
	if err := c.SaveTest(t.Context(), "admin", "alpha", "health", detached, row.Revision); err == nil {
		t.Fatal("manual edit assigned source ownership")
	}
}
