package sources_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestRepositorySnapshotRunsTwoSuitesBesideApplicationSource(t *testing.T) {
	runtime := os.Getenv("TESTSTUDIO_SOURCE_RUNTIME_IMAGE")
	if runtime == "" {
		t.Skip("set TESTSTUDIO_SOURCE_RUNTIME_IMAGE to a cached immutable local Go source runtime")
	}
	s, c, provider, _ := repositorySetup(t, true, false)
	files := map[string][]byte{
		"application/go.mod":        []byte("module source.local/repository\n\ngo 1.24\n"),
		"application/calc/total.go": []byte("package calc\nfunc Add(a, b int) int { return a+b }\n"),
	}
	for _, suite := range []string{"health", "math"} {
		pkg, testName, id := "checks", "TestHealth", "REPO-HEALTH-001"
		if suite == "math" {
			pkg, testName, id = "morechecks", "TestMath", "REPO-MATH-001"
		}
		source := pkg + "/total_test.go"
		files["application/"+source] = []byte("package " + pkg + "\nimport (\"testing\"; \"source.local/repository/calc\")\nfunc " + testName + "(t *testing.T) { if calc.Add(2,3) != 5 { t.Fatal(\"unexpected total\") } }\n")
		manifest := suitespec.Manifest{Schema: 1, Suite: suite, Type: "unit", Runner: "go", ModuleRoot: "application", Tests: suitespec.Tests{Packages: []string{"./" + pkg}}, Catalog: "catalogs/" + suite + ".json", BuildProfile: "go-module"}
		document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: id, Title: suite + " application check", Category: "Unit", Type: "unit", Runner: "go", State: "enabled", Safety: "safe", Rationale: "Computes in memory", TimeoutSeconds: 30, Source: "application/" + source, Selector: "./" + pkg + "::" + testName}}}
		files["suites/"+suite+".json"], _ = json.Marshal(manifest)
		files["catalogs/"+suite+".json"], _ = json.Marshal(document)
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, data := range files {
		entry, err := writer.Create("repository-root/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	provider.archive = archive.Bytes()
	root := t.TempDir()
	builder := &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "contexts")}
	inspector := &localcontainer.Executor{Runner: "go", AllowProjectImages: true, RequireProject: true}
	s.Builders = map[string]preparation.ArtifactBuilder{"go": builder}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"go": inspector}
	s.RecoveryInspectors = s.Inspectors
	sync := syncRepository(t, s, "local-two-suites", false)
	if sync.State != "completed" || len(sync.Preparations) != 2 || provider.fetches != 1 {
		t.Fatalf("sync: %+v", sync)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for suite, id := range sync.Preparations {
			row, err := s.ReadJob(ctx, "admin", "alpha", suite, id)
			if err != nil {
				t.Error(err)
				continue
			}
			job, err := storage.Decode[sources.Job](row)
			if err != nil {
				t.Error(err)
				continue
			}
			if job.Reference != nil {
				if err := inspector.CleanupInspection(ctx, preparation.InspectionRequest{Image: job.Image, Suite: suite, Reference: *job.Reference}); err != nil {
					t.Error(err)
				}
				if err := builder.CleanupBuild(ctx, *job.Reference); err != nil {
					t.Error(err)
				}
				for _, check := range [][]string{{"container", "ls", "-aq", "--filter", "label=teststudio.run=" + id}, {"volume", "ls", "-q", "--filter", "label=teststudio.run=" + id}, {"image", "ls", "-q", "--filter", "label=teststudio.execution=" + job.Reference.ID}} {
					output, err := exec.CommandContext(ctx, "docker", check...).CombinedOutput()
					if err != nil || strings.TrimSpace(string(output)) != "" {
						t.Errorf("preparation cleanup %v: %s %v", check, output, err)
					}
				}
			}
		}
	})
	for range 2 {
		if done, err := s.ProcessBuildNext(t.Context(), "local-builder"); err != nil || !done {
			t.Fatal(done, err)
		}
	}
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	driver := &localcontainer.Executor{Runner: "go", ApproveArtifact: s.ApproveArtifact}
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Go: driver, Location: "local", Concurrency: 1, PollInterval: 20 * time.Millisecond}
	ctx, cancel := context.WithCancel(t.Context())
	done := runner.Start(ctx)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("run worker did not stop")
		}
	})
	for _, suite := range []string{"health", "math"} {
		row, err := s.ReadJob(t.Context(), "admin", "alpha", suite, sync.Preparations[suite])
		if err != nil {
			t.Fatal(err)
		}
		job, _ := storage.Decode[sources.Job](row)
		if job.State != "ready" {
			t.Fatalf("%s preparation: %+v", suite, job)
		}
		release, err := s.Publish(t.Context(), "admin", "alpha", suite, job.Upload, 1, true)
		if err != nil {
			t.Fatal(err)
		}
		if release.Draft.Artifact.Provenance.Commit != provider.commit {
			t.Fatal("release lost commit")
		}
		run, err := runner.Create(t.Context(), "admin", "alpha", runs.Request{Suite: suite, TimeoutSeconds: 90}, "run-"+suite, "manual")
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(100 * time.Second)
		for time.Now().Before(deadline) {
			current, _, err := runner.Get(t.Context(), "alpha", run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if runs.Terminal(current.State) {
				if current.State != "completed" || current.Outcome != "passed" || current.Counts.Total != 1 || current.Counts.Passed != 1 || current.EvidenceState != "complete" {
					t.Fatalf("normal run: %+v", current)
				}
				t.Logf("suite=%s commit=%s image=%s run=%s outcome=%s count=%d evidence=%s", suite, provider.commit, job.Image, run.ID, current.Outcome, current.Counts.Total, current.EvidenceState)
				break
			}
			time.Sleep(30 * time.Millisecond)
		}
		current, _, _ := runner.Get(t.Context(), "alpha", run.ID)
		if !runs.Terminal(current.State) {
			t.Fatal("run deadline exceeded")
		}
		for _, kind := range []string{"container", "volume"} {
			args := []string{kind, "ls", "-q", "--filter", "label=teststudio.run=" + run.ID}
			if kind == "container" {
				args = append(args, "-a")
			}
			output, err := exec.CommandContext(t.Context(), "docker", args...).CombinedOutput()
			if err != nil || strings.TrimSpace(string(output)) != "" {
				t.Fatalf("run cleanup: %s %v", output, err)
			}
		}
	}
}
