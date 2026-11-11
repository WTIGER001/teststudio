package runs_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type gitSourceGate struct {
	calls   int
	failure error
}

func (g *gitSourceGate) CheckRunSource(context.Context, string, string, string, catalog.Release) (*time.Time, []storage.Record, error) {
	g.calls++
	now := time.Now().UTC()
	return &now, nil, g.failure
}
func TestGitSourceChecksOnlyStartsAndPreservesReviewedVersions(t *testing.T) {
	s, _, _ := setup(t, nil)
	gate := &gitSourceGate{}
	s.Sources = gate
	active, err := s.Catalog.ActiveRelease(t.Context(), "project", "health")
	if err != nil {
		t.Fatal(err)
	}
	request := runs.Request{Suite: "health", Application: "app", Environment: "local", TimeoutSeconds: 60, ReviewedRelease: active.ID}
	if _, err = s.Estimate(t.Context(), "admin", "project", request); err != nil || gate.calls != 0 {
		t.Fatal("an estimate or schedule save refreshed Git", err)
	}
	run, err := s.Create(t.Context(), "admin", "project", request, "git-reviewed", "manual")
	if err != nil || gate.calls != 1 || run.SourceResolvedAt == nil {
		t.Fatal("run did not check source freshness", err)
	}
	if !reflect.DeepEqual(run.Release.Draft.Artifact, active.Draft.Artifact) {
		t.Fatal("source check changed the trusted immutable artifact")
	}
	repeated, err := s.Create(t.Context(), "admin", "project", request, "git-reviewed", "manual")
	if err != nil || repeated.ID != run.ID || gate.calls != 1 {
		t.Fatal("idempotent recovery refreshed or changed the pinned source", err)
	}
	stale := request
	stale.ReviewedRelease = "older-version"
	if _, err = s.Create(t.Context(), "admin", "project", stale, "git-stale", "manual"); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale manual review silently adopted a new catalog", err)
	}
	gate.failure = errors.New("repository unavailable")
	if _, err = s.CreateScheduled(t.Context(), "admin", "project", request, "git-scheduled", runs.ScheduleTrigger{ID: "schedule", Firing: "firing", DueAt: time.Now().UTC()}, func(runs.Run) ([]storage.Change, error) { return nil, nil }); err == nil || gate.calls != 2 {
		t.Fatal("scheduled start skipped the source check", err)
	}
	gate.failure = nil
	pinned := request
	pinned.Release = active.ID
	count := gate.calls
	if _, err = s.Create(t.Context(), "admin", "project", pinned, "git-retained", "manual"); err != nil || gate.calls != count {
		t.Fatal("explicit retained version fetched a moving branch", err)
	}
}
