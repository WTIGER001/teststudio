package registryimage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/preparation"
)

func TestRegistryAssemblyRejectsPreparedBasesAndUnpreparedDependencies(t *testing.T) {
	for _, name := range []string{"prepared-base", "build-trigger", "extra-dependency", "whiteout", "origin-mismatch"} {
		t.Run(name, func(t *testing.T) {
			f := fixture(t, "playwright", func(base map[string]any) {
				if name == "prepared-base" {
					base["config"].(map[string]any)["Labels"].(map[string]string)["teststudio.project"] = "foreign"
				}
				if name == "build-trigger" {
					base["config"].(map[string]any)["OnBuild"] = []string{"RUN must-not-execute"}
				}
			})
			input := buildInput(t, "playwright")
			switch name {
			case "extra-dependency":
				input.Snapshot.Files["checks/package.json"] = []byte(`{"devDependencies":{"@playwright/test":"1.64.1","extra-code":"1.0.0"}}`)
			case "whiteout":
				input.Snapshot.Files[".wh..wh..opq"] = nil
			case "origin-mismatch":
				input.Origin.SnapshotDigest = "sha256:" + strings.Repeat("e", 64)
			}
			builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
			if result, err := builder.Build(t.Context(), input); err == nil || result.Image != "" {
				t.Fatal(name, result, err)
			}
			f.mu.Lock()
			posts := f.posts
			f.mu.Unlock()
			if posts != 0 {
				t.Fatal("invalid source opened a registry upload")
			}
		})
	}
}

func TestRegistryCleanupRecoversAnInterruptedSessionCheckpoint(t *testing.T) {
	f := fixture(t, "go", nil)
	input := buildInput(t, "go")
	builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
	journal, parent, err := builder.reserve(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Opening("sha256:" + strings.Repeat("d", 64)); err != nil {
		t.Fatal(err)
	}
	pending := journal.state
	pending.Upload = f.server.URL + repositoryPath + "blobs/uploads/pending?_state=private-upload-canary"
	data, _ := json.Marshal(pending)
	if err := journal.root.WriteFile("state.pending", data, 0600); err != nil {
		t.Fatal(err)
	}
	journal.root.Close()
	parent.Close()
	f.mu.Lock()
	f.sessions["pending"] = true
	f.mu.Unlock()
	if err := builder.CleanupBuild(t.Context(), input.Reference); err != nil {
		t.Fatal("interrupted upload checkpoint prevented owned cleanup", err)
	}
	f.mu.Lock()
	remaining := len(f.sessions)
	f.mu.Unlock()
	if remaining != 0 {
		t.Fatal("interrupted registry session remained active")
	}
}

func TestRegistryCleanupRejectsForeignOwnershipAndDirectoryLinks(t *testing.T) {
	f := fixture(t, "go", nil)
	input := buildInput(t, "go")
	builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
	journal, parent, err := builder.reserve(input)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	defer journal.root.Close()
	journal.state.Owner.Project = "foreign"
	if err := journal.save(); err != nil {
		t.Fatal(err)
	}
	if err := builder.CleanupBuild(t.Context(), input.Reference); err == nil {
		t.Fatal("foreign owner was accepted")
	}
	journal.state.Owner = input.Reference
	if err := journal.save(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Opening("sha256:" + strings.Repeat("d", 64)); err != nil {
		t.Fatal(err)
	}
	if err := journal.Opened("https://foreign.invalid/private-upload-canary"); err != nil {
		t.Fatal(err)
	}
	if err := builder.CleanupBuild(t.Context(), input.Reference); !errors.Is(err, preparation.ErrCleanupIncomplete) || strings.Contains(err.Error(), "private-upload-canary") {
		t.Fatal("cleanup accepted or exposed a foreign upload location", err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(builder.Directory, linked); err != nil {
		t.Fatal(err)
	}
	builder.Directory = linked
	if err := builder.CleanupBuild(t.Context(), input.Reference); err == nil {
		t.Fatal("cleanup accepted a symbolic recovery directory")
	}
}
