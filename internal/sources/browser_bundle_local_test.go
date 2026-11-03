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

func TestBrowserSourceBundleBuildsPublishesAndRunsWithRuntimeDependencies(t *testing.T) {
	runtime := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE")
	if runtime == "" {
		t.Skip("set TESTSTUDIO_BROWSER_RUNTIME_IMAGE to a cached immutable local Playwright source runtime")
	}
	s, c, _ := bundleSetup(t)
	manifest := suitespec.Manifest{Schema: 1, Suite: "health", Type: "browser", Runner: "playwright", ModuleRoot: "browser", Tests: suitespec.Tests{Config: "playwright.config.ts", Project: "chromium"}, Catalog: "catalog/browser.json", BuildProfile: "playwright-node"}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "BROWSER-BUNDLE-001", Title: "Render application greeting", Category: "Interface", Type: "browser", Runner: "playwright", State: "enabled", Safety: "safe", Rationale: "Renders local source in an isolated browser", TimeoutSeconds: 30, Source: "browser/checks/greeting.spec.ts", Selector: "@BROWSER-BUNDLE-001"}}}
	metadata, _ := json.Marshal(document)
	declaration, _ := json.Marshal(manifest)
	files := map[string][]byte{
		"suites/browser.json": declaration, "catalog/browser.json": metadata,
		"application/greeting.ts":         []byte("export function greeting(name: string) { return `<h1>Hello ${name}</h1>` }\n"),
		"browser/package.json":            []byte(`{"name":"independent-browser-bundle","type":"module","private":true,"devDependencies":{"@playwright/test":"1.64.1"}}`),
		"browser/package-lock.json":       []byte(`{"lockfileVersion":3,"packages":{"":{"devDependencies":{"@playwright/test":"1.64.1"}},"node_modules/@playwright/test":{"version":"1.64.1"},"node_modules/playwright":{"version":"1.64.1"},"node_modules/playwright-core":{"version":"1.64.1"}}}`),
		"browser/playwright.config.ts":    []byte("import { defineConfig } from '@playwright/test'\nexport default defineConfig({testDir:'./checks', outputDir:'/tmp/browser-output', projects:[{name:'chromium',use:{browserName:'chromium'}}]})\n"),
		"browser/checks/greeting.spec.ts": []byte("import { test, expect } from '@playwright/test'\nimport { greeting } from '../../application/greeting'\ntest('renders the application greeting', {tag:'@BROWSER-BUNDLE-001'}, async ({page}) => { await page.setContent(greeting('TestStudio')); await expect(page.getByRole('heading', {name:'Hello TestStudio'})).toBeVisible() })\n"),
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, data := range files {
		entry, err := writer.Create(name)
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
	root := t.TempDir()
	builder := &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "contexts")}
	inspector := &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true}
	s.BundleRunners = []string{"go", "playwright"}
	s.Builders = map[string]preparation.ArtifactBuilder{"playwright": builder}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"playwright": inspector}
	s.RecoveryInspectors = s.Inspectors
	row, err := s.EnqueueBundle(t.Context(), "admin", "alpha", "health", "browser-upload", "browser.zip", "suites/browser.json", archive.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stored, err := s.ReadJob(ctx, "admin", "alpha", "health", row.ID)
		if err != nil {
			t.Error(err)
			return
		}
		job, _ := storage.Decode[sources.Job](stored)
		if job.Reference != nil {
			if err := inspector.CleanupInspection(ctx, preparation.InspectionRequest{Image: job.Image, Suite: job.Suite, Reference: *job.Reference}); err != nil {
				t.Error(err)
			}
			if err := builder.CleanupBuild(ctx, *job.Reference); err != nil {
				t.Error(err)
			}
		}
	})
	if done, err := s.ProcessBuildNext(t.Context(), "browser-builder"); err != nil || !done {
		t.Fatal(done, err)
	}
	stored, err := s.ReadJob(t.Context(), "admin", "alpha", "health", row.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](stored)
	if job.State != "ready" {
		t.Fatalf("browser preparation: %+v", job)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.Draft.Tests) != 1 || release.Draft.Artifact.Discovered["BROWSER-BUNDLE-001"] != "@BROWSER-BUNDLE-001" {
		t.Fatal("browser catalog mapping changed")
	}
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	driver := &localcontainer.Executor{Runner: "playwright", ApproveArtifact: s.ApproveArtifact}
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Playwright: driver, Location: "local", Concurrency: 1, PollInterval: 20 * time.Millisecond}
	ctx, cancel := context.WithCancel(t.Context())
	done := runner.Start(ctx)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("browser worker did not stop")
		}
	})
	run, err := runner.Create(t.Context(), "admin", "alpha", runs.Request{Suite: "health", TimeoutSeconds: 60}, "browser-run", "manual")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(70 * time.Second)
	for time.Now().Before(deadline) {
		current, _, err := runner.Get(t.Context(), "alpha", run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(current.State) {
			if current.State != "completed" || current.Outcome != "passed" || current.Counts.Total != 1 || current.Counts.Passed != 1 || current.EvidenceState != "complete" {
				t.Fatalf("normal browser run: %+v", current)
			}
			t.Logf("image=%s run=%s outcome=%s count=%d evidence=%s", job.Image, current.ID, current.Outcome, current.Counts.Total, current.EvidenceState)
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	current, _, _ := runner.Get(t.Context(), "alpha", run.ID)
	if !runs.Terminal(current.State) {
		t.Fatal("browser run deadline exceeded")
	}
	for _, id := range []string{row.ID, run.ID} {
		for _, kind := range []string{"container", "volume"} {
			args := []string{kind, "ls", "-q", "--filter", "label=teststudio.run=" + id}
			if kind == "container" {
				args = append(args, "-a")
			}
			output, err := exec.CommandContext(t.Context(), "docker", args...).CombinedOutput()
			if err != nil || strings.TrimSpace(string(output)) != "" {
				t.Fatalf("browser resource cleanup: %s %v", output, err)
			}
		}
	}
}
