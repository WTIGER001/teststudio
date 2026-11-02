package catalog_test

import (
	"errors"
	"path/filepath"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func setup(t *testing.T) (*catalog.Service, *projects.Service) {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
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
		if err := p.SaveSuite(t.Context(), "admin", project, projects.Suite{ID: "health", Name: "Health", RequiresTarget: true}, 0); err != nil {
			t.Fatal(err)
		}
	}
	return &catalog.Service{Store: db, Auth: auth}, p
}

func check(id string) catalog.Test {
	return catalog.Test{ID: id, Title: "Endpoint is healthy", Type: "https", Runner: "http", State: "enabled", Category: "Availability", Safety: "safe", Rationale: "Reads the registered health endpoint.", TimeoutSeconds: 10, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/healthz", Method: "GET", StatusCodes: []int{200}}}
}

func TestPublicationFreezesCatalogAndChecksRevisions(t *testing.T) {
	s, _ := setup(t)
	test := check("HEALTH-001")
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", test, 0); err != nil {
		t.Fatal(err)
	}
	draft, err := s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", draft.ID, "publish-once", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Publish(t.Context(), "admin", "alpha", "health", draft.ID, "publish-once", 1, true)
	if err != nil || again.ID != release.ID {
		t.Fatalf("idempotent publish: %v %v", again, err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", draft.ID, "publish-once", 1, false); err == nil {
		t.Fatal("idempotency key ignored changed activation request")
	}
	test.Title = "Renamed health check"
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", test, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", test, 1); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale edit: %v", err)
	}
	active, err := s.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.Draft.Tests[0].Title == test.Title {
		t.Fatalf("published catalog mutated: %+v %v", active, err)
	}
	if err := s.ChangeState(t.Context(), "admin", "alpha", "health", test.ID, "disabled", 2); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckCurrentState(t.Context(), "alpha", "health", active.Draft.Tests, true); err == nil {
		t.Fatal("disabled test remains eligible through old release")
	}
	draft.Tests[0].Title = "Cannot overwrite publication"
	if err := s.SaveDraft(t.Context(), "admin", "alpha", "health", draft, 2); err == nil {
		t.Fatal("published draft was mutable")
	}
}

func TestInvalidPublicationPreservesActiveRelease(t *testing.T) {
	s, _ := setup(t)
	test := check("HEALTH-001")
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", test, 0); err != nil {
		t.Fatal(err)
	}
	d, err := s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Publish(t.Context(), "admin", "alpha", "health", d.ID, "first", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	test.Dependencies = []string{"DOES-NOT-EXIST"}
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", test, 1); err != nil {
		t.Fatal(err)
	}
	d, err = s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", d.ID, "invalid", 1, true); err == nil {
		t.Fatal("accepted unresolved dependency")
	}
	active, err := s.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.ID != first.ID {
		t.Fatal("failed publication displaced active release")
	}
}

func TestCatalogAuthorizationAndProtectedDependencies(t *testing.T) {
	s, _ := setup(t)
	if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "reader", Name: "Reader"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "reader"}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Tests(t.Context(), "reader", "beta", "health"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("project isolation: %v", err)
	}
	if err := s.SaveTest(t.Context(), "reader", "alpha", "health", check("H1"), 0); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("reader edited catalog: %v", err)
	}
	a, b := check("A"), check("B")
	a.Dependencies = []string{"B"}
	b.Safety = "unsafe"
	if _, err := catalog.ResolveSelection([]catalog.Test{a, b}, []string{"A"}, true); err == nil {
		t.Fatal("unsafe dependency ran on protected target")
	}
	selected, err := catalog.ResolveSelection([]catalog.Test{a, b}, []string{"A"}, false)
	if err != nil || len(selected) != 2 || selected[0].ID != "B" {
		t.Fatalf("dependency order: %+v %v", selected, err)
	}
}

func TestHealthDefinitionValidation(t *testing.T) {
	for _, path := range []string{"https://other.example/", "//other.example/path", "missing-slash"} {
		test := check("H1")
		test.Check.Path = path
		if len(catalog.Validate(catalog.Draft{Tests: []catalog.Test{test}})) == 0 {
			t.Errorf("accepted path %s", path)
		}
	}
	test := check("H1")
	test.Check.Method = "HEAD"
	test.Check.RequiredText = "healthy"
	if len(catalog.Validate(catalog.Draft{Tests: []catalog.Test{test}})) == 0 {
		t.Fatal("HEAD accepted a body assertion")
	}
}
