package sources_test

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
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
	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestRegisteredGitHubSourceCompletesUIReviewPublicationAndExecution(t *testing.T) {
	pwRuntime, goRuntime, gateway, ui := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE"), os.Getenv("TESTSTUDIO_SOURCE_RUNTIME_IMAGE"), os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_UI_DIRECTORY")
	if pwRuntime == "" || goRuntime == "" || gateway == "" || ui == "" {
		t.Skip("supply cached Go, Playwright, gateway images and the built UI")
	}
	s, c, _ := bundleSetup(t)
	root := os.Getenv("TESTSTUDIO_REPOSITORY_UI_OUTPUT")
	if root == "" {
		root = t.TempDir()
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	fixtureArchive := repositoryArchive(t, false)
	var downloads atomic.Int32
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/owner/suite":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "full_name": "owner/suite", "private": false})
		case strings.HasPrefix(r.URL.Path, "/repos/owner/suite/commits/"):
			_, _ = w.Write([]byte(strings.Repeat("a", 40)))
		case r.URL.Path == "/repos/owner/suite/zipball/"+strings.Repeat("a", 40):
			downloads.Add(1)
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(fixtureArchive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(provider.Close)
	roots := x509.NewCertPool()
	roots.AddCert(provider.Certificate())
	s.Providers = map[string]sourceprovider.Provider{"github": &githubsource.Client{APIBase: provider.URL, DownloadOrigin: provider.URL, RootCAs: roots, Network: networkguard.Loopback}}
	s.BundleRunners = []string{"go", "playwright"}
	s.Builders = map[string]preparation.ArtifactBuilder{"go": &localimage.Builder{RuntimeImage: goRuntime, TempDirectory: filepath.Join(root, "go-contexts")}, "playwright": &localimage.Builder{RuntimeImage: pwRuntime, TempDirectory: filepath.Join(root, "browser-contexts")}}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"go": &localcontainer.Executor{Runner: "go", AllowProjectImages: true, RequireProject: true}, "playwright": &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true}}
	s.RecoveryInspectors = s.Inspectors
	projectService := &projects.Service{Store: s.Store, Auth: s.Auth}
	if err := projectService.SaveSuite(t.Context(), "admin", "beta", projects.Suite{ID: "health", Name: "Repository checks", Applications: []string{"app"}, RequiresTarget: false}, 1); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	sessionService := &sessions.Service{Store: s.Store, Directory: s.Auth.Directory, Origin: origin}
	response := httptest.NewRecorder()
	session, err := sessionService.Create(t.Context(), response, "admin", "repository-ui-fixture", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sessionService.Revoke(context.Background(), session.ID); err != nil {
			t.Error(err)
		}
	})
	cookie := response.Result().Cookies()[0]
	cookieJSON, _ := json.Marshal(map[string]any{"name": cookie.Name, "value": cookie.Value, "url": origin, "httpOnly": true, "sameSite": "Lax"})
	specification, err := os.ReadFile("../../tests/authoring/repository-release.spec.ts")
	if err != nil {
		t.Fatal(err)
	}
	draft := sources.StarterDraft()
	draft.ID = "repository-ui"
	draft.Name = "Repository publication UI"
	draft.TargetURL = origin
	draft.Files[0] = sources.EditorFile{Path: "checks/repository-release.spec.ts", Content: strings.ReplaceAll(string(specification), "FIXTURE_SESSION_COOKIE", string(cookieJSON))}
	var document catalog.Document
	_ = json.Unmarshal([]byte(draft.Files[1].Content), &document)
	document.Tests[0].ID = "STUDIO-REPOSITORY-RELEASE-001"
	document.Tests[0].Title = "Repository UI release publication"
	document.Tests[0].Selector = "@STUDIO-REPOSITORY-RELEASE-001"
	document.Tests[0].Source = draft.Files[0].Path
	document.Tests[0].TimeoutSeconds = 120
	document.Tests[0].Capture.Trace = "always"
	metadata, _ := json.Marshal(document)
	draft.Files[1].Content = string(metadata)
	if err := projectService.SaveApplication(t.Context(), "admin", "alpha", projects.Application{ID: "studio", Name: "Local TestStudio"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveEnvironment(t.Context(), "admin", "alpha", projects.Environment{ID: "studio", Application: "studio", Name: "Local TestStudio", Stage: "local", Endpoints: map[string]string{"api": origin}, Locations: []string{"local"}}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveSuite(t.Context(), "admin", "alpha", projects.Suite{ID: "ui", Name: "Source authoring UI", Applications: []string{"studio"}, RequiresTarget: true}, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cleanupRepositoryUIJobs(t, s, root) })
	saved, err := s.SaveDraft(t.Context(), "admin", "alpha", "ui", draft, 0)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareDraft(t.Context(), "admin", "alpha", "ui", draft.ID, "repository-ui", saved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := s.ProcessBuildNext(t.Context(), "repository-ui-builder"); err != nil || !processed {
		t.Fatal(processed, err)
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "ui", prepared.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	if job.State != "ready" {
		t.Fatalf("UI preparation: state=%s error=%s log=%s", job.State, job.Error, job.Log)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "ui", job.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = blobs.Close() })
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Location: "local", Concurrency: 2, PollInterval: 20 * time.Millisecond, Go: &localcontainer.Executor{Runner: "go", ApproveArtifact: s.ApproveArtifact}, Browser: &localcontainer.Executor{Runner: "playwright", Environment: true, GatewayImage: gateway, ApproveArtifact: s.ApproveArtifact}}
	api := &httpapi.Server{Store: s.Store, Auth: s.Auth, Projects: projectService, Catalog: c, Sources: s, Runs: runner, Evidence: runner.Evidence, Sessions: sessionService, UI: ui}
	server := httptest.NewUnstartedServer(api.Handler())
	_ = server.Listener.Close()
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(t.Context())
	stops := []<-chan struct{}{runner.Start(ctx), s.StartRepositorySynchronization(ctx), s.StartBuildPreparation(ctx)}
	t.Cleanup(func() {
		cancel()
		for _, done := range stops {
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				t.Error("owned source or execution worker did not stop")
			}
		}
	})
	run, err := runner.Create(t.Context(), "admin", "alpha", runs.Request{Suite: "ui", Environment: "studio", TimeoutSeconds: 150}, "repository-ui-run", "manual")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(160 * time.Second)
	for time.Now().Before(deadline) {
		run, _, err = runner.Get(t.Context(), "alpha", run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(run.State) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	data, _ := json.MarshalIndent(map[string]any{"run": run, "downloads": downloads.Load()}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "summary.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if run.State != "completed" || run.Outcome != "passed" || run.Counts.Passed != 1 || run.EvidenceState != "complete" {
		t.Fatalf("UI execution: state=%s outcome=%s results=%+v evidence=%v", run.State, run.Outcome, run.Results, run.EvidenceErrors)
	}
	if downloads.Load() != 1 {
		t.Fatal("repository archive did not use one pinned source snapshot", downloads.Load())
	}
	betaRuns, err := s.Store.All(t.Context(), "runs", "beta")
	if err != nil || len(betaRuns) != 1 {
		t.Fatal("publication creates unintended runs", err, len(betaRuns))
	}
	betaRun, _ := storage.Decode[runs.Run](betaRuns[0])
	if betaRun.Outcome != "passed" || betaRun.Counts.Total != 1 || betaRun.Counts.Passed != 1 {
		t.Fatal("published Go release did not run", betaRun.Results)
	}
	data, _ = json.MarshalIndent(map[string]any{"uiRun": run, "repositoryRun": betaRun, "downloads": downloads.Load()}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "summary.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("UI run=%s repository run=%s archive downloads=%d", run.ID, betaRun.ID, downloads.Load())
}

func cleanupRepositoryUIJobs(t *testing.T, s *sources.Service, root string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	ids := []string{}
	images := []string{}
	for _, scope := range [][2]string{{"alpha", "ui"}, {"beta", "health"}} {
		rows, err := s.Store.All(ctx, sources.JobBucket(scope[0], scope[1]), scope[0])
		if err != nil {
			t.Error(err)
			continue
		}
		for _, row := range rows {
			job, err := storage.Decode[sources.Job](row)
			if err != nil {
				t.Error(err)
				continue
			}
			ids = append(ids, job.ID)
			if job.Reference != nil {
				if err := s.Inspectors[job.Runner].CleanupInspection(ctx, preparation.InspectionRequest{Image: job.Image, Suite: job.Suite, Reference: *job.Reference}); err != nil {
					t.Error(err)
				}
				if err := s.Builders[job.Runner].CleanupBuild(ctx, *job.Reference); err != nil {
					t.Error(err)
				}
			}
			if job.Image != "" {
				images = append(images, job.Image)
				output, err := exec.CommandContext(ctx, "docker", "image", "rm", job.Image).CombinedOutput()
				if err != nil && !strings.Contains(string(output), "No such image") {
					t.Errorf("owned source image cleanup: %s %v", output, err)
				}
			}
		}
	}
	for _, project := range []string{"alpha", "beta"} {
		rows, err := s.Store.All(ctx, "runs", project)
		if err != nil {
			t.Error(err)
			continue
		}
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
	}
	for _, id := range ids {
		for _, kind := range []string{"container", "volume"} {
			args := []string{kind, "ls", "-q", "--filter", "label=teststudio.run=" + id}
			if kind == "container" {
				args = append(args, "-a")
			}
			output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
			if err != nil || len(strings.TrimSpace(string(output))) > 0 {
				t.Errorf("owned resource remains: %s %v", output, err)
			}
		}
	}
	data, _ := json.MarshalIndent(map[string]any{"resourceIds": ids, "images": images}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "resources.json"), data, 0600); err != nil {
		t.Error(err)
	}
}
