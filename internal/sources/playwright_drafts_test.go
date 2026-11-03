package sources_test

import (
	"errors"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestPlaywrightEditorDraftRevisionAndImmutablePreparation(t *testing.T) {
	s, _, _ := preparedService(t)
	s.BundleRunners = []string{"playwright"}
	value := sources.StarterDraft()
	value.ID = "welcome"
	row, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "once", row.Revision)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "once", row.Revision)
	if err != nil || first.ID != again.ID {
		t.Fatal("retry created another preparation", err)
	}
	job, _ := storage.Decode[sources.Job](first)
	if job.Draft != value.ID || job.DraftRevision != row.Revision || job.State != "received" {
		t.Fatal("missing frozen draft provenance")
	}
	bundle, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := sourcebundle.Validate(bundle.Content, bundle.Manifest, "health")
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot.Files["checks/welcome.spec.ts"]) != value.Files[0].Content || !strings.Contains(string(snapshot.Files["package.json"]), `"@playwright/test":"1.64.1"`) {
		t.Fatal("generated bundle differs from saved draft")
	}
	value.Files[0].Content += "\nexport const changed = true\n"
	updated, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, row.Revision)
	if err != nil || updated.Revision != row.Revision+1 {
		t.Fatal("draft update failed", err)
	}
	if _, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, row.Revision); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale update accepted", err)
	}
	if _, err := s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "stale", row.Revision); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale preparation accepted", err)
	}
	retained, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", first.ID)
	if err != nil || string(retained.Content) != string(bundle.Content) {
		t.Fatal("editing changed an immutable preparation")
	}
	if _, err := s.Draft(t.Context(), "admin", "beta", "health", value.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("draft crossed project boundary", err)
	}
	if _, err := s.SaveDraft(t.Context(), "admin", "beta", "health", value, 0); err == nil {
		t.Fatal("target environment suite accepted isolated editor")
	}
	if _, err := s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "once", updated.Revision); err == nil {
		t.Fatal("idempotency key accepted a different draft revision")
	}
	page, err := s.Drafts(t.Context(), "admin", "alpha", "health", "", 1)
	if err != nil || len(page.Items) != 1 || page.Items[0].Revision != updated.Revision {
		t.Fatal("saved source missing from bounded list", err)
	}
	for _, permission := range []string{"reader"} {
		if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: permission, Name: permission}, 0); err != nil {
			t.Fatal(err)
		}
		if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: permission}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.SaveDraft(t.Context(), permission, "alpha", "health", value, updated.Revision); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("reader changed source", err)
		}
	}
}

func TestPlaywrightEditorDraftBoundsAndInvalidSourceRecovery(t *testing.T) {
	s, _, _ := preparedService(t)
	s.BundleRunners = []string{"playwright"}
	for _, path := range []string{"../escape.ts", "package.json", "PACKAGE-LOCK.JSON", "node_modules/fake.ts", "teststudio-suite.json", "checks/welcome.spec.ts/child"} {
		value := sources.StarterDraft()
		value.ID = "invalid"
		value.Files = append(value.Files, sources.EditorFile{Path: path, Content: "source"})
		if _, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, 0); err == nil {
			t.Fatal("invalid file accepted", path)
		}
	}
	value := sources.StarterDraft()
	value.ID = "recoverable"
	value.Files[1].Content = "unfinished JSON"
	row, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, 0)
	if err != nil {
		t.Fatal("unfinished source must remain saveable", err)
	}
	if _, err := s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "invalid", row.Revision); err == nil {
		t.Fatal("invalid catalog was queued")
	}
	page, err := s.ListJobs(t.Context(), "admin", "alpha", "health", "", 30)
	if err != nil || len(page.Items) != 0 {
		t.Fatal("invalid preparation changed queue", err)
	}
}
