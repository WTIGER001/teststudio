package sources_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func retainedBrowserSource(t *testing.T, additions map[string][]byte) (*sources.Service, storage.Record, []byte) {
	t.Helper()
	s, _, _ := preparedService(t)
	s.BundleRunners = []string{"playwright"}
	if err := (&projects.Service{Store: s.Store, Auth: s.Auth}).SaveSuite(t.Context(), "admin", "alpha", projects.Suite{ID: "browser-bundles", Name: "Browser", Applications: []string{"app"}, RequiresTarget: false}, 0); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "selftest", "playwright", "fixtures", "bundle-browser.zip"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := sourcebundle.Validate(data, "suites/browser.json", "browser-bundles")
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range additions {
		snapshot.Files[name] = content
	}
	for _, name := range snapshot.Paths() {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0644)
		if name == "scripts/fixture.sh" {
			header.SetMode(0755)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(snapshot.Files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data = buffer.Bytes()
	row, err := s.EnqueueBundle(t.Context(), "admin", "alpha", "browser-bundles", "source", "browser.zip", "suites/browser.json", data)
	if err != nil {
		t.Fatal(err)
	}
	return s, row, data
}

func TestPlaywrightEditorRetainedSourcePreservesContextAndOriginalBytes(t *testing.T) {
	s, original, data := retainedBrowserSource(t, map[string][]byte{"scripts/fixture.sh": []byte("#!/bin/sh\nexit 0\n"), "application/greeting.ts": []byte("export const label = 'Welcome'\n")})
	value, err := s.EditorSource(t.Context(), "admin", "alpha", "browser-bundles", original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if value.ManifestPath != "suites/browser.json" || value.Files[0].Path != "browser/checks/greeting.spec.ts" {
		t.Fatal("import changed source layout or default file")
	}
	before, err := sourcebundle.Validate(data, value.ManifestPath, "browser-bundles")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range value.Files {
		if file.Content != string(before.Files[file.Path]) || file.Executable != before.Executable[file.Path] {
			t.Fatal("import changed source bytes or executable mode", file.Path)
		}
	}
	value.Files[0].Content += "\nexport const edited = true\n"
	saved, err := s.SaveDraft(t.Context(), "admin", "alpha", "browser-bundles", value, 0)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := s.PrepareDraft(t.Context(), "admin", "alpha", "browser-bundles", value.ID, "edited", saved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := s.ReadBundle(t.Context(), "admin", "alpha", "browser-bundles", queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	after, err := sourcebundle.Validate(bundle.Content, bundle.Manifest, "browser-bundles")
	if err != nil {
		t.Fatal(err)
	}
	if after.Manifest.ModuleRoot != "browser" || after.Manifest.Catalog != "catalog/browser.json" || len(after.Files) != len(before.Files) {
		t.Fatal("preparation flattened or lost source context")
	}
	for _, file := range value.Files {
		if string(after.Files[file.Path]) != file.Content || after.Executable[file.Path] != file.Executable {
			t.Fatal("preparation changed editor source", file.Path)
		}
	}
	retained, err := s.ReadBundle(t.Context(), "admin", "alpha", "browser-bundles", original.ID)
	if err != nil || !bytes.Equal(data, retained.Content) {
		t.Fatal("editor rewrote original source", err)
	}
	if _, err := s.EditorSource(t.Context(), "admin", "beta", "browser-bundles", original.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("retained source crossed project", err)
	}
	if err := s.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "unassigned", Name: "Unassigned"}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditorSource(t.Context(), "unassigned", "alpha", "browser-bundles", original.ID); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("source without read authority opened", err)
	}
	if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "unassigned"}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditorSource(t.Context(), "unassigned", "alpha", "browser-bundles", original.ID); err != nil {
		t.Fatal("authorized reader could not inspect source", err)
	}
	row, err := s.Store.Get(t.Context(), "source-bundles/alpha", original.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := storage.Decode[sources.Bundle](row)
	stored.ExpiresAt = time.Now().Add(-time.Minute)
	change, _ := storage.Put(row.Bucket, row.ID, "alpha", row.Revision, stored)
	if err := s.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditorSource(t.Context(), "admin", "alpha", "browser-bundles", original.ID); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatal("expired source opened", err)
	}
}

func TestPlaywrightEditorRetainedSourceRejectsLossyCopies(t *testing.T) {
	for _, item := range []struct {
		name    string
		data    []byte
		message string
	}{
		{"application/image.bin", []byte{0, 1, 2}, "UTF-8"},
		{"application/large.txt", bytes.Repeat([]byte("x"), (256<<10)+1), "editor limits"},
	} {
		t.Run(item.name, func(t *testing.T) {
			s, original, data := retainedBrowserSource(t, map[string][]byte{item.name: item.data})
			if _, err := s.EditorSource(t.Context(), "admin", "alpha", "browser-bundles", original.ID); err == nil || !strings.Contains(err.Error(), item.message) {
				t.Fatal("lossy import accepted", err)
			}
			retained, err := s.ReadBundle(t.Context(), "admin", "alpha", "browser-bundles", original.ID)
			if err != nil || !bytes.Equal(retained.Content, data) {
				t.Fatal("rejected import changed original")
			}
			drafts, err := s.Drafts(t.Context(), "admin", "alpha", "browser-bundles", "", 30)
			if err != nil || len(drafts.Items) != 0 {
				t.Fatal("opening source unexpectedly saved a draft")
			}
		})
	}
}
