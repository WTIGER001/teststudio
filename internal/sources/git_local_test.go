package sources_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestGitUntaggedRepositoryOnboardsAndRunsPinnedSource(t *testing.T) {
	runtime := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE")
	if runtime == "" {
		t.Skip("set TESTSTUDIO_BROWSER_RUNTIME_IMAGE to the updated cached Playwright runtime")
	}
	s, c, p, _ := repositorySetup(t, false, false)
	root := t.TempDir()
	builder := &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "contexts")}
	inspector := &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true}
	s.BundleRunners = []string{"playwright"}
	s.Builders = map[string]preparation.ArtifactBuilder{"playwright": builder}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"playwright": inspector}
	s.RecoveryInspectors = s.Inspectors
	files := map[string][]byte{"application/server.txt": []byte("application source is retained beside its tests"), "browser/playwright.config.ts": []byte("export default { testDir: '.', projects: [{ name: 'chromium', use: { browserName: 'chromium' } }] }"), "browser/check.spec.ts": []byte("import { test, expect } from '@playwright/test'\ntest.beforeAll(() => { if (!JSON.parse(process.env.TESTSTUDIO_WORK_ITEM || '{}').test) throw new Error('Discovery executed a hook') })\ntest.describe('Proof', () => { test('Compute a value', async () => { await test.step('Calculate', () => expect(2+3).toBe(5)) }); test('Must remain unselected', () => { throw new Error('Unselected test executed') }) })\n")}
	for _, name := range []string{"package.json", "package-lock.json"} {
		data, err := os.ReadFile("../../tests/kubernetes/suites/playwright/browser/" + name)
		if err != nil {
			t.Fatal(err)
		}
		files["browser/"+name] = data
	}
	makeArchive := func() []byte {
		var out bytes.Buffer
		z := zip.NewWriter(&out)
		for name, value := range files {
			f, err := z.Create("repository-root/" + name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.Write(value); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	p.archive = makeArchive()
	row, err := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	if err != nil {
		t.Fatal(err)
	}
	registered, _ := storage.Decode[sources.RepositorySource](row)
	config := registered.Configuration
	config.AutoActivate = true
	config.Bindings = []sources.RepositoryBinding{{Suite: "health", Manifest: "testing/website.json", Runner: "playwright", Profile: "playwright-node", Discover: true, ModuleRoot: "browser", Config: "playwright.config.ts", Project: "chromium"}}
	if err = s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", config, row.Revision); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		page, err := s.Store.List(ctx, sources.JobBucket("alpha", "health"), "alpha", "", 100)
		if err != nil {
			t.Error(err)
			return
		}
		for _, row := range page.Items {
			job, err := storage.Decode[sources.Job](row)
			if err != nil {
				t.Error(err)
				continue
			}
			if job.Reference != nil {
				if err := inspector.CleanupInspection(ctx, preparation.InspectionRequest{Image: job.Image, Suite: "health", Reference: *job.Reference}); err != nil {
					t.Error(err)
				}
				if err := builder.CleanupBuild(ctx, *job.Reference); err != nil {
					t.Error(err)
				}
				for _, args := range [][]string{{"container", "ls", "-aq", "--filter", "label=teststudio.run=" + job.ID}, {"volume", "ls", "-q", "--filter", "label=teststudio.run=" + job.ID}} {
					output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
					if err != nil || len(bytes.TrimSpace(output)) != 0 {
						t.Error("preparation resources remained after cleanup")
					}
				}
			}
		}
	})
	prepare := func(key string) {
		sync := syncRepository(t, s, key, false)
		_, job := processBundle(t, s, sync.Preparations["health"])
		if job.State != "ready" {
			t.Fatalf("preparation failed: %s %s", job.Error, job.Log)
		}
		if _, err := s.ProcessRepositoryPublication(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	prepare("discover")
	metadata, err := s.ReadRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health")
	if err != nil {
		t.Fatal(err)
	}
	m, err := storage.Decode[sources.RepositoryMetadata](metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Catalog.Tests) != 2 {
		t.Fatal("untagged discovery lost tests")
	}
	id := ""
	for i := range m.Catalog.Tests {
		test := &m.Catalog.Tests[i]
		if test.State != "disabled" || test.Safety != "needs-review" {
			t.Fatal("import enabled an unreviewed test")
		}
		if test.Title == "Compute a value" {
			test.State = "enabled"
			test.Safety = "safe"
			test.Rationale = "Only computes in memory"
			id = test.ID
		}
	}
	if id == "" {
		encoded, _ := json.Marshal(m.Catalog)
		t.Fatalf("missing selected identity: %s", encoded)
	}
	if err = s.SaveRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", m, metadata.Revision); err != nil {
		t.Fatal(err)
	}
	prepare("reviewed")
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	defer blobs.Close()
	driver := &localcontainer.Executor{Runner: "playwright", ApproveArtifact: s.ApproveArtifact}
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Sources: s, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Playwright: driver, Location: "local", Concurrency: 1, PollInterval: 20 * time.Millisecond}
	ctx, cancel := context.WithCancel(t.Context())
	done := runner.Start(ctx)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			t.Error("run worker did not stop")
		}
	})
	run, err := runner.Create(t.Context(), "admin", "alpha", runs.Request{Suite: "health", IDs: []string{id}, TimeoutSeconds: 90}, "run-pinned", "manual")
	if err != nil {
		t.Fatal(err)
	}
	end := time.Now().Add(100 * time.Second)
	for time.Now().Before(end) {
		current, _, err := runner.Get(t.Context(), "alpha", run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(current.State) {
			if current.Outcome != "passed" || current.Counts.Total != 1 || current.Counts.Passed != 1 || current.Release.Draft.Artifact.Provenance.Commit != p.commit || current.SourceResolvedAt == nil {
				page, _ := s.Store.List(t.Context(), evidence.Bucket(current.ID), "alpha", "", 100)
				for _, row := range page.Items {
					item, _ := storage.Decode[evidence.Artifact](row)
					if item.Kind == "log" {
						_, stream, err := runner.Evidence.Open(t.Context(), "alpha", current.ID, item.ID)
						if err == nil {
							data, _ := io.ReadAll(io.LimitReader(stream, 16000))
							stream.Close()
							t.Logf("fixture runner log: %s", data)
						}
					}
				}
				t.Fatalf("normal untagged run failed: %s %s %s results=%+v", current.State, current.Error, current.Outcome, current.Results)
			}
			t.Logf("run=%s test=%s commit=%s durationMs=%.3f", run.ID, id, p.commit, current.Results[0].DurationMs)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	current, _, _ := runner.Get(t.Context(), "alpha", run.ID)
	if !runs.Terminal(current.State) {
		t.Fatal("run did not finish")
	}
	p.commit = strings.Repeat("b", 40)
	files["browser/check.spec.ts"] = append([]byte("\n"), files["browser/check.spec.ts"]...)
	p.archive = makeArchive()
	if _, err := runner.Create(t.Context(), "admin", "alpha", runs.Request{Suite: "health", IDs: []string{id}}, "changed-source", "manual"); err == nil {
		t.Fatal("new branch tip silently used an old image")
	}
	if processed, err := s.ProcessRepositorySync(t.Context(), "sync-worker"); err != nil || !processed {
		t.Fatal(err)
	}
	sourceRow, _ := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	source, _ := storage.Decode[sources.RepositorySource](sourceRow)
	syncRow, _ := s.ReadRepositorySync(t.Context(), "admin", "alpha", "monorepo", source.LastSync)
	sync, _ := storage.Decode[sources.RepositorySync](syncRow)
	_, job := processBundle(t, s, sync.Preparations["health"])
	if job.State != "ready" {
		t.Fatalf("updated source setup failed: %s %s", job.Error, job.Log)
	}
	if _, err := s.ProcessRepositoryPublication(t.Context()); err != nil {
		t.Fatal(err)
	}
	next, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, test := range next.Draft.Tests {
		found = found || test.ID == id
	}
	if !found || next.Draft.Artifact.Provenance.Commit != p.commit || current.Release.Draft.Artifact.Provenance.Commit == p.commit {
		t.Fatal("new source lost identity or changed retained history")
	}
}
