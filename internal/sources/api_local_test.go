package sources_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/diagnostics"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
	"github.com/wtiger001/teststudio/internal/trends"
)

func TestGuardedAPIWorkspaceBuildsPublishesAndRunsManuallyAndOnSchedule(t *testing.T) {
	testGuardedAPIWorkspace(t, "playwright")
}

func testGuardedAPIWorkspace(t *testing.T, kind string) {
	runtime, gateway := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE"), os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE")
	if kind == "go" {
		runtime = os.Getenv("TESTSTUDIO_GO_RUNTIME_IMAGE")
	}
	if runtime == "" || gateway == "" {
		t.Skip("supply cached request-policy/v1 runtime and gateway images for the local API workflow")
	}
	s, catalogService, _ := bundleSetup(t)
	root := os.Getenv("TESTSTUDIO_API_VERIFY_OUTPUT")
	if root == "" {
		root = t.TempDir()
	}
	boundSecret := "credential-value-" + domain.NewID()
	var hits atomic.Int32
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	fixture := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+boundSecret {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://unapproved.invalid/private-route", 302)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "up", "token": "response-canary", "privateContact": "contact-canary", "echo": boundSecret})
	}))
	fixture.Listener.Close()
	fixture.Listener = listener
	fixture.Start()
	t.Cleanup(fixture.Close)
	projectService := &projects.Service{Store: s.Store, Auth: s.Auth}
	credentialFile := filepath.Join(root, "credential")
	if err := os.WriteFile(credentialFile, []byte(boundSecret), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(credentialFile) })
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	provider, err := localsecrets.New([]localsecrets.Binding{{ID: "api-reader", Project: "alpha", Purpose: secrets.APICredential, File: credentialFile, Origins: []string{origin}, Network: networkguard.Private}})
	if err != nil {
		t.Fatal(err)
	}
	projectService.Secrets = provider
	if err := projectService.SaveApplication(t.Context(), "admin", "alpha", projects.Application{ID: "requests", Name: "Owned request fixture"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveEnvironment(t.Context(), "admin", "alpha", projects.Environment{ID: "requests", Name: "Guarded local API", Application: "requests", Stage: "local", Protected: true, Endpoints: map[string]string{"api": origin}, Credentials: map[string]string{"api-login": "api-reader"}}, 0); err != nil {
		t.Fatal(err)
	}
	row, _ := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
	suite, _ := storage.Decode[projects.Suite](row)
	suite.RequiresTarget, suite.Applications = true, []string{"requests"}
	if err := projectService.SaveSuite(t.Context(), "admin", "alpha", suite, row.Revision); err != nil {
		t.Fatal(err)
	}
	builder := &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "contexts")}
	inspector := &localcontainer.Executor{Runner: kind, AllowProjectImages: true, RequireProject: true}
	s.BundleRunners = []string{kind}
	s.Builders, s.RecoveryBuilders = map[string]preparation.ArtifactBuilder{kind: builder}, map[string]preparation.ArtifactBuilder{kind: builder}
	s.Inspectors, s.RecoveryInspectors = map[string]preparation.ArtifactInspector{kind: inspector}, map[string]preparation.ArtifactInspector{kind: inspector}
	draft, err := sources.APIStarterDraft("health")
	if err != nil {
		t.Fatal(err)
	}
	draft.ID = "guarded-api"
	for index, file := range draft.Files {
		switch file.Path {
		case draft.ManifestPath:
			var manifest suitespec.Manifest
			_ = json.Unmarshal([]byte(file.Content), &manifest)
			manifest.Browser.Requests = &apipolicy.Spec{Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/health", ReadOnly: true}, {Method: "GET", PathPrefix: "/redirect", ReadOnly: true}, {Method: "POST", PathPrefix: "/search", ReadOnly: true, MaxBodyBytes: 8}}, MaxRequests: 3, MaxResponseBytes: 65536}
			manifest.Browser.Requests.RedactFields = []string{"privateContact"}
			manifest.Browser.Requests.Authentication = &apipolicy.Authentication{Binding: "api-login", Scheme: "bearer"}
			manifest.RequiredBindings = []string{"api-login"}
			if kind == "go" {
				manifest.Runner = "go"
				manifest.BuildProfile = "go-environment"
				manifest.Tests = suitespec.Tests{Packages: []string{"./checks"}}
				for index := 0; index < 61; index++ {
					manifest.Browser.Requests.Rules = append(manifest.Browser.Requests.Rules, apipolicy.Rule{Method: "GET", PathPrefix: fmt.Sprintf("/reserved/%02d/", index) + strings.Repeat("<", 1900), ReadOnly: true})
				}
			}
			data, _ := json.Marshal(manifest)
			draft.Files[index].Content = string(data)
		case "catalog.json":
			var document catalog.Document
			_ = json.Unmarshal([]byte(file.Content), &document)
			document.Tests[0].Safety, document.Tests[0].Rationale = "safe", "Reads newly created local fixture state and checks denied requests without application mutations"
			document.Tests[0].RequiredBindings = []string{"api-login"}
			if kind == "go" {
				document.Tests[0].Runner = "go"
				document.Tests[0].Source = "checks/api_test.go"
				document.Tests[0].Selector = "./checks::TestGuardedAPIRequests"
			}
			data, _ := json.Marshal(document)
			draft.Files[index].Content = string(data)
		case "checks/api.spec.ts":
			draft.Files[index].Content = `import { test, expect } from '@playwright/test'
import http from 'node:http'
test('Guard API requests', { tag: '@API-001' }, async ({ request }, testInfo) => {
  await test.step('Read the permitted endpoint', async () => {
    const response = await request.get('/health/')
    expect(response.status()).toBe(200)
    expect(await response.json()).toHaveProperty('status', 'up')
    expect(await response.json()).toHaveProperty('echo', '[REDACTED]')
    expect(process.env.TESTSTUDIO_WORK_ITEM).not.toContain('credential-value-')
    expect(process.env.TESTSTUDIO_WORK_ITEM).not.toContain('api-reader')
    await testInfo.attach('response', { body: await response.body(), contentType: 'application/json' })
    process.stdout.write('Authorization: Bearer split-')
    process.stdout.write('canary\n')
    console.log('actual response-canary contact-canary split-canary')
    await testInfo.attach('opaque response', { body: Buffer.from('binary-canary'), contentType: 'application/octet-stream' })
  })
  await test.step('Reject unreviewed methods, paths, overrides and oversized payloads', async () => {
    expect((await request.delete('/health')).status()).toBe(403)
    expect((await request.get('/outside')).status()).toBe(403)
    expect((await request.get('/health', { headers: { 'X-HTTP-Method-Override': 'DELETE' } })).status()).toBe(403)
    expect((await request.post('/search', { data: '123456789' })).status()).toBe(413)
  })
  await test.step('Run an explicitly reviewed read-only POST', async () => {
    expect((await request.post('/search', { data: 'owned' })).status()).toBe(200)
  })
  await test.step('Keep redirects and direct CONNECT requests behind the gateway', async () => {
    const response = await request.get('/redirect', { maxRedirects: 0 })
    expect(response.status()).toBe(302)
    expect(response.headers().location).toBeUndefined()
    expect(await response.text()).not.toContain('private-route')
    const status = await new Promise((resolve, reject) => {
      const connection = http.request(process.env.TESTSTUDIO_API_BASE_URL!, { method: 'CONNECT', path: 'unused.invalid:443' }, response => { response.resume(); resolve(response.statusCode) })
      connection.once('connect', (response, socket) => { socket.destroy(); resolve(response.statusCode) })
      connection.once('error', reject)
      connection.end()
    })
    expect(status).toBe(403)
  })
  await test.step('Stop another request https://example.test/?token=query-canary', async () => {
    expect((await request.get('/health')).status()).toBe(429)
  })
})
`
		}
	}
	var queued storage.Record
	if kind == "go" {
		queued, err = s.EnqueueBundle(t.Context(), "admin", "alpha", "health", "guarded-go-api", "go-api.zip", draft.ManifestPath, guardedGoZIP(t, draft, origin))
	} else {
		saved, saveErr := s.SaveDraft(t.Context(), "admin", "alpha", "health", draft, 0)
		if saveErr != nil {
			t.Fatal(saveErr)
		}
		queued, err = s.PrepareDraft(t.Context(), "admin", "alpha", "health", draft.ID, "guarded-api-source", saved.Revision)
	}
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{queued.ID}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stored, err := s.ReadJob(ctx, "admin", "alpha", "health", queued.ID)
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
		clean := true
		for _, id := range ids {
			for _, kind := range []string{"container", "volume"} {
				args := []string{kind, "ls", "-q", "--filter", "label=teststudio.run=" + id}
				if kind == "container" {
					args = append(args, "-a")
				}
				output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
				if err != nil || strings.TrimSpace(string(output)) != "" {
					t.Error("owned API resources remain", kind, id, string(output), err)
					clean = false
				}
			}
		}
		if job.Image != "" && job.Image != runtime {
			output, err := exec.CommandContext(ctx, "docker", "image", "inspect", job.Image, "--format", "{{.Id}}").CombinedOutput()
			if err == nil {
				if output, err := exec.CommandContext(ctx, "docker", "image", "rm", job.Image).CombinedOutput(); err != nil {
					t.Error("owned API image cleanup", string(output), err)
					clean = false
				}
			} else if !strings.Contains(string(output), "No such image") && !strings.Contains(string(output), "No such object") {
				t.Error("owned API image absence cannot be verified", string(output), err)
				clean = false
			}
		}
		data, _ := json.Marshal(map[string]any{"resourcesAbsent": clean, "ownedRunIds": ids, "fixtureRequests": hits.Load()})
		if err := os.WriteFile(filepath.Join(root, "cleanup.json"), data, 0600); err != nil {
			t.Error(err)
		}
	})
	if done, err := s.ProcessBuildNext(t.Context(), "api-builder"); err != nil || !done {
		t.Fatal(done, err)
	}
	prepared, err := s.ReadJob(t.Context(), "admin", "alpha", "health", queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](prepared)
	if job.State != "ready" {
		t.Fatal("API preparation fails", job.State, job.Error, job.Log)
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	driver := &localcontainer.Executor{Runner: kind, Environment: true, GatewayImage: gateway, ApproveArtifact: s.ApproveArtifact, Secrets: provider}
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: catalogService, Evidence: &evidence.Service{Store: s.Store, Blobs: files}, Browser: driver, Location: "local", Concurrency: 1, PollInterval: 10 * time.Millisecond}
	if kind == "go" {
		runner.Browser = nil
		runner.GoAPI = driver
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := runner.Start(ctx)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("API worker does not stop")
		}
	})
	request := runs.Request{Suite: "health", Environment: "requests", TimeoutSeconds: 60}
	if kind == "go" {
		request.TimeoutSeconds = 180
	}
	manual, err := runner.Create(t.Context(), "admin", "alpha", request, "api-manual", "manual")
	if err != nil {
		t.Fatal(err)
	}
	ids = append(ids, manual.ID)
	clock := time.Now().UTC()
	scheduler := &schedules.Service{Store: s.Store, Auth: s.Auth, Runs: runner, Now: func() time.Time { return clock }}
	if err := scheduler.Save(t.Context(), "admin", "alpha", schedules.Definition{ID: "api-clock", Name: "Guarded API schedule", Enabled: true, Kind: "interval", IntervalSeconds: 5, Timezone: "UTC", Overlap: "allow", Request: request}, 0); err != nil {
		t.Fatal(err)
	}
	scheduleRow, _ := s.Store.Get(t.Context(), schedules.Bucket("alpha"), "api-clock")
	schedule, _ := storage.Decode[schedules.Schedule](scheduleRow)
	clock = *schedule.NextAt
	if _, err := scheduler.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	scheduleRow, _ = s.Store.Get(t.Context(), schedules.Bucket("alpha"), "api-clock")
	schedule, _ = storage.Decode[schedules.Schedule](scheduleRow)
	ids = append(ids, schedule.LastRun)
	completed := []runs.Run{}
	for _, id := range ids[1:] {
		deadline := time.Now().Add(65 * time.Second)
		if kind == "go" {
			deadline = time.Now().Add(150 * time.Second)
		}
		var current runs.Run
		for time.Now().Before(deadline) {
			current, _, err = runner.Get(t.Context(), "alpha", id)
			if err != nil {
				t.Fatal(err)
			}
			if runs.Terminal(current.State) {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		if current.State != "completed" || current.Outcome != "passed" || current.Counts.Total != 1 || current.Counts.Passed != 1 || current.EvidenceState != expectedAPIEvidence(kind) {
			t.Fatalf("API run differs: state=%s outcome=%s results=%+v error=%s gaps=%v", current.State, current.Outcome, current.Results, current.Error, current.EvidenceErrors)
		}
		if len(current.Results) != 1 || len(current.Results[0].Steps) < 5 || current.Tests[0].Type != "api" {
			t.Fatal("API steps inflate or lose logical test identity")
		}
		if current.Results[0].DiagnosticsPolicy != diagnostics.Version || (kind == "playwright" && len(current.EvidenceErrors) == 0) {
			t.Fatal("diagnostic policy and omission are missing")
		}
		page, err := s.Store.List(t.Context(), evidence.Bucket(id), "alpha", "", 100)
		if err != nil {
			t.Fatal(err)
		}
		inspected := 0
		for _, row := range page.Items {
			artifact, err := storage.Decode[evidence.Artifact](row)
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Test != "API-001" {
				continue
			}
			coded := strings.HasPrefix(artifact.Stream, kind+".") || artifact.Kind == "attachment"
			if coded && artifact.RedactionPolicy != diagnostics.Version {
				t.Fatal("retained artifact lacks redaction policy")
			}
			_, reader, err := runner.Evidence.Open(t.Context(), "alpha", id, artifact.ID)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil || strings.Contains(string(data), "canary") || strings.Contains(string(data), boundSecret) {
				t.Fatal("API private diagnostic value reached evidence", artifact.Kind, err)
			}
			if coded {
				inspected++
			}
		}
		encoded, _ := json.Marshal(current.Results)
		if inspected < 2 || strings.Contains(string(encoded), "canary") {
			t.Fatal("API diagnostics or redaction were lost")
		}
		completed = append(completed, current)
	}
	if hits.Load() != 6 {
		t.Fatal("denied requests reach the upstream fixture", hits.Load())
	}
	history, err := runner.TestHistory(t.Context(), "admin", "alpha", "health", "API-001", "", 10)
	if err != nil || len(history.Items) != 2 {
		t.Fatal("API history differs from normal results", err)
	}
	trend, err := (&trends.Service{Store: s.Store, Auth: s.Auth}).Read(t.Context(), "admin", "alpha", trends.Filter{Suite: "health", Test: "API-001", Since: time.Now().Add(-time.Hour), Until: time.Now().Add(time.Minute)}, "", 20)
	if err != nil || trend.Totals.Total != 2 || trend.Totals.Passed != 2 || len(trend.Points) != 2 {
		t.Fatal("API trends inflate request counts", trend.Totals, err)
	}
	data, _ := json.MarshalIndent(map[string]any{"preparation": job.ID, "release": release.ID, "runs": completed, "trends": trend, "history": history}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "summary.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func expectedAPIEvidence(kind string) string {
	if kind == "go" {
		return "complete"
	}
	return "incomplete"
}
