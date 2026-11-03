package localimage_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestBuilderAssemblesOnlyControlledOfflineInstructions(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "image.json")
	runtime := "sha256:" + strings.Repeat("a", 64)
	ref := execution.Reference{ID: "job", Project: "alpha", Installation: "preparation-installation"}
	digest := "sha256:" + strings.Repeat("b", 64)
	builder := &localimage.Builder{RuntimeImage: runtime, TempDirectory: root, Command: func(ctx context.Context, bin string, args ...string) *exec.Cmd {
		if bin != "docker" {
			t.Fatal("unexpected command")
		}
		command := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestBuilderCommandProcess$", "--"}, args...)...)
		command.Env = append(os.Environ(), "TESTSTUDIO_BUILD_HELPER=1", "TESTSTUDIO_BUILD_STATE="+state)
		return command
	}}
	snapshot := sourcebundle.Snapshot{Digest: digest, Manifest: suitespec.Manifest{Schema: 1, Suite: "totals", Runner: "go", BuildProfile: "go-module"}, Files: map[string][]byte{"module/checks/total_test.go": []byte("new independent source"), "Dockerfile": []byte("RUN unwanted-command"), ".dockerignore": []byte("*")}}
	result, err := builder.Build(t.Context(), preparation.BuildRequest{Snapshot: snapshot, Origin: suitespec.Origin{SnapshotDigest: digest}, Reference: ref})
	if err != nil {
		t.Fatal(err)
	}
	if result.Image != "sha256:"+strings.Repeat("c", 64) || result.SourceDigest != digest || !result.LogTruncated {
		t.Fatal("build result lost image provenance or bounded diagnostics")
	}
	if _, err := builder.Build(t.Context(), preparation.BuildRequest{Snapshot: snapshot, Origin: suitespec.Origin{SnapshotDigest: digest}, Reference: ref}); err == nil {
		t.Fatal("build reused an unrecovered image reference")
	}
	foreign := ref
	foreign.Project = "beta"
	if err := builder.CleanupBuild(t.Context(), foreign); err == nil {
		t.Fatal("cleanup crossed project ownership")
	}
	if err := builder.CleanupBuild(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("build image reference survived cleanup")
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatal("build context survived cleanup", err)
	}
}
func TestBuilderCommandProcess(t *testing.T) {
	if os.Getenv("TESTSTUDIO_BUILD_HELPER") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	args = args[1:]
	state := os.Getenv("TESTSTUDIO_BUILD_STATE")
	fail := func(message string) { os.Stderr.WriteString(message); os.Exit(1) }
	switch args[0] + " " + args[1] {
	case "image inspect":
		if strings.HasPrefix(args[2], "sha256:") {
			os.Stdout.WriteString(`{"id":"` + args[2] + `","runtime":"go1.26.8-linux-source-v1","kind":"source-base/v1","onBuild":[]}`)
		} else {
			data, err := os.ReadFile(state)
			if err != nil {
				fail("No such image")
			}
			os.Stdout.Write(data)
		}
	case "build --pull=false":
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--network=none") || !strings.Contains(joined, "--no-cache") {
			fail("unbounded build policy")
		}
		directory := args[len(args)-1]
		dockerfile, err := os.ReadFile(filepath.Join(directory, "Dockerfile"))
		if err != nil {
			fail("missing controlled Dockerfile")
		}
		if strings.Contains(string(dockerfile), "RUN ") || !strings.Contains(string(dockerfile), "COPY --chown=1000:1000 suite/ /suite/") {
			fail("uploaded code controls the build")
		}
		file, err := os.ReadFile(filepath.Join(directory, "suite/module/checks/total_test.go"))
		if err != nil || string(file) != "new independent source" {
			fail("source context missing")
		}
		origin, err := os.ReadFile(filepath.Join(directory, "suite/teststudio-origin.json"))
		if err != nil {
			fail("server provenance missing")
		}
		var parsed suitespec.Origin
		if json.Unmarshal(origin, &parsed) != nil {
			fail("invalid provenance")
		}
		os.WriteFile(state, []byte(`{"id":"sha256:`+strings.Repeat("c", 64)+`","project":"alpha","installation":"preparation-installation","preparation":"job","source":"`+parsed.SnapshotDigest+`"}`), 0600)
		os.Stdout.WriteString(strings.Repeat("build output\n", 30000))
	case "image rm":
		if err := os.Remove(state); err != nil {
			fail("cleanup failed")
		}
	default:
		fail("unexpected docker operation")
	}
	os.Exit(0)
}

func TestBuilderRecoveryRemovesOnlyItsPrivateInterruptedContext(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "absent-image.json")
	ref := execution.Reference{ID: "interrupted", Project: "alpha", Installation: "preparation-installation"}
	sum := sha256.Sum256([]byte(ref.Installation + "\x00" + ref.Project + "\x00" + ref.ID))
	directory := filepath.Join(root, "teststudio-build-"+hex.EncodeToString(sum[:]))
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	owner, _ := json.Marshal(ref)
	if err := os.WriteFile(filepath.Join(directory, "owner.json"), owner, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "source.txt"), []byte("private interrupted context"), 0600); err != nil {
		t.Fatal(err)
	}
	builder := &localimage.Builder{TempDirectory: root, Command: func(ctx context.Context, bin string, args ...string) *exec.Cmd {
		command := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestBuilderCommandProcess$", "--"}, args...)...)
		command.Env = append(os.Environ(), "TESTSTUDIO_BUILD_HELPER=1", "TESTSTUDIO_BUILD_STATE="+state)
		return command
	}}
	wrong := ref
	wrong.Project = "beta"
	foreign, _ := json.Marshal(wrong)
	if err := os.WriteFile(filepath.Join(directory, "owner.json"), foreign, 0600); err != nil {
		t.Fatal(err)
	}
	if err := builder.CleanupBuild(t.Context(), ref); err == nil {
		t.Fatal("recovery removed another context owner")
	}
	if err := os.WriteFile(filepath.Join(directory, "owner.json"), owner, 0600); err != nil {
		t.Fatal(err)
	}
	if err := builder.CleanupBuild(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("interrupted source context survived recovery")
	}
}
