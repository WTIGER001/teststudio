package sourcebundle_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type entry struct {
	name, content string
	mode          os.FileMode
}

func archive(t *testing.T, entries []entry) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, item := range entries {
		header := &zip.FileHeader{Name: item.name, Method: zip.Store}
		if item.mode != 0 {
			header.SetMode(item.mode)
		}
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(item.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
func bundle(t *testing.T) []entry {
	t.Helper()
	manifest := suitespec.Manifest{Schema: 1, Suite: "totals", Type: "unit", Runner: "go", ModuleRoot: "module", Tests: suitespec.Tests{Packages: []string{"./checks"}}, Catalog: "catalog/totals.json", BuildProfile: "go-module"}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "BUNDLE-001", Title: "Compute a total", Category: "Unit", Type: "unit", Runner: "go", State: "enabled", Safety: "safe", Rationale: "Computes in memory", TimeoutSeconds: 10, Source: "module/checks/total_test.go", Selector: "./checks::TestTotal"}}}
	manifestJSON, _ := json.Marshal(manifest)
	catalogJSON, _ := json.Marshal(document)
	return []entry{{name: "suites/totals.json", content: string(manifestJSON)}, {name: "catalog/totals.json", content: string(catalogJSON)}, {name: "module/go.mod", content: "module totals\n\ngo 1.26.2\n"}, {name: "module/checks/total_test.go", content: "package checks\nimport \"testing\"\nfunc TestTotal(t *testing.T) { if 2+2 != 4 { t.Fatal(\"total\") } }\n"}}
}
func TestBundleSelectsManifestWithoutDiscardingRepositoryContext(t *testing.T) {
	entries := append(bundle(t), entry{name: "shared/value.txt", content: "shared context", mode: 0700})
	data := archive(t, entries)
	snapshot, err := sourcebundle.Validate(data, "suites/totals.json", "totals")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Manifest.ModuleRoot != "module" || len(snapshot.Files) != 5 || !snapshot.Executable["shared/value.txt"] || snapshot.Catalog.Tests[0].ID != "BUNDLE-001" || !strings.HasPrefix(snapshot.Digest, "sha256:") {
		t.Fatal("bundle context or manifest selection was lost")
	}
	altered := append(entries, entry{name: "suites/other.json", content: "another suite"})
	second, err := sourcebundle.Validate(archive(t, altered), "suites/totals.json", "totals")
	if err != nil || second.Digest == snapshot.Digest {
		t.Fatal("digest ignored shared repository changes", err)
	}
	if _, err := sourcebundle.Validate(data, "suites/totals.json", "other"); err == nil {
		t.Fatal("bundle crossed destination suite")
	}
}
func TestBundleRejectsUnsafeAndAmbiguousEntries(t *testing.T) {
	for _, item := range []entry{{name: "../outside"}, {name: "/absolute"}, {name: "module\\outside"}, {name: "C:drive"}, {name: "./dot"}, {name: "module//double"}, {name: "module/link", content: "../outside", mode: os.ModeSymlink | 0777}, {name: "module/device", mode: os.ModeDevice | 0600}, {name: "MODULE/GO.MOD", content: "case collision"}, {name: "module", content: "file blocks directory"}, {name: "module/go.mod", content: "duplicate"}} {
		t.Run(item.name, func(t *testing.T) {
			if _, err := sourcebundle.Validate(archive(t, append(bundle(t), item)), "suites/totals.json", "totals"); err == nil {
				t.Fatal("unsafe ZIP was accepted")
			}
		})
	}
}
func TestBundleRejectsCorruptionAndMissingContext(t *testing.T) {
	entries := bundle(t)
	data := archive(t, entries)
	needle := []byte("module totals")
	offset := bytes.Index(data, needle)
	if offset < 0 {
		t.Fatal("fixture not stored")
	}
	data[offset] = 'M'
	if _, err := sourcebundle.Validate(data, "suites/totals.json", "totals"); err == nil {
		t.Fatal("corrupt ZIP passed CRC validation")
	}
	for _, index := range []int{0, 1, 2, 3} {
		without := append([]entry{}, entries[:index]...)
		without = append(without, entries[index+1:]...)
		if _, err := sourcebundle.Validate(archive(t, without), "suites/totals.json", "totals"); err == nil {
			t.Fatal("required suite context was missing")
		}
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	_, err := writer.CreateRaw(&zip.FileHeader{Name: "large.txt", Method: zip.Store, UncompressedSize64: sourcebundle.FileLimit + 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := sourcebundle.Validate(output.Bytes(), "suites/totals.json", "totals"); err == nil {
		t.Fatal("oversized declared ZIP entry passed")
	}
}

func TestRepositoryArchiveWrapperKeepsSourceDigestAndRejectsOtherRoots(t *testing.T) {
	original := bundle(t)
	wrapped := make([]entry, 0, len(original))
	for _, file := range original {
		file.name = "repository-root/" + file.name
		wrapped = append(wrapped, file)
	}
	data := archive(t, wrapped)
	root, err := sourcebundle.WrappedRoot(data)
	if err != nil || root != "repository-root" {
		t.Fatal("repository wrapper missing", err)
	}
	snapshot, err := sourcebundle.ValidateRoot(data, "suites/totals.json", "totals", root)
	if err != nil || snapshot.Files["module/go.mod"] == nil || snapshot.ArchiveRoot != root {
		t.Fatal("selected repository context missing", err)
	}
	for _, file := range []entry{{name: "other-root/file.txt"}, {name: "repository-root/../outside"}} {
		if _, err := sourcebundle.ValidateRoot(archive(t, append(wrapped, file)), "suites/totals.json", "totals", root); err == nil {
			t.Fatal("repository wrapper admitted another context")
		}
	}
	if _, err := sourcebundle.WrappedRoot(archive(t, append(wrapped, entry{name: "other-root/file.txt"}))); err == nil {
		t.Fatal("multiple repository roots accepted")
	}
}
