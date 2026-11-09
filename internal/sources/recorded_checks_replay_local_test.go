package sources_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func replayRecordedChecks(t *testing.T, service *sources.Service, runner *runs.Service, recorded recording.Session, root string) {
	t.Helper()
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<!doctype html><title>Independent News</title><style>body{margin:0}a{display:block;padding:15px}h2{margin:20px}</style><div id="application"><nav><a href="/news">News</a></nav>`)
		if r.URL.Path == "/news" {
			io.WriteString(w, `<main><div class="story-card"><h2>Regional harvest begins</h2><p>A new independent news fixture.</p></div><div class="story-card"><h2>A different story in the selected environment</h2><p>Another independent news fixture.</p></div><div class="story-card" hidden><h2>Hidden draft story</h2><p>This draft must not count as a displayed story.</p></div></main><aside style="position:fixed;right:10px;top:300px;width:240px;display:grid;gap:8px"><input aria-label="Search stories" value="Harvest" style="height:32px;box-sizing:border-box"><button disabled style="height:32px">Unavailable action</button><label style="height:32px"><input aria-label="Include archived" type="checkbox" checked> Include archived</label></aside>`)
		}
		io.WriteString(w, `</div>`)
	}))
	target.Listener.Close()
	target.Listener = listener
	target.Start()
	t.Cleanup(target.Close)
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	registry := &projects.Service{Store: service.Store, Auth: service.Auth}
	if err := registry.SaveApplication(t.Context(), "admin", "beta", projects.Application{ID: "replay", Name: "Recorded checks replay"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := registry.SaveEnvironment(t.Context(), "admin", "beta", projects.Environment{ID: "replay", Application: "replay", Name: "Different local News target", Stage: "local", Endpoints: map[string]string{"api": origin}, Locations: []string{"local"}}, 0); err != nil {
		t.Fatal(err)
	}
	suiteRow, err := service.Store.Get(t.Context(), projects.Bucket("suites", "beta"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		t.Fatal(err)
	}
	suite.RequiresTarget = true
	suite.Applications = []string{"replay"}
	if err := registry.SaveSuite(t.Context(), "admin", "beta", suite, suiteRow.Revision); err != nil {
		t.Fatal(err)
	}
	jobs, err := service.ListJobs(t.Context(), "admin", "beta", "health", "", 30)
	if err != nil || len(jobs.Items) != 1 {
		t.Fatal("saved recording preparation was not queued once", err)
	}
	queued, err := storage.Decode[sources.Job](jobs.Items[0])
	if err != nil || queued.Draft != recorded.ID {
		t.Fatal("preparation selected another recording", err)
	}
	if done, err := service.ProcessBuildNext(t.Context(), "recorded-checks-builder"); err != nil || !done {
		t.Fatal("saved recording did not prepare", done, err)
	}
	row, err := service.ReadJob(t.Context(), "admin", "beta", "health", queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err := storage.Decode[sources.Job](row)
	if err != nil || job.State != "ready" {
		t.Fatalf("saved recording preparation state=%s error=%s: %v", job.State, job.Error, err)
	}
	release, err := service.Publish(t.Context(), "admin", "beta", "health", job.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.Draft.Tests) != 1 || release.Draft.Tests[0].ID != recorded.TestID {
		t.Fatal("publication changed the recorded test identity")
	}
	policy := capture.DefaultPolicy()
	policy.Screenshots = "always"
	policy.Trace = "always"
	run, err := runner.Create(t.Context(), "admin", "beta", runs.Request{Suite: "health", Environment: "replay", IDs: []string{recorded.TestID}, TimeoutSeconds: 60, Capture: &policy}, "recorded-checks-explicit-run", "manual")
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(70 * time.Second); !runs.Terminal(run.State) && time.Now().Before(deadline); {
		time.Sleep(30 * time.Millisecond)
		run, _, err = runner.Get(t.Context(), "beta", run.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	data, _ := json.MarshalIndent(run, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "recorded-checks-replay.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if run.State != "completed" || run.Outcome != "passed" || run.Counts.Total != 1 || run.Counts.Passed != 1 || run.EvidenceState != "complete" {
		t.Fatalf("recorded checks replay state=%s outcome=%s results=%+v evidence=%v", run.State, run.Outcome, run.Results, run.EvidenceErrors)
	}
	t.Logf("recorded checks replay=%s duration=%.3fms; original source ran against a different reviewed environment", run.ID, run.Results[0].DurationMs)
}
