package isolatedcontainer

import (
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/recordingworker"
)

func TestIsolatedRecordingUsesSandboxGatewayAndOwnedCleanup(t *testing.T) {
	image, gateway, seccomp := os.Getenv("TESTSTUDIO_RECORDING_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_SECCOMP")
	if image == "" || gateway == "" || seccomp == "" {
		t.Skip("supply the cached recording and gateway images and pinned sandbox policy")
	}
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	foreignListener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	var foreignRequests atomic.Int32
	foreign := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { foreignRequests.Add(1); w.WriteHeader(200) }))
	foreign.Listener.Close()
	foreign.Listener = foreignListener
	foreign.Start()
	t.Cleanup(foreign.Close)
	foreignURL := fmt.Sprintf("http://host.docker.internal:%d/forbidden", foreignListener.Addr().(*net.TCPAddr).Port)
	policyComplete := make(chan struct{}, 1)
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/policy-complete" {
			select {
			case policyComplete <- struct{}{}:
			default:
			}
			w.WriteHeader(200)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><style>body{margin:0}a{display:block;padding:15px}h2{margin:20px}</style><nav><a href="/news">News</a></nav>`)
		if r.URL.Path == "/news" {
			fmt.Fprint(w, `<main><article><h2>Regional harvest begins</h2></article></main>`)
			address, _ := json.Marshal(foreignURL)
			fmt.Fprintf(w, `<script>Promise.allSettled([fetch(%s,{signal:AbortSignal.timeout(3000)}),fetch('http://169.254.169.254/latest/meta-data/',{signal:AbortSignal.timeout(3000)})]).then(()=>fetch('/policy-complete'))</script>`, address)
		}
	}))
	target.Listener.Close()
	target.Listener = listener
	target.Start()
	t.Cleanup(target.Close)
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	b := recordingworker.Binding{ID: domain.NewID(), Actor: "tester", Project: "fixture", Suite: "browser", URL: origin, Deadline: time.Now().Add(2 * time.Minute).UTC()}
	runtime := &Runtime{Image: image, GatewayImage: gateway, Seccomp: seccomp, Directory: t.TempDir()}
	ref, err := runtime.Plan(b, browserpolicy.Target{Origin: origin, AllowPrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := runtime.Cleanup(ctx, b, ref); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	driver, err := runtime.Start(ctx, b, ref, browserpolicy.Target{Origin: origin, AllowPrivate: true})
	if err != nil {
		data, _ := os.ReadFile(filepath.Join(runtime.Directory, ref.Fingerprint, b.ID, "recorder.log"))
		t.Log(string(data))
		t.Fatal(err)
	}
	metadata, err := exec.CommandContext(ctx, runtime.binary(), "inspect", ref.Container, "--format", "{{json .HostConfig}}").Output()
	var host struct {
		NetworkMode    string
		ReadonlyRootfs bool
		CapDrop        []string
		Memory         int64
		SecurityOpt    []string
	}
	if err != nil || json.Unmarshal(metadata, &host) != nil || host.NetworkMode != "none" || !host.ReadonlyRootfs || len(host.CapDrop) != 1 || host.CapDrop[0] != "ALL" || host.Memory != 2<<30 {
		t.Fatal("recording browser isolation did not match its fixed profile")
	}
	bypass := exec.CommandContext(ctx, runtime.binary(), "exec", ref.Container, "node", "-e", `fetch(process.argv[1],{signal:AbortSignal.timeout(1000)}).then(()=>process.exit(1)).catch(()=>process.exit(0))`, origin+"/bypass")
	if bypass.Run() != nil {
		t.Fatal("recording browser could bypass its gateway")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if _, err := driver.Stop(ctx, b.ID); err != nil {
			t.Error(err)
		}
	})
	view, err := driver.Interact(ctx, b.ID, recording.Action{Action: "click", Sequence: 1, X: 35, Y: 25})
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); !strings.HasSuffix(view.URL, "/news") && time.Now().Before(deadline); {
		view, err = driver.View(ctx, b.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !strings.HasSuffix(view.URL, "/news") {
		t.Fatal("isolated browser did not navigate through its gateway")
	}
	if _, err := driver.Interact(ctx, b.ID, recording.Action{Action: "assert-visible", Sequence: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := driver.Interact(ctx, b.ID, recording.Action{Action: "click", Sequence: 3, X: 120, Y: 85}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-policyComplete:
	case <-ctx.Done():
		t.Fatal("browser policy probes did not complete")
	}
	if foreignRequests.Load() != 0 {
		t.Fatal("recording browser reached an unreviewed origin")
	}
	snapshot, err := driver.Stop(ctx, b.ID)
	if err != nil || !strings.Contains(snapshot.Code, "toBeVisible()") {
		t.Fatal("isolated recorder did not capture the story assertion", err)
	}
	if err := runtime.Cleanup(ctx, b, ref); err != nil {
		t.Fatal(err)
	}
}

func TestIsolatedRecordingStopsAtOriginalDeadlineWithoutController(t *testing.T) {
	image, gateway, seccomp := os.Getenv("TESTSTUDIO_RECORDING_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_SECCOMP")
	if image == "" || gateway == "" || seccomp == "" {
		t.Skip("supply the cached recording and gateway images and pinned sandbox policy")
	}
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!doctype html><h1>Independent deadline fixture</h1>")
	}))
	target.Listener.Close()
	target.Listener = listener
	target.Start()
	t.Cleanup(target.Close)
	origin := fmt.Sprintf("http://host.docker.internal:%d", listener.Addr().(*net.TCPAddr).Port)
	b := recordingworker.Binding{ID: domain.NewID(), Actor: "tester", Project: "fixture", Suite: "browser", URL: origin, Deadline: time.Now().Add(5 * time.Second).UTC()}
	runtime := &Runtime{Image: image, GatewayImage: gateway, Seccomp: seccomp, Directory: t.TempDir()}
	ref, err := runtime.Plan(b, browserpolicy.Target{Origin: origin, AllowPrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := runtime.Cleanup(ctx, b, ref); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	driver, err := runtime.Start(ctx, b, ref, browserpolicy.Target{Origin: origin, AllowPrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{ref.Container, ref.Gateway} {
		stopped := false
		for time.Now().Before(b.Deadline.Add(4 * time.Second)) {
			data, err := exec.CommandContext(ctx, runtime.binary(), "inspect", name, "--format", "{{.State.Running}}").Output()
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(data)) == "false" {
				stopped = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !stopped {
			t.Fatal("an owned recording process outlived its original deadline")
		}
	}
	if snapshot, err := driver.Read(ctx, b.ID); err != nil || snapshot.State != "failed" || snapshot.Code == "" {
		t.Fatal("independent deadline lost captured source", err)
	}
}
