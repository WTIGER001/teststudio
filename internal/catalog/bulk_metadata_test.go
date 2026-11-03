package catalog_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestBulkMetadataPreservesSourceFactsAndAtomicRevisions(t *testing.T) {
	s, _ := setup(t)
	first, second := check("CODE-1"), check("CODE-2")
	for _, test := range []*catalog.Test{&first, &second} {
		test.Type, test.Runner, test.Source, test.Selector = "browser", "playwright", "checks/new.spec.ts", "@"+test.ID
		test.Check = nil
	}
	artifact := catalog.Artifact{SourceProject: "alpha", SourceSnapshot: "prepared-example", Image: "sha256:" + strings.Repeat("a", 64), Discovered: map[string]string{first.ID: first.Selector, second.ID: second.Selector}}
	prepared, err := s.PreviewPreparedImport(t.Context(), "admin", "alpha", "health", catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{first, second}}, "prepared-example", "example-source", artifact)
	if err != nil || len(prepared.Issues) != 0 {
		t.Fatal(prepared, err)
	}
	if _, err := s.ApplySourceImport(t.Context(), "admin", "alpha", "health", prepared.ID, 1, "prepared-example"); err != nil {
		t.Fatal(err)
	}
	category, owner, description := "Browser / Availability", "Example team", "Reviewed metadata"
	tags := []string{"reviewed", "local"}
	input := catalog.BulkRequest{Action: "metadata", Metadata: &catalog.BulkMetadata{Category: &category, Owner: &owner, Description: &description, Tags: &tags}}
	for _, test := range []catalog.Test{first, second} {
		input.Tests = append(input.Tests, struct {
			ID       string `json:"id"`
			Revision int64  `json:"revision"`
		}{test.ID, 1})
	}
	plan, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input)
	if err != nil || len(plan.Issues) != 0 || len(plan.Changes) != 2 {
		t.Fatal(plan, err)
	}
	for _, change := range plan.Changes {
		if change.Test.Owner != owner || change.Test.Category != category || !reflect.DeepEqual(change.Test.Tags, tags) || !catalog.SameSourceFacts(*change.Before, change.Test) || change.Test.State != change.Before.State {
			t.Fatal("metadata changed source facts or lifecycle", change)
		}
	}
	for range 2 {
		if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{first.ID, second.ID} {
		row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), id)
		if err != nil || row.Revision != 2 {
			t.Fatal("idempotent apply added revisions", row, err)
		}
	}
	input.Tests[0].Revision, input.Tests[1].Revision = 2, 2
	emptyOwner, emptyTags := "", []string{}
	input.Metadata = &catalog.BulkMetadata{Owner: &emptyOwner, Tags: &emptyTags}
	stale, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ChangeState(t.Context(), "admin", "alpha", "health", second.ID, "disabled", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", stale.ID, 1); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale metadata plan did not conflict", err)
	}
	row, _ := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), first.ID)
	stored, _ := storage.Decode[catalog.Test](row)
	if row.Revision != 2 || stored.Owner != owner || !reflect.DeepEqual(stored.Tags, tags) {
		t.Fatal("conflicting plan partially cleared metadata", stored)
	}
	input.Tests[1].Revision = 3
	clearing, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", clearing.ID, 1); err != nil {
		t.Fatal(err)
	}
	row, _ = s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), first.ID)
	stored, _ = storage.Decode[catalog.Test](row)
	if stored.Owner != "" || len(stored.Tags) != 0 || stored.Category != category || stored.DefinitionSource != "example-source" {
		t.Fatal("clear changed unselected metadata or source identity", stored)
	}
}

func TestBulkMetadataRequiresExplicitRationaleAndCurrentAuthority(t *testing.T) {
	s, _ := setup(t)
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", check("HEALTH-1"), 0); err != nil {
		t.Fatal(err)
	}
	safety, rationale, category := "unsafe", "Performs an intentional mutation", "Reviewed"
	input := catalog.BulkRequest{Action: "metadata", Metadata: &catalog.BulkMetadata{Safety: &safety}}
	input.Tests = append(input.Tests, struct {
		ID       string `json:"id"`
		Revision int64  `json:"revision"`
	}{"HEALTH-1", 1})
	if _, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input); err == nil {
		t.Fatal("safety selection reused an old rationale")
	}
	input.Metadata.Rationale = &rationale
	plan, err := s.PreviewBulk(t.Context(), "admin", "alpha", "health", input)
	if err != nil {
		t.Fatal(err)
	}
	object := "alpha/test/health/HEALTH-1"
	if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.catalog.publish", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("safety apply ignored current publishing denial", err)
	}
	input.Metadata = &catalog.BulkMetadata{Category: &category}
	plan, err = s.PreviewBulk(t.Context(), "admin", "alpha", "health", input)
	if err != nil {
		t.Fatal("ordinary metadata required publishing authority", err)
	}
	if err := s.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.catalog.update", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyImport(t.Context(), "admin", "alpha", "health", plan.ID, 1); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("metadata apply ignored current update denial", err)
	}
	row, _ := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "HEALTH-1")
	if row.Revision != 1 {
		t.Fatal("rejected metadata apply changed the definition")
	}
}
