package catalog_test

import (
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"testing"
	"time"
)

func reviewFixture(t *testing.T) (*catalog.Service, catalog.Draft, catalog.Release) {
	t.Helper()
	s, _ := setup(t)
	for _, id := range []string{"A", "B"} {
		if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check(id), 0); err != nil {
			t.Fatal(err)
		}
	}
	draft, err := s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Publish(t.Context(), "admin", "alpha", "health", draft.ID, "initial", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	a := check("A")
	a.Title = "Changed assertion title"
	a.Safety = "unsafe"
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", a, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangeState(t.Context(), "admin", "alpha", "health", "B", "archived", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check("C"), 0); err != nil {
		t.Fatal(err)
	}
	draft, err = s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	return s, draft, first
}
func reviewSchedule(t *testing.T, s *catalog.Service, id, release string, generation int64) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), "schedules/alpha", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
	value := map[string]any{"id": id, "name": "New schedule " + id, "enabled": true, "archived": false, "generation": generation, "request": map[string]string{"suiteId": "health", "releaseId": release}, "lastRunId": "runtime-only"}
	change, err := storage.Put("schedules/alpha", id, "alpha", row.Revision, value)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
}
func TestPublicationReviewDiffSchedulePrivacyAndIdempotentPublication(t *testing.T) {
	s, draft, first := reviewFixture(t)
	reviewSchedule(t, s, "follow", "", 1)
	reviewSchedule(t, s, "pinned", first.ID, 1)
	r, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if r.Active != first.ID || r.Total != 2 || r.Enabled != 2 || len(r.Changes) != 3 || len(r.Schedules) != 2 || len(r.Issues) != 0 {
		t.Fatal(r)
	}
	if r.Changes[0].Action != "changed" || !r.Changes[0].SafetyChanged || r.Changes[1].Action != "removed" || r.Changes[2].Action != "added" {
		t.Fatal(r.Changes)
	}
	if !r.Schedules[0].FollowsActive || r.Schedules[1].FollowsActive {
		t.Fatal(r.Schedules)
	}
	row, _ := s.Store.Get(t.Context(), "schedules/alpha", "follow")
	value, _ := storage.Decode[map[string]any](row)
	value["lastRunId"] = "different-run"
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err := s.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	release, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "reviewed", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	active, err := s.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.ID != first.ID {
		t.Fatal("nonactivated publication replaced active release", err)
	}
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check("LATER"), 0); err != nil {
		t.Fatal(err)
	}
	replay, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "reviewed", 1, false)
	if err != nil || replay.ID != release.ID {
		t.Fatal("publication replay lost its original result", err)
	}
	if _, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "other-key", 1, false); !errors.Is(err, storage.ErrConflict) {
		t.Fatal(err)
	}
	if _, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "reviewed", 1, true); !errors.Is(err, storage.ErrConflict) {
		t.Fatal(err)
	}
	rows, _ := s.Store.All(t.Context(), catalog.Bucket("releases", "alpha", "health"), "alpha")
	if len(rows) != 2 {
		t.Fatal("publication duplicated", len(rows))
	}
}
func TestPublicationReviewRejectsStaleInputsWithoutPartialPublication(t *testing.T) {
	for _, kind := range []string{"catalog", "draft", "active", "project", "schedule", "expired"} {
		t.Run(kind, func(t *testing.T) {
			s, draft, first := reviewFixture(t)
			reviewSchedule(t, s, "follow", "", 1)
			r, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "catalog":
				err = s.SaveTest(t.Context(), "admin", "alpha", "health", check("NEW"), 0)
			case "draft":
				err = s.SaveDraft(t.Context(), "admin", "alpha", "health", draft, 1)
			case "active":
				row, _ := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
				err = s.Activate(t.Context(), "admin", "alpha", "health", first.ID, row.Revision)
			case "project":
				row, _ := s.Store.Get(t.Context(), "projects", "alpha")
				value, _ := storage.Decode[projects.Project](row)
				value.Description = "New description"
				change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
				err = s.Store.Commit(t.Context(), change)
			case "schedule":
				reviewSchedule(t, s, "follow", "", 2)
			case "expired":
				row, _ := s.Store.Get(t.Context(), catalog.Bucket("publication-reviews", "alpha", "health"), r.ID)
				value, _ := storage.Decode[map[string]any](row)
				value["review"].(map[string]any)["expiresAt"] = time.Now().Add(-time.Second)
				change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
				err = s.Store.Commit(t.Context(), change)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "stale", 1, true); !errors.Is(err, storage.ErrConflict) {
				t.Fatal(kind, err)
			}
			rows, _ := s.Store.All(t.Context(), catalog.Bucket("releases", "alpha", "health"), "alpha")
			if len(rows) != 1 {
				t.Fatal("rejected publication leaked a release", len(rows))
			}
			stored, _ := s.Store.Get(t.Context(), catalog.Bucket("drafts", "alpha", "health"), draft.ID)
			d, _ := storage.Decode[catalog.Draft](stored)
			if d.PublishedRelease != "" {
				t.Fatal("rejected draft became published")
			}
			active, _ := s.ActiveRelease(t.Context(), "alpha", "health")
			if active.ID != first.ID {
				t.Fatal("rejected publication changed activation")
			}
		})
	}
}
func TestPublicationReviewHonorsCurrentDenialsAndHidesUnreadableSchedules(t *testing.T) {
	s, draft, _ := reviewFixture(t)
	reviewSchedule(t, s, "private", "", 1)
	scope := "alpha/schedule/private"
	grants := &access.Service{Store: s.Store, Auth: s.Auth}
	if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &scope, PermissionName: "teststudio.schedules.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	version, err := s.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Directory.CommitAt(t.Context(), version); err != nil {
		t.Fatal(err)
	}
	r, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1)
	if err != nil || len(r.Schedules) != 0 {
		t.Fatal("review exposed a denied schedule", r, err)
	}
	if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "other", Name: "Other authorized publisher"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "other"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Check(t.Context(), "other", "alpha", "suite/health", "catalog.publish"); err != nil {
		t.Fatal("actor fixture lacks publication authority", err)
	}
	if _, err := s.ReadPublicationReview(t.Context(), "other", "alpha", "health", r.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("review actor isolation failed", err)
	}
	if _, err := s.PublishReviewed(t.Context(), "other", "alpha", "health", draft.ID, r.ID, "foreign-actor", 1, true); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("foreign actor used a review", err)
	}
	if _, err := s.ReadPublicationReview(t.Context(), "admin", "beta", "health", r.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
	scope = "test/health/A"
	if _, err := grants.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: scope, Reason: "Verify revoked reviewed publication access"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadPublicationReview(t.Context(), "admin", "alpha", "health", r.ID); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "denied", 1, true); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
}
func TestPublicationReviewFlagsAnAlreadyOutdatedDraft(t *testing.T) {
	s, draft, _ := reviewFixture(t)
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check("LATER"), 0); err != nil {
		t.Fatal(err)
	}
	r, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1)
	if err != nil || len(r.Issues) == 0 {
		t.Fatal(r, err)
	}
	if _, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, r.ID, "old-draft", 1, true); err == nil {
		t.Fatal("outdated draft published")
	}
}

