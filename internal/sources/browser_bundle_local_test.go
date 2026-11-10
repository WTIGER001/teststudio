package sources_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
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
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestBrowserSourceBundleBuildsPublishesAndRunsWithRuntimeDependencies(t *testing.T) {
	verifyBrowserSourceWorkflow(t, false)
}

func TestPlaywrightEditorLocalStarterBuildsPublishesAndRuns(t *testing.T) {
	verifyBrowserSourceWorkflow(t, true)
}

func TestRecordedWebsiteBuildsPublishesAndRunsThroughGuardedGateway(t *testing.T) {
	if os.Getenv("TESTSTUDIO_NATIVE_RECORDING_SOURCE") == "" || os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE") == "" {
		t.Skip("supply the newly captured local recording and trusted gateway image")
	}
	verifyBrowserSourceWorkflow(t, true, true)
}

func verifyBrowserSourceWorkflow(t *testing.T, editor bool, website ...bool) {
	t.Helper()
	runtime := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE")
	if runtime == "" {
		t.Skip("set TESTSTUDIO_BROWSER_RUNTIME_IMAGE to a cached immutable local Playwright source runtime")
	}
	s, c, _ := bundleSetup(t)
	recorded := len(website) > 0 && website[0]
	var captured recording.Session
	var target string
	if recorded {
		data, err := os.ReadFile(os.Getenv("TESTSTUDIO_NATIVE_RECORDING_SOURCE"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &captured); err != nil {
			t.Fatal(err)
		}
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Fatal(err)
		}
		fixture := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "http://denied.invalid/", 302)
				return
			}
			if r.URL.Path == "/probe" {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("Target API ready"))
				return
			}
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<main><h1>Website fixture</h1><button onclick="document.querySelector('h1').textContent='Ready for review'">Continue</button></main>`))
		}))
		fixture.Listener.Close()
		fixture.Listener = listener
		fixture.Start()
		t.Cleanup(fixture.Close)
		target = fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
		projectService := projects.Service{Store: s.Store, Auth: s.Auth}
		if err := projectService.SaveApplication(t.Context(), "admin", "alpha", projects.Application{ID: "website", Name: "Website fixture"}, 0); err != nil {
			t.Fatal(err)
		}
		if err := projectService.SaveEnvironment(t.Context(), "admin", "alpha", projects.Environment{ID: "website", Name: "Local browser fixture", Application: "app", Stage: "local", Endpoints: map[string]string{"api": target}, Locations: []string{"local"}}, 0); err != nil {
			t.Fatal(err)
		}
		suiteRow, err := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
		if err != nil {
			t.Fatal(err)
		}
		suite, _ := storage.Decode[projects.Suite](suiteRow)
		suite.RequiresTarget = true
		suite.Applications = []string{"app"}
		if err := projectService.SaveSuite(t.Context(), "admin", "alpha", suite, suiteRow.Revision); err != nil {
			t.Fatal(err)
		}
	}
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
	if recorded && os.Getenv("TESTSTUDIO_BROWSER_OUTPUT") != "" {
		root = os.Getenv("TESTSTUDIO_BROWSER_OUTPUT")
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
	}
	builder := &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "contexts")}
	inspector := &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true}
	s.BundleRunners = []string{"go", "playwright"}
	s.Builders = map[string]preparation.ArtifactBuilder{"playwright": builder}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"playwright": inspector}
	s.RecoveryInspectors = s.Inspectors
	expectedID := "BROWSER-BUNDLE-001"
	var row storage.Record
	var err error
	if editor {
		value := sources.StarterDraft()
		if recorded {
			value, err = sources.RecordedDraft(captured)
			if err != nil {
				t.Fatal(err)
			}
			value.Files[0].Content = strings.Replace(value.Files[0].Content, "async ({ page })", "async ({ page, request })", 1)
			closing := strings.LastIndex(value.Files[0].Content, "});")
			if closing < 0 {
				t.Fatal("recorder source has no closing test declaration")
			}
			steps := `  await expect(page.getByRole('heading', { name: 'Website fixture' })).toBeVisible();
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.getByRole('heading', { name: 'Ready for review' })).toBeVisible();
  const probe = await request.get('/probe');
  expect(await probe.text()).toBe('Target API ready');
  const blocked = await request.get('/redirect');
  expect(blocked.status()).toBe(403);
`
			value.Files[0].Content = value.Files[0].Content[:closing] + steps + value.Files[0].Content[closing:]
			var doc catalog.Document
			if err := json.Unmarshal([]byte(value.Files[1].Content), &doc); err != nil {
				t.Fatal(err)
			}
			doc.Tests[0].Capture.Screenshots = "always"
			doc.Tests[0].Capture.Trace = "always"
			encoded, _ := json.Marshal(doc)
			value.Files[1].Content = string(encoded)
		}
		value.ID = "local-editor"
		saved, saveErr := s.SaveDraft(t.Context(), "admin", "alpha", "health", value, 0)
		if saveErr != nil {
			t.Fatal(saveErr)
		}
		row, err = s.PrepareDraft(t.Context(), "admin", "alpha", "health", value.ID, "browser-editor", saved.Revision)
		expectedID = "BROWSER-001"
		if recorded {
			expectedID = captured.TestID
		}
	} else {
		row, err = s.EnqueueBundle(t.Context(), "admin", "alpha", "health", "browser-upload", "browser.zip", "suites/browser.json", archive.Bytes())
	}
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
	if editor && job.Image != "" {
		t.Cleanup(func() {
			output, err := exec.Command("docker", "image", "rm", job.Image).CombinedOutput()
			if err != nil {
				t.Errorf("editor image cleanup: %s %v", output, err)
			}
		})
	}
	if job.State != "ready" {
		t.Fatalf("browser preparation: %+v", job)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.Draft.Tests) != 1 || release.Draft.Artifact.Discovered[expectedID] != "@"+expectedID {
		t.Fatal("browser catalog mapping changed")
	}
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	driver := &localcontainer.Executor{Runner: "playwright", ApproveArtifact: s.ApproveArtifact}
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Playwright: driver, Location: "local", Concurrency: 1, PollInterval: 20 * time.Millisecond}
	if recorded {
		browser := *driver
		browser.Environment = true
		browser.GatewayImage = os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE")
		runner.Browser = &browser
	}
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
	request := runs.Request{Suite: "health", TimeoutSeconds: 60}
	if recorded {
		request.Environment = "website"
	}
	run, err := runner.Create(t.Context(), "admin", "alpha", request, "browser-run", "manual")
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
				t.Fatalf("browser run: state=%s outcome=%s results=%+v evidence=%s gaps=%v", current.State, current.Outcome, current.Results, current.EvidenceState, current.EvidenceErrors)
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
	if recorded && os.Getenv("TESTSTUDIO_BROWSER_OUTPUT") != "" {
		data, _ := json.MarshalIndent(current, "", "  ")
		if err := os.WriteFile(filepath.Join(root, "run.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	resourceIDs := []string{row.ID, run.ID}
	if len(website) > 1 && website[1] {
		scheduled := verifyScheduledWebsiteParity(t, runner, current, root)
		resourceIDs = append(resourceIDs, scheduled.ID)
	}
	for _, id := range resourceIDs {
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
