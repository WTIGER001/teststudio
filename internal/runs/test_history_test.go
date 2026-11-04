package runs_test

import (
	"errors"
	"fmt"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"io"
	"testing"
	"time"
)

func TestRetainedTestHistorySurvivesArchiveRemovalAndPinnedEvidence(t *testing.T) {
	s, p, _ := setup(t, nil)
	start(t, s)
	first, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "history-first", "manual")
	if err != nil {
		t.Fatal(err)
	}
	original := finished(t, s, first.ID)
	attachment, err := s.Evidence.SaveBytes(t.Context(), evidence.Artifact{Project: "project", Run: original.ID, Test: "HEALTH-1", Attempt: 1, Kind: "attachment", Filename: "retained-context.json", ContentType: "application/json"}, []byte(`{"fresh":"lifecycle evidence"}`), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Evidence.SavePin(t.Context(), "admin", "project", original.ID, true, nil, "Retain lifecycle investigation", 0); err != nil {
		t.Fatal(err)
	}
	row, _ := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	test, _ := storage.Decode[catalog.Test](row)
	test.Title = "Changed current title"
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, row.Revision); err != nil {
		t.Fatal(err)
	}
	draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "history-second-release", 1, true); err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local"}, "history-second", "manual")
	if err != nil {
		t.Fatal(err)
	}
	finished(t, s, second.ID)
	if err := s.Catalog.ChangeState(t.Context(), "admin", "project", "health", "HEALTH-1", "archived", 2); err != nil {
		t.Fatal(err)
	}
	suiteRow, _ := s.Store.Get(t.Context(), projects.Bucket("suites", "project"), "health")
	if _, err := p.ChangeSuiteState(t.Context(), "admin", "project", "health", "archive", suiteRow.Revision); err != nil {
		t.Fatal(err)
	}
	retained, err := s.Catalog.RetainedTest(t.Context(), "admin", "project", "health", "HEALTH-1")
	if err != nil || !retained.SuiteArchived || retained.Lifecycle != "archived" || retained.Test.Title != test.Title {
		t.Fatal(retained, err)
	}
	page, err := s.TestHistory(t.Context(), "admin", "project", "health", "HEALTH-1", "", 1)
	if err != nil || len(page.Items) != 1 || page.Items[0].Run != second.ID || page.Items[0].Title != test.Title || page.Cursor == "" {
		t.Fatal(page, err)
	}
	older, err := s.TestHistory(t.Context(), "admin", "project", "health", "HEALTH-1", page.Cursor, 1)
	if err != nil || len(older.Items) != 1 || older.Items[0].Run != first.ID || older.Items[0].Title != "Health" {
		t.Fatal(older, err)
	}
	row, _ = s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err := s.Store.Commit(t.Context(), storage.Change{Record: row, Delete: true}); err != nil {
		t.Fatal(err)
	}
	removed, err := s.Catalog.RetainedTest(t.Context(), "admin", "project", "health", "HEALTH-1")
	if err != nil || removed.Current || removed.Lifecycle != "removed" || removed.Test.Title != test.Title {
		t.Fatal(removed, err)
	}
	_, content, err := s.Evidence.Open(t.Context(), "project", first.ID, attachment.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer content.Close()
	data, err := io.ReadAll(content)
	if err != nil || string(data) != `{"fresh":"lifecycle evidence"}` {
		t.Fatal(string(data), err)
	}
	saved, _, _ := s.Get(t.Context(), "project", first.ID)
	if saved.Tests[0].Title != "Health" || saved.Release.ID != original.Release.ID || saved.Results[0].State != "passed" {
		t.Fatal("history changed run snapshots")
	}
}

func TestRetainedTestHistoryBoundsScansAndUsesCurrentExactPermissions(t *testing.T) {
	s, _, _ := setup(t, nil)
	changes := []storage.Change{}
	for index := 0; index < 505; index++ {
		test := catalog.Test{ID: "OTHER", Title: "Unrelated snapshot"}
		if index == 0 {
			test.ID = "TARGET"
			test.Title = "Original target"
		}
		value := runs.Run{ID: fmt.Sprintf("history-%04d", index), Project: "project", Request: runs.Request{Suite: "health", Location: "local"}, Tests: []catalog.Test{test}, State: "completed", Outcome: "passed", CreatedAt: time.Now().UTC()}
		row, err := storage.Put("runs", value.ID, "project", 0, value)
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, row)
	}
	if err := s.Store.CommitAll(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	page, err := s.TestHistory(t.Context(), "admin", "project", "health", "TARGET", "", 100)
	if err != nil || page.Scanned != 500 || !page.Partial || page.Cursor == "" || len(page.Items) != 0 {
		t.Fatal(page, err)
	}
	next, err := s.TestHistory(t.Context(), "admin", "project", "health", "TARGET", page.Cursor, 100)
	if err != nil || next.Scanned != 5 || len(next.Items) != 1 || next.Items[0].TestState != "not-recorded" || next.Items[0].Duration != nil {
		t.Fatal(next, err)
	}
	if _, err := s.TestHistory(t.Context(), "admin", "project", "health", "OTHER", page.Cursor, 100); err == nil {
		t.Fatal("cursor changed test scope")
	}
	grants := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err := grants.CreateGrant(t.Context(), "admin", "project", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.runs.read", Effect: permissions.EffectDeny, Object: "run/history-0000", Reason: "Keep a run private in test history"}); err != nil {
		t.Fatal(err)
	}
	denied, err := s.TestHistory(t.Context(), "admin", "project", "health", "TARGET", page.Cursor, 100)
	if err != nil || len(denied.Items) != 0 {
		t.Fatal("denied run entered test history", denied, err)
	}
	if _, err := grants.CreateGrant(t.Context(), "admin", "project", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: "test/health/HEALTH-1", Reason: "Revoke retained definition access"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Catalog.RetainedTest(t.Context(), "admin", "project", "health", "HEALTH-1"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := s.TestHistory(t.Context(), "admin", "project", "health", "HEALTH-1", "", 20); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
}
