package catalog_test

import (
	"errors"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestRetainedReleaseRegistryKeepsHistoryAndConflictsAcrossRollback(t *testing.T) {
	service, projectsService := setup(t)
	test := check("RELEASE-001")
	if err := service.SaveTest(t.Context(), "admin", "alpha", "health", test, 0); err != nil {
		t.Fatal(err)
	}
	publish := func(key string) catalog.Release {
		t.Helper()
		draft, err := service.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
		if err != nil {
			t.Fatal(err)
		}
		release, err := service.Publish(t.Context(), "admin", "alpha", "health", draft.ID, key, 1, true)
		if err != nil {
			t.Fatal(err)
		}
		return release
	}
	first := publish("first")
	test.Title = "New release title"
	if err := service.SaveTest(t.Context(), "admin", "alpha", "health", test, 1); err != nil {
		t.Fatal(err)
	}
	second := publish("second")
	registry, err := service.ReleaseRegistry(t.Context(), "admin", "alpha", "health")
	if err != nil || len(registry.Items) != 2 || registry.Active != second.ID || !registry.CanActivate || registry.Items[0].ID != second.ID {
		t.Fatal(registry, err)
	}
	if err := service.Activate(t.Context(), "admin", "alpha", "health", first.ID, registry.SuiteRevision-1); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("activation ignored stale suite revision", err)
	}
	if err := service.Activate(t.Context(), "admin", "alpha", "health", first.ID, registry.SuiteRevision); err != nil {
		t.Fatal(err)
	}
	after, err := service.ReleaseRegistry(t.Context(), "admin", "alpha", "health")
	if err != nil || after.Active != first.ID || after.Items[0].Draft.Tests[0].Title != test.Title || after.Items[1].Draft.Tests[0].Title != first.Draft.Tests[0].Title {
		t.Fatal("rollback changed retained definitions", after, err)
	}
	if err := service.Activate(t.Context(), "admin", "beta", "health", first.ID, 1); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("release crossed a project boundary", err)
	}
	row, _ := service.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
	suite, _ := storage.Decode[projects.Suite](row)
	suite.Archived = true
	if err := projectsService.SaveSuite(t.Context(), "admin", "alpha", suite, row.Revision); err != nil {
		t.Fatal(err)
	}
	archived, err := service.ReleaseRegistry(t.Context(), "admin", "alpha", "health")
	if err != nil || len(archived.Items) != 2 || !archived.Archived || archived.CanActivate {
		t.Fatal("archive removed readable release history", archived, err)
	}
	if err := service.Activate(t.Context(), "admin", "alpha", "health", second.ID, archived.SuiteRevision); err == nil {
		t.Fatal("archived suite activated")
	}
}
func TestRetainedReleaseRegistryReflectsExactPublishDeny(t *testing.T) {
	service, _ := setup(t)
	control := &access.Service{Store: service.Store, Auth: service.Auth}
	if _, err := control.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.publish", Effect: permissions.EffectDeny, Object: "suite/health", Reason: "Verify exact retained-release authority"}); err != nil {
		t.Fatal(err)
	}
	registry, err := service.ReleaseRegistry(t.Context(), "admin", "alpha", "health")
	if err != nil || registry.CanActivate {
		t.Fatal(registry, err)
	}
	if err := service.Activate(t.Context(), "admin", "alpha", "health", "unavailable", registry.SuiteRevision); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("activation skipped current publish deny", err)
	}
}
