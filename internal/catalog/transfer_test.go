package catalog_test

import (
	"errors"
	"sync"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestImportPreviewAtomicApplyAndConcurrentIdempotency(t *testing.T) {
	s, _ := setup(t)
	first := check("HEALTH-1")
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", first, 0); err != nil {
		t.Fatal(err)
	}
	first.Title = "Revised title"
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{first, check("HEALTH-2")}}
	plan, err := s.PreviewImport(t.Context(), "admin", "alpha", "health", document)
	if err != nil || len(plan.Issues) != 0 || len(plan.Changes) != 2 || plan.Changes[0].Action != "update" || plan.Changes[1].Action != "create" {
		t.Fatalf("preview: %+v %v", plan, err)
	}
	if _, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "HEALTH-2"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("preview wrote a definition")
	}
	var workers sync.WaitGroup
	for range 3 {
		workers.Go(func() {
			if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	rows, err := s.Tests(t.Context(), "admin", "alpha", "health")
	if err != nil || len(rows) != 2 {
		t.Fatalf("applied: %+v %v", rows, err)
	}
	r, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "HEALTH-1")
	if err != nil || r.Revision != 2 {
		t.Fatalf("duplicate import writes: %+v %v", r, err)
	}
	exported, err := s.Export(t.Context(), "admin", "alpha", "health")
	if err != nil || exported.Schema != catalog.SchemaVersion || len(exported.Tests) != 2 || exported.Tests[0].Title != "Revised title" {
		t.Fatalf("export: %+v %v", exported, err)
	}
}

func TestCatalogMutationAfterPreviewPreventsEveryImportedWrite(t *testing.T) {
	s, _ := setup(t)
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check("HEALTH-1"), 0); err != nil {
		t.Fatal(err)
	}
	plan, err := s.PreviewImport(t.Context(), "admin", "alpha", "health", catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{check("HEALTH-2"), check("HEALTH-3")}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ChangeState(t.Context(), "admin", "alpha", "health", "HEALTH-1", "disabled", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale plan: %v", err)
	}
	for _, id := range []string{"HEALTH-2", "HEALTH-3"} {
		if _, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), id); !errors.Is(err, storage.ErrNotFound) {
			t.Fatal("stale import partially applied")
		}
	}
}

func TestImportValidationAndSafetyAuthorization(t *testing.T) {
	s, _ := setup(t)
	if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "editor", Name: "Editor"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "editor"}, "builtin.teststudio.editor", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	first := check("HEALTH-1")
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", first, 0); err != nil {
		t.Fatal(err)
	}
	first.Safety = "unsafe"
	if _, err := s.PreviewImport(t.Context(), "editor", "alpha", "health", catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{first}}); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("unprivileged safety change: %v", err)
	}
	if _, err := s.PreviewImport(t.Context(), "editor", "beta", "health", catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{check("NEW")}}); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("cross project import: %v", err)
	}
	invalid := check("BAD")
	invalid.Dependencies = []string{"missing"}
	plan, err := s.PreviewImport(t.Context(), "admin", "alpha", "health", catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{invalid, invalid}})
	if err != nil || len(plan.Issues) < 2 {
		t.Fatalf("field validation: %+v %v", plan, err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); err == nil {
		t.Fatal("invalid import applied")
	}
}

func TestBulkLifecycleAndDuplicateKeepIndependentHistory(t *testing.T) {
	s, _ := setup(t)
	for _, id := range []string{"HEALTH-1", "HEALTH-2"} {
		if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check(id), 0); err != nil {
			t.Fatal(err)
		}
	}
	input := catalog.BulkRequest{Action: "archive"}
	input.Tests = append(input.Tests, struct {
		ID       string `json:"id"`
		Revision int64  `json:"revision"`
	}{"HEALTH-1", 1}, struct {
		ID       string `json:"id"`
		Revision int64  `json:"revision"`
	}{"HEALTH-2", 1})
	plan, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input)
	if err != nil || len(plan.Issues) > 0 {
		t.Fatalf("bulk preview: %+v %v", plan, err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Duplicate(t.Context(), "admin", "alpha", "health", "HEALTH-1", "COPY-1", 2); err != nil {
		t.Fatal(err)
	}
	r, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "COPY-1")
	if err != nil {
		t.Fatal(err)
	}
	copy, err := storage.Decode[catalog.Test](r)
	if err != nil || r.Revision != 1 || copy.State != "draft" || copy.ID != "COPY-1" {
		t.Fatalf("duplicate identity: %+v %v", copy, err)
	}
	if _, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("stale bulk revisions: %v", err)
	}
}