func TestPublicationReviewConcurrentSameKeyKeepsOneRelease(t *testing.T) {
	s, draft, _ := reviewFixture(t)
	review, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		release catalog.Release
		err     error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			release, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, review.ID, "shared-key", 1, true)
			results <- outcome{release, err}
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.release.ID != second.release.ID {
		t.Fatal(first.err, second.err)
	}
	rows, _ := s.Store.All(t.Context(), catalog.Bucket("releases", "alpha", "health"), "alpha")
	if len(rows) != 2 {
		t.Fatal("concurrent publication duplicated", len(rows))
	}
}

func TestPublicationReviewRejectsRevokedPublishingAuthority(t *testing.T) {
	s, draft, _ := reviewFixture(t)
	review, err := s.ReviewPublication(t.Context(), "admin", "alpha", "health", draft.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	grants := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err := grants.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.publish", Effect: permissions.EffectDeny, Object: "suite/health", Reason: "Revoke publication after review"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublishReviewed(t.Context(), "admin", "alpha", "health", draft.ID, review.ID, "revoked-publisher", 1, true); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	rows, _ := s.Store.All(t.Context(), catalog.Bucket("releases", "alpha", "health"), "alpha")
	if len(rows) != 1 {
		t.Fatal("revoked authority created a release")
	}
}
