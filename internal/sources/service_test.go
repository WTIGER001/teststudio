package sources_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func setup(t *testing.T) (*sources.Service, *catalog.Service) {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "sources.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	directory := &identity.Directory{Store: db}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Admin"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	p := &projects.Service{Store: db, Auth: auth}
	for _, project := range []string{"alpha", "beta"} {
		if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: project, Name: project}, 0); err != nil {
			t.Fatal(err)
		}
		if err := p.SaveApplication(t.Context(), "admin", project, projects.Application{ID: "app", Name: "Application"}, 0); err != nil {
			t.Fatal(err)
		}
		if err := p.SaveSuite(t.Context(), "admin", project, projects.Suite{ID: "health", Name: "Health", Applications: []string{"app"}, RequiresTarget: true}, 0); err != nil {
			t.Fatal(err)
		}
	}
	return &sources.Service{Store: db, Auth: auth}, &catalog.Service{Store: db, Auth: auth}
}

func upload(t *testing.T, id string) sources.UploadInput {
	t.Helper()
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: id, Title: "Health", Category: "Availability", Type: "https", Runner: "http", State: "enabled", Safety: "safe", Rationale: "Reads health only", TimeoutSeconds: 5, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/healthz", Method: "GET", StatusCodes: []int{200}}}}}
	content, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return sources.UploadInput{Name: "health.json", Format: "json", Content: string(content)}
}

func staged(t *testing.T, service *sources.Service, input sources.UploadInput) sources.Preview {
	t.Helper()
	preview, err := service.Stage(t.Context(), "admin", "alpha", "health", input)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Plan == nil || len(preview.Plan.Issues) > 0 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	return preview
}

func TestNativeUploadAtomicReleaseAndImmutableSource(t *testing.T) {
	s, catalogs := setup(t)
	input := upload(t, "HEALTH-1")
	preview := staged(t, s, input)
	if _, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "HEALTH-1"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("preview changed catalog")
	}
	if _, err := catalogs.ApplyImport(t.Context(), "admin", "alpha", "health", preview.Plan.ID, 1); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("source plan bypassed atomic publication")
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if release.Draft.Artifact.SourceSnapshot != preview.Upload.ID || len(release.Draft.Artifact.SourceRevision) != 71 {
		t.Fatal("release lacks source provenance")
	}
	if _, err := catalogs.ActiveRelease(t.Context(), "alpha", "health"); err == nil {
		t.Fatal("publication activated implicitly")
	}
	again, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, false)
	if err != nil || again.ID != release.ID {
		t.Fatalf("retry: %v", err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, true); err == nil {
		t.Fatal("retry changed activation")
	}
	if err := catalogs.Activate(t.Context(), "admin", "alpha", "health", release.ID, 2); err != nil {
		t.Fatal(err)
	}
	content, err := s.Content(t.Context(), "admin", "alpha", "health", preview.Upload.ID)
	if err != nil || content.Content != input.Content || content.Digest != release.Draft.Artifact.SourceRevision {
		t.Fatalf("snapshot: %v", err)
	}
	rows, err := s.Store.All(t.Context(), "runs", "alpha")
	if err != nil || len(rows) != 0 {
		t.Fatal("publication executed tests")
	}
	second := staged(t, s, upload(t, "HEALTH-2"))
	next, err := s.Publish(t.Context(), "admin", "alpha", "health", second.Upload.ID, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	active, err := catalogs.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.ID != next.ID {
		t.Fatal("explicit activation failed")
	}
	original, err := s.Store.Get(t.Context(), catalog.Bucket("releases", "alpha", "health"), release.ID)
	if err != nil {
		t.Fatal(err)
	}
	frozen, _ := storage.Decode[catalog.Release](original)
	if len(frozen.Draft.Tests) != 1 {
		t.Fatal("source update mutated previous release")
	}
	if err := catalogs.Activate(t.Context(), "admin", "alpha", "health", release.ID, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Content(t.Context(), "admin", "alpha", "health", second.Upload.ID); err != nil {
		t.Fatal("rollback lost snapshot")
	}
}

func TestStalePreviewAndPublicationFailureRollbackEveryCatalogWrite(t *testing.T) {
	s, catalogs := setup(t)
	first := staged(t, s, upload(t, "HEALTH-1"))
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", first.Upload.ID, 1, true); err != nil {
		t.Fatal(err)
	}
	preview := staged(t, s, upload(t, "HEALTH-2"))
	if err := catalogs.ChangeState(t.Context(), "admin", "alpha", "health", "HEALTH-1", "disabled", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, true); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale upload: %v", err)
	}
	if _, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "HEALTH-2"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("stale preview partially applied")
	}
	coded := catalog.Test{ID: "CODED", Title: "Coded", Type: "browser", Runner: "playwright", Category: "Browser", State: "enabled", Safety: "safe", Rationale: "read", TimeoutSeconds: 5, Source: "test.spec.ts", Selector: "coded"}
	if err := catalogs.SaveTest(t.Context(), "admin", "alpha", "health", coded, 0); err != nil {
		t.Fatal(err)
	}
	preview = staged(t, s, upload(t, "HEALTH-3"))
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, true); err == nil {
		t.Fatal("nonexecutable release published")
	}
	if _, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "HEALTH-3"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("failed release left imported tests")
	}
	active, err := catalogs.ActiveRelease(t.Context(), "alpha", "health")
	original, _ := s.Read(t.Context(), "admin", "alpha", "health", first.Upload.ID)
	u, _ := storage.Decode[sources.Upload](original.Upload)
	if err != nil || active.ID != u.ReleaseID {
		t.Fatal("failed release replaced active release")
	}
	row, _ := s.Store.Get(t.Context(), catalog.Bucket("imports", "alpha", "health"), preview.Plan.ID)
	plan, _ := storage.Decode[catalog.ImportPlan](row)
	if plan.AppliedAt != nil {
		t.Fatal("failed publication consumed staging plan")
	}
}

