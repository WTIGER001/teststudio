package localplaywright

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/sources"
)

func descendantPIDs(t *testing.T, root int, known map[int]bool) map[int]bool {
	t.Helper()
	output, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		t.Fatal(err)
	}
	parents := map[int]int{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		parent, _ := strconv.Atoi(fields[1])
		parents[pid] = parent
	}
	current := map[int]bool{root: true}
	for changed := true; changed; {
		changed = false
		for pid, parent := range parents {
			if current[parent] && !current[pid] {
				current[pid] = true
				known[pid] = true
				changed = true
			}
		}
	}
	alive := map[int]bool{}
	for pid := range known {
		if _, ok := parents[pid]; ok {
			alive[pid] = true
		}
	}
	return alive
}

func TestRecordingLocalPlaywrightWritesSourceAndClosesOwnedBrowsers(t *testing.T) {
	if os.Getenv("TESTSTUDIO_NATIVE_RECORDING") != "1" {
		t.Skip("enable the targeted local desktop recorder check")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	driver, err := New(t.Context(), node, filepath.Join(root, "runners/playwright/node_modules/playwright/cli.js"), filepath.Join(root, "scripts/record-browser.mjs"), filepath.Join(t.TempDir(), "recordings"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><title>TestStudio recording verification</title><h1>Recording verification</h1><button>Continue</button>"))
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		if err := driver.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := driver.Start(t.Context(), "native-recording", server.URL); err != nil {
		t.Fatal(err)
	}
	driver.mu.Lock()
	pid := driver.processes["native-recording"].command.Process.Pid
	driver.mu.Unlock()
	known := map[int]bool{pid: true}
	deadline := time.Now().Add(20 * time.Second)
	var code string
	for time.Now().Before(deadline) {
		descendantPIDs(t, pid, known)
		snapshot, err := driver.Read(t.Context(), "native-recording")
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.State == "failed" {
			t.Fatal(snapshot.Error)
		}
		code = snapshot.Code
		if strings.Contains(code, "page.goto(") && strings.Contains(code, server.URL) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(code, server.URL) {
		t.Fatal("real recorder did not emit the local navigation")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	stopped, err := driver.Stop(ctx, "native-recording")
	if err != nil || stopped.State != "ready" || !strings.Contains(stopped.Code, server.URL) {
		t.Fatal("recording did not retain source on close", stopped.State, err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(descendantPIDs(t, pid, known)) == 0 {
			draft, err := sources.RecordedDraft(recording.Session{Input: recording.Input{TestID: "NATIVE-RECORD-001", Title: "Native recorded navigation", URL: server.URL}, State: "ready", Code: stopped.Code})
			if err != nil || !strings.Contains(draft.Files[0].Content, "@NATIVE-RECORD-001") {
				t.Fatal("real generated source did not become a tagged catalog draft", err)
			}
			t.Logf("real recorded navigation converted to tagged source; %d owned processes exited", len(known))
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("owned recording browser processes remain")
}
