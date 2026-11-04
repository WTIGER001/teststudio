package catalog_test

import (
	"encoding/json"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/paging"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"testing"
	"time"
)

func listFixture(t *testing.T) *catalog.Service {
	t.Helper()
	s, p := setup(t)
	if err := p.SaveSuite(t.Context(), "admin", "alpha", projects.Suite{ID: "second", Name: "Second"}, 0); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct{ id, suite, title, category string }{{"A", "health", "Same name", "Availability"}, {"B", "health", "Local check", "Availability / Local"}, {"C", "health", "Adjacent category", "AvailabilityExtra"}, {"A", "second", "Same name", "Availability/Local"}, {"HIDDEN", "health", "Private title", "Private"}} {
		value := check(entry.id)
		value.Title = entry.title
		value.Category = entry.category
		value.Tags = []string{"smoke"}
		value.Owner = "QA"
		value.Source = "new-source"
		if err := s.SaveTest(t.Context(), "admin", "alpha", entry.suite, value, 0); err != nil {
			t.Fatal(err)
		}
	}
	return s
}
func TestCatalogListFiltersCategoryFacetsAndStablePages(t *testing.T) {
	s := listFixture(t)
	q := catalog.ListQuery{State: "active", Category: " Availability ", Descendants: true, Tag: "smoke", Owner: "QA", Runner: "http", Safety: "safe", Source: "new-source", Type: "https", Sort: "id", Limit: 2}
	first, err := s.ListTests(t.Context(), "admin", "alpha", "", q, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 5 || first.Matched != 3 || len(first.Items) != 2 || first.Cursor == "" || first.Items[0].ID != "A" || first.Items[0].Suite != "health" || first.Items[1].Suite != "second" {
		t.Fatal(first)
	}
	if len(first.Categories) != 3 || first.Categories[0].Path != "Availability" || first.Categories[0].Count != 3 || first.Categories[0].Children[0].Count != 2 {
		t.Fatal(first.Categories)
	}
	second, err := s.ListTests(t.Context(), "admin", "alpha", "", q, first.Cursor)
	if err != nil || len(second.Items) != 1 || second.Items[0].ID != "B" || second.Cursor != "" {
		t.Fatal(second, err)
	}
	q.Descendants = false
	q.Limit = 30
	exact, err := s.ListTests(t.Context(), "admin", "alpha", "", q, "")
	if err != nil || exact.Matched != 1 {
		t.Fatal(exact, err)
	}
	q.Category = ""
	q.Query = "LOCAL"
	q.Suite = "health"
	searched, err := s.ListTests(t.Context(), "admin", "alpha", "", q, "")
	if err != nil || searched.Matched != 1 || searched.Items[0].ID != "B" {
		t.Fatal(searched, err)
	}
	for _, query := range []catalog.ListQuery{{Limit: 101}, {Sort: "unknown"}, {State: "unknown"}, {Category: "a", Uncategorized: true}, {Owner: "bad\x00value"}} {
		if _, err := s.ListTests(t.Context(), "admin", "alpha", "", query, ""); err == nil {
			t.Fatal("invalid query accepted", query)
		}
	}
}
func TestCatalogListOmitsDeniedTestsAndSuitesFromEverySummary(t *testing.T) {
	s := listFixture(t)
	grants := &access.Service{Store: s.Store, Auth: s.Auth}
	for _, object := range []string{"test/health/HIDDEN", "suite/second"} {
		if _, err := grants.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: object, Reason: "Verify catalog list privacy"}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.ListTests(t.Context(), "admin", "alpha", "", catalog.ListQuery{}, "")
	if err != nil || page.Total != 3 || page.Matched != 3 || page.Counts.Total != 3 || page.Counts.Suites != 1 {
		t.Fatal(page, err)
	}
	if len(page.Categories) != 2 {
		t.Fatal("denied categories contributed to navigation", page.Categories)
	}
	for _, entry := range page.Items {
		if entry.ID == "HIDDEN" || entry.Suite == "second" {
			t.Fatal("denied entry leaked", entry)
		}
	}
	if _, err := s.ListTests(t.Context(), "admin", "alpha", "second", catalog.ListQuery{}, ""); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := s.Export(t.Context(), "admin", "alpha", "health"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("export bypassed exact test read denial", err)
	}
}
func TestCatalogListOpaqueCursorRejectsChangedScopeCatalogAndAccess(t *testing.T) {
	for _, kind := range []string{"filter", "project", "route", "catalog", "access", "tampered", "actor", "expired"} {
		t.Run(kind, func(t *testing.T) {
			s := listFixture(t)
			q := catalog.ListQuery{Limit: 2}
			page, err := s.ListTests(t.Context(), "admin", "alpha", "", q, "")
			if err != nil {
				t.Fatal(err)
			}
			project, route, cursor := "alpha", "", page.Cursor
			user := "admin"
			switch kind {
			case "filter":
				q.Sort = "id"
			case "project":
				project = "beta"
			case "route":
				route = "health"
			case "catalog":
				row, _ := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "A")
				value, _ := storage.Decode[catalog.Test](row)
				value.Title = "Changed title"
				if err := s.SaveTest(t.Context(), "admin", "alpha", "health", value, row.Revision); err != nil {
					t.Fatal(err)
				}
			case "access":
				grants := &access.Service{Store: s.Store, Auth: s.Auth}
				if _, err := grants.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: "test/health/HIDDEN", Reason: "Revoke a catalog row between pages"}); err != nil {
					t.Fatal(err)
				}
			case "actor":
				if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "other", Name: "Another list reader"}, 0); err != nil {
					t.Fatal(err)
				}
				if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "other"}, "builtin.teststudio.installation-admin", nil); err != nil {
					t.Fatal(err)
				}
				user = "other"
			case "expired":
				normalized := q
				normalized.State = "all"
				normalized.Sort = "title"
				data, _ := json.Marshal(normalized)
				scope := "admin\x00alpha\x00\x00" + string(data)
				raw, err := paging.Open(t.Context(), s.Store, "catalog-list-settings", scope, cursor)
				if err != nil {
					t.Fatal(err)
				}
				var value map[string]any
				if err := json.Unmarshal([]byte(raw), &value); err != nil {
					t.Fatal(err)
				}
				value["expiresAt"] = time.Now().Add(-time.Second)
				data, _ = json.Marshal(value)
				cursor, err = paging.Seal(t.Context(), s.Store, "catalog-list-settings", scope, string(data))
				if err != nil {
					t.Fatal(err)
				}
			case "tampered":
				cursor = cursor[:len(cursor)-3] + "bad"
			}
			if _, err := s.ListTests(t.Context(), user, project, route, q, cursor); err == nil {
				t.Fatal("stale or foreign cursor accepted", kind)
			}
		})
	}
}

func TestCatalogExportRejectsExactExportDenyWithoutRestrictingListReads(t *testing.T) {
	s := listFixture(t)
	grants := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err := grants.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.export", Effect: permissions.EffectDeny, Object: "test/health/A", Reason: "Verify export permission stays separate from read"}); err != nil {
		t.Fatal(err)
	}
	page, err := s.ListTests(t.Context(), "admin", "alpha", "", catalog.ListQuery{}, "")
	if err != nil || page.Total != 5 {
		t.Fatal("export denial removed permitted catalog reads", page, err)
	}
	if _, err := s.Export(t.Context(), "admin", "alpha", "health"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("exact export denial was bypassed", err)
	}
}