func TestSourcePermissionsAndActorBinding(t *testing.T) {
	s, _ := setup(t)
	preview := staged(t, s, upload(t, "HEALTH-1"))
	for _, id := range []string{"viewer", "editor", "publisher", "other"} {
		if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
		role := id
		if id == "other" {
			role = "installation-admin"
		}
		if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: id}, "builtin.teststudio."+role, map[string]any{"team": "alpha"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Content(t.Context(), "viewer", "alpha", "health", preview.Upload.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Content(t.Context(), "viewer", "beta", "health", preview.Upload.ID); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("cross project content: %v", err)
	}
	for _, actor := range []string{"viewer", "editor", "publisher"} {
		if _, err := s.Stage(t.Context(), actor, "alpha", "health", upload(t, "BAD")); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatalf("unprivileged upload %s: %v", actor, err)
		}
	}
	if _, err := s.Publish(t.Context(), "other", "alpha", "health", preview.Upload.ID, 1, true); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("another actor consumed upload: %v", err)
	}
}

func TestStagingExpiryRemovesContentAndPlansButRetainsPublishedSnapshots(t *testing.T) {
	s, _ := setup(t)
	published := staged(t, s, upload(t, "HEALTH-1"))
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", published.Upload.ID, 1, true); err != nil {
		t.Fatal(err)
	}
	pending := staged(t, s, upload(t, "HEALTH-2"))
	row, err := s.Store.Get(t.Context(), "source-staging", pending.Upload.ID)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := storage.Decode[sources.Upload](row)
	u.ExpiresAt = time.Now().Add(-time.Hour)
	queue, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, u)
	metadata, _ := storage.Put(pending.Upload.Bucket, pending.Upload.ID, "alpha", 1, u)
	if err := s.Store.Commit(t.Context(), queue, metadata); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", pending.Upload.ID, 2, true); err == nil {
		t.Fatal("expired source published")
	}
	if _, err := s.Sweep(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ bucket, id string }{{"source-content/alpha", pending.Upload.ID}, {catalog.Bucket("imports", "alpha", "health"), pending.Plan.ID}, {"source-staging", pending.Upload.ID}} {
		if _, err := s.Store.Get(t.Context(), item.bucket, item.id); !errors.Is(err, storage.ErrNotFound) {
			t.Fatal("expired staging retained data")
		}
	}
	expired, err := s.Read(t.Context(), "admin", "alpha", "health", pending.Upload.ID)
	if err != nil || expired.Plan != nil {
		t.Fatal("expired preview remained available")
	}
	if _, err := s.Content(t.Context(), "admin", "alpha", "health", published.Upload.ID); err != nil {
		t.Fatal("cleanup removed published snapshot")
	}
}

func TestConcurrentPublicationDoesNotCreateDuplicateReleases(t *testing.T) {
	s, _ := setup(t)
	preview := staged(t, s, upload(t, "HEALTH-1"))
	var workers sync.WaitGroup
	ids := make(chan string, 3)
	for range 3 {
		workers.Go(func() {
			release, err := s.Publish(t.Context(), "admin", "alpha", "health", preview.Upload.ID, 1, true)
			if err != nil {
				t.Error(err)
				return
			}
			ids <- release.ID
		})
	}
	workers.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first != "" && first != id {
			t.Fatal("duplicate publication")
		}
		first = id
	}
	rows, err := s.Store.All(t.Context(), catalog.Bucket("releases", "alpha", "health"), "alpha")
	if err != nil || len(rows) != 1 {
		t.Fatalf("releases: %d %v", len(rows), err)
	}
}
