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
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/adapters/recording/embeddedplaywright"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

type observedEmbeddedRecorder struct {
	*embeddedplaywright.Driver
	observer *observedRecorder
}

func (d *observedEmbeddedRecorder) Start(ctx context.Context, id, address string) error {
	err := d.Driver.Start(ctx, id, address)
	d.observer.observe()
	return err
}

func (d *observedEmbeddedRecorder) Stop(ctx context.Context, id string) (recording.Snapshot, error) {
	d.observer.observe()
	return d.Driver.Stop(ctx, id)
}

func (d *observedEmbeddedRecorder) Read(ctx context.Context, id string) (recording.Snapshot, error) {
	d.observer.observe()
	return d.Driver.Read(ctx, id)
}

func TestEmbeddedRecorderCompletesNewsRecordingAndDraftThroughStudioExecution(t *testing.T) {
	if os.Getenv("TESTSTUDIO_EMBEDDED_RECORDING_UI") != "1" {
		t.Skip("enable the focused native recording UI check")
	}
	runRecordingAuthoringUI(t, false)
}

func runRecordingAuthoringUI(t *testing.T, hosted bool) {
	runtime, gateway, ui := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE"), os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_UI_DIRECTORY")
	if runtime == "" || gateway == "" || ui == "" {
		t.Skip("enable the focused embedded recorder UI check with cached runtime, gateway, and built UI")
	}
	workspace, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("TESTSTUDIO_EMBEDDED_UI_OUTPUT")
	if hosted {
		root = os.Getenv("TESTSTUDIO_HOSTED_UI_OUTPUT")
	}
	if root == "" {
		root = t.TempDir()
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	s, c, _ := bundleSetup(t)
	s.BundleRunners = []string{"playwright"}
	s.Builders = map[string]preparation.ArtifactBuilder{"playwright": &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "browser-contexts")}}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"playwright": &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true}}
	s.RecoveryInspectors = s.Inspectors
	t.Cleanup(func() { cleanupRepositoryUIJobs(t, s, root) })
	dependencyOrigin := ""
	if hosted {
		dependency := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/javascript")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			io.WriteString(w, `document.querySelector('#application').innerHTML='<nav><a href="/news">News</a></nav>'+(location.pathname==='/news'?'<main><article><h2>Regional harvest begins</h2><p>A new independent news fixture.</p></article></main>':'');`)
		}))
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Fatal(err)
		}
		dependency.Listener.Close()
		dependency.Listener = listener
		dependency.Start()
		t.Cleanup(dependency.Close)
		dependencyOrigin = fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	}
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if hosted {
			fmt.Fprintf(w, `<!doctype html><style>body{margin:0}a{display:block;padding:15px}h2{margin:20px}</style><div id="application">Loading the independent News application...</div><script type="module" src="%s/bootstrap.js"></script>`, dependencyOrigin)
			return
		}
		io.WriteString(w, `<!doctype html><style>body{margin:0}a{display:block;padding:15px}h2{margin:20px}</style><nav><a href="/news">News</a></nav>`)
		if r.URL.Path == "/news" {
			io.WriteString(w, `<main><article><h2>Regional harvest begins</h2><p>A new independent news fixture.</p></article></main>`)
		}
	}))
	targetAddress := "127.0.0.1:0"
	if hosted {
		targetAddress = "0.0.0.0:0"
	}
	targetListener, err := net.Listen("tcp", targetAddress)
	if err != nil {
		t.Fatal(err)
	}
	target.Listener.Close()
	target.Listener = targetListener
	target.Start()
	targetURL := target.URL
	if hosted {
		targetURL = fmt.Sprintf("http://host.docker.internal:%d", targetListener.Addr().(*net.TCPAddr).Port)
	}
	t.Cleanup(target.Close)
	private := t.TempDir()
	observer := &observedRecorder{known: map[int]bool{}}
	var driver recording.InteractiveDriver
	var closeDriver func() error
	if hosted {
		client := newHostedRecordingClient(t, s, private, root, observer, targetURL, dependencyOrigin)
		driver, closeDriver = client, client.Close
	} else {
		node, err := exec.LookPath("node")
		if err != nil {
			t.Fatal(err)
		}
		native, err := embeddedplaywright.New(t.Context(), node, filepath.Join(workspace, "runners/playwright/node_modules/playwright/cli.js"), filepath.Join(workspace, "scripts/record-browser-embedded.mjs"), filepath.Join(private, "captures"))
		if err != nil {
			t.Fatal(err)
		}
		driver, closeDriver = &observedEmbeddedRecorder{Driver: native, observer: observer}, native.Close
	}
	t.Cleanup(func() {
		if err := closeDriver(); err != nil {
			t.Error(err)
		}
	})
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	sessionService := &sessions.Service{Store: s.Store, Directory: s.Auth.Directory, Origin: origin}
	response := httptest.NewRecorder()
	session, err := sessionService.Create(t.Context(), response, "admin", "recorder-ui-fixture", time.Now().Add(5*time.Minute))
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
	projectService := &projects.Service{Store: s.Store, Auth: s.Auth}
	if err := projectService.SaveApplication(t.Context(), "admin", "alpha", projects.Application{ID: "studio", Name: "Local TestStudio"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveEnvironment(t.Context(), "admin", "alpha", projects.Environment{ID: "studio", Application: "studio", Name: "Local TestStudio", Stage: "local", Endpoints: map[string]string{"api": origin}, Locations: []string{"local"}}, 0); err != nil {
		t.Fatal(err)
	}
	if err := projectService.SaveSuite(t.Context(), "admin", "alpha", projects.Suite{ID: "ui", Name: "Recording authoring UI", Applications: []string{"studio"}, RequiresTarget: true}, 0); err != nil {
		t.Fatal(err)
	}
	specification, err := os.ReadFile(filepath.Join(workspace, "tests/authoring/embedded-recorder.spec.ts"))
	if err != nil {
		t.Fatal(err)
	}
	targetJSON, _ := json.Marshal(targetURL)
	draft := sources.StarterDraft()
	draft.ID, draft.Name, draft.TargetURL = "recorder-ui", "Embedded recorder authoring UI", origin
	draft.Files[0] = sources.EditorFile{Path: "checks/embedded-recorder.spec.ts", Content: strings.ReplaceAll(strings.ReplaceAll(string(specification), "FIXTURE_SESSION_COOKIE", string(cookieJSON)), "FIXTURE_TARGET_URL", string(targetJSON))}
	var document catalog.Document
	if err := json.Unmarshal([]byte(draft.Files[1].Content), &document); err != nil {
		t.Fatal(err)
	}
	testID := "STUDIO-EMBEDDED-RECORDER-001"
	if hosted {
		testID = "STUDIO-HOSTED-RECORDER-001"
		draft.Files[0].Content = strings.ReplaceAll(draft.Files[0].Content, "@STUDIO-EMBEDDED-RECORDER-001", "@"+testID)
	}
	document.Tests[0].ID, document.Tests[0].Title, document.Tests[0].Selector = testID, "Embedded recorder authoring UI", "@"+testID
	document.Tests[0].Source, document.Tests[0].TimeoutSeconds = draft.Files[0].Path, 90
	document.Tests[0].Capture.Trace = "off"
	metadata, _ := json.Marshal(document)
	draft.Files[1].Content = string(metadata)
	saved, err := s.SaveDraft(t.Context(), "admin", "alpha", "ui", draft, 0)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareDraft(t.Context(), "admin", "alpha", "ui", draft.ID, "recorder-ui", saved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := s.ProcessBuildNext(t.Context(), "recorder-ui-builder"); err != nil || !processed {
		t.Fatal(processed, err)
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "ui", prepared.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	if job.State != "ready" {
		t.Fatalf("UI preparation state=%s error=%s log=%s", job.State, job.Error, job.Log)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "ui", job.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	runService := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Location: "local", PollInterval: 20 * time.Millisecond, Browser: &localcontainer.Executor{Runner: "playwright", Environment: true, GatewayImage: gateway, ApproveArtifact: s.ApproveArtifact}}
	recordings := &recording.Service{Store: s.Store, Auth: s.Auth, Driver: driver}
	api := &httpapi.Server{Store: s.Store, Auth: s.Auth, Projects: projectService, Catalog: c, Sources: s, Runs: runService, Evidence: runService.Evidence, Sessions: sessionService, Recordings: recordings, UI: ui}
	server := httptest.NewUnstartedServer(api.Handler())
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		if err := closeDriver(); err != nil {
			t.Error(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		alive := observer.observe()
		for len(alive) > 0 && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
			alive = observer.observe()
		}
		data, _ := json.MarshalIndent(map[string]any{"ownedProcesses": len(observer.known), "remainingProcesses": len(alive)}, "", "  ")
		os.WriteFile(filepath.Join(root, "cleanup.json"), data, 0600)
		if len(alive) != 0 {
			t.Error("owned verification processes remain after cleanup", len(alive))
		}
	})
	workerCtx, cancelWorkers := context.WithCancel(t.Context())
	done := runService.Start(workerCtx)
	t.Cleanup(func() { cancelWorkers(); <-done })
	run, err := runService.Create(t.Context(), "admin", "alpha", runs.Request{Suite: "ui", Environment: "studio", TimeoutSeconds: 110}, "recorder-ui-run", "manual")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		run, _, err = runService.Get(t.Context(), "alpha", run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(run.State) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	data, _ := json.MarshalIndent(map[string]any{"run": run}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "summary.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if run.State != "completed" || run.Outcome != "passed" || run.Counts.Passed != 1 || run.EvidenceState != "complete" {
		t.Fatalf("UI execution state=%s outcome=%s results=%+v evidence=%v", run.State, run.Outcome, run.Results, run.EvidenceErrors)
	}
	rows, err := recordings.List(t.Context(), "admin", "beta", "health", "", 30)
	if err != nil || len(rows.Items) != 2 {
		t.Fatal("recording history did not retain the finished and cancelled sessions", err)
	}
	var captured recording.Session
	cancelled := 0
	for _, row := range rows.Items {
		value, _ := storage.Decode[recording.Session](row)
		if value.State == "ready" {
			captured = value
		}
		if value.State == "cancelled" {
			cancelled++
		}
	}
	if cancelled != 1 {
		t.Fatal("the cancelled browser was not retained as cancelled")
	}
	if captured.State != "ready" || (!strings.Contains(captured.Code, "toBeVisible()") || len(captured.Steps) < 3) {
		t.Fatal("embedded recording did not retain News navigation and a story assertion")
	}
	drafts, err := s.Drafts(t.Context(), "admin", "beta", "health", "", 30)
	if err != nil || len(drafts.Items) != 1 {
		t.Fatal("reviewed recording did not save exactly one source draft", err)
	}
	data, _ = json.MarshalIndent(map[string]any{"run": run, "recordingId": captured.ID, "recordingState": captured.State, "savedDrafts": len(drafts.Items)}, "", "  ")
	os.WriteFile(filepath.Join(root, "summary.json"), data, 0600)
	t.Logf("UI run=%s embedded recording=%s; reviewed source saved without a target test run", run.ID, captured.ID)
}
