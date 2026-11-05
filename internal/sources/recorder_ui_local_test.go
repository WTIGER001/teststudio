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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/adapters/recording/companion"
	"github.com/wtiger001/teststudio/adapters/recording/localplaywright"
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

type observedRecorder struct {
	*localplaywright.Driver
	mu    sync.Mutex
	known map[int]bool
}

func (d *observedRecorder) observe() map[int]bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	command := exec.Command("ps", "-axo", "pid=,ppid=")
	output, err := command.Output()
	if err != nil {
		return map[int]bool{-1: true}
	}
	parents := map[int]int{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		if pid == command.Process.Pid {
			continue
		}
		parent, _ := strconv.Atoi(fields[1])
		parents[pid] = parent
	}
	current := map[int]bool{os.Getpid(): true}
	for changed := true; changed; {
		changed = false
		for pid, parent := range parents {
			if current[parent] && !current[pid] {
				current[pid] = true
				d.known[pid] = true
				changed = true
			}
		}
	}
	alive := map[int]bool{}
	for pid := range d.known {
		if _, ok := parents[pid]; ok {
			alive[pid] = true
		}
	}
	return alive
}

func (d *observedRecorder) Read(ctx context.Context, id string) (recording.Snapshot, error) {
	d.observe()
	return d.Driver.Read(ctx, id)
}

func TestHostedRecorderCompanionCompletesGuidedUIAndNativeCapture(t *testing.T) {
	runtime, gateway, ui := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE"), os.Getenv("TESTSTUDIO_BROWSER_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_UI_DIRECTORY")
	if os.Getenv("TESTSTUDIO_NATIVE_RECORDING_UI") != "1" || runtime == "" || gateway == "" || ui == "" {
		t.Skip("enable the focused native companion UI check with cached runtime, gateway, and built UI")
	}
	workspace, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("TESTSTUDIO_RECORDER_UI_OUTPUT")
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
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<!doctype html><title>Recorder companion verification</title><main><h1>Independent local recording target</h1><button>Continue</button></main>")
	}))
	t.Cleanup(target.Close)
	private := t.TempDir()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	native, err := localplaywright.New(t.Context(), node, filepath.Join(workspace, "runners/playwright/node_modules/playwright/cli.js"), filepath.Join(workspace, "scripts/record-browser.mjs"), filepath.Join(private, "captures"))
	if err != nil {
		t.Fatal(err)
	}
	driver := &observedRecorder{Driver: native, known: map[int]bool{}}
	t.Cleanup(func() {
		if err := driver.Close(); err != nil {
			t.Error(err)
		}
	})
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	desktopOrigin := fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port)
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
	specification, err := os.ReadFile(filepath.Join(workspace, "tests/authoring/recorder-companion.spec.ts"))
	if err != nil {
		t.Fatal(err)
	}
	targetJSON, _ := json.Marshal(target.URL)
	draft := sources.StarterDraft()
	draft.ID, draft.Name, draft.TargetURL = "recorder-ui", "Hosted recorder authoring UI", origin
	draft.Files[0] = sources.EditorFile{Path: "checks/recorder-companion.spec.ts", Content: strings.ReplaceAll(strings.ReplaceAll(string(specification), "FIXTURE_SESSION_COOKIE", string(cookieJSON)), "FIXTURE_TARGET_URL", string(targetJSON))}
	var document catalog.Document
	if err := json.Unmarshal([]byte(draft.Files[1].Content), &document); err != nil {
		t.Fatal(err)
	}
	document.Tests[0].ID, document.Tests[0].Title, document.Tests[0].Selector = "STUDIO-COMPANION-001", "Hosted recorder authoring UI", "@STUDIO-COMPANION-001"
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
	recordings := &recording.Service{Store: s.Store, Auth: s.Auth, CompanionEnabled: true}
	api := &httpapi.Server{Store: s.Store, Auth: s.Auth, Projects: projectService, Catalog: c, Sources: s, Runs: runService, Evidence: runService.Evidence, Sessions: sessionService, Recordings: recordings, UI: ui}
	mux := http.NewServeMux()
	mux.Handle("/", api.Handler())
	var connected atomic.Bool
	desktopCtx, cancelDesktop := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancelDesktop()
	desktopDone := make(chan error, 1)
	mux.HandleFunc("POST /verification/companion", func(w http.ResponseWriter, r *http.Request) {
		current, err := sessionService.Read(r.Context(), r)
		if err != nil || current.ID != session.ID || sessionService.CheckMutation(r, current) != nil {
			http.Error(w, "forbidden", 403)
			return
		}
		var input struct {
			Code string `json:"code"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 1025)).Decode(&input) != nil || len(input.Code) > 1024 {
			http.Error(w, "invalid connection", 422)
			return
		}
		state, err := companion.NewState(desktopOrigin, input.Code)
		if err != nil {
			http.Error(w, "invalid connection", 422)
			return
		}
		if _, err := recordings.Read(r.Context(), "admin", "beta", "health", state.ID); err != nil || connected.Load() {
			http.Error(w, "unavailable recording", 409)
			return
		}
		journal, err := companion.NewJournal(filepath.Join(private, "connections"))
		if err != nil || journal.Create(state) != nil {
			http.Error(w, "connection unavailable", 500)
			return
		}
		remote, err := companion.NewClient(desktopOrigin)
		if err != nil {
			http.Error(w, "connection unavailable", 500)
			return
		}
		if !connected.CompareAndSwap(false, true) {
			http.Error(w, "unavailable recording", 409)
			return
		}
		go func() {
			desktopDone <- (&companion.Runner{Remote: remote, Driver: driver, Journal: journal}).Run(desktopCtx, &state)
		}()
		httpapi.JSON(w, 202, map[string]bool{"started": true})
	})
	server := httptest.NewUnstartedServer(mux)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		cancelDesktop()
		if err := driver.Close(); err != nil {
			t.Error(err)
		}
		if connected.Load() {
			select {
			case err := <-desktopDone:
				if err != nil {
					t.Error("desktop companion:", err)
				}
			case <-time.After(20 * time.Second):
				t.Error("desktop companion did not stop")
			}
		}
		deadline := time.Now().Add(5 * time.Second)
		alive := driver.observe()
		for len(alive) > 0 && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
			alive = driver.observe()
		}
		data, _ := json.MarshalIndent(map[string]any{"ownedProcesses": len(driver.known), "remainingProcesses": len(alive)}, "", "  ")
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
	if err != nil || len(rows.Items) != 1 {
		t.Fatal("recording history did not retain exactly one session", err)
	}
	captured, _ := storage.Decode[recording.Session](rows.Items[0])
	if captured.State != "ready" || !strings.Contains(captured.Code, target.URL) {
		t.Fatal("native browser recording did not retain the real initial navigation")
	}
	drafts, err := s.Drafts(t.Context(), "admin", "beta", "health", "", 30)
	if err != nil || len(drafts.Items) != 1 {
		t.Fatal("reviewed recording did not save exactly one source draft", err)
	}
	data, _ = json.MarshalIndent(map[string]any{"run": run, "recordingId": captured.ID, "recordingState": captured.State, "savedDrafts": len(drafts.Items)}, "", "  ")
	os.WriteFile(filepath.Join(root, "summary.json"), data, 0600)
	t.Logf("UI run=%s native recording=%s; reviewed source saved without a target test run", run.ID, captured.ID)
}
