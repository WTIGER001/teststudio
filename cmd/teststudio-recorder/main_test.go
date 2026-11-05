package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/companion"
	"github.com/wtiger001/teststudio/internal/recording"
)

func TestRecorderCommandPairsCapturesAndClosesOwnedHelper(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js is required for the native helper integration")
	}
	root := t.TempDir()
	cli, helper := filepath.Join(root, "cli.mjs"), filepath.Join(root, "helper.mjs")
	if err := os.WriteFile(cli, []byte("console.log('Version "+recording.PlaywrightVersion+"')"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helper, []byte(`import { writeFileSync } from 'node:fs';
const [, output, address] = process.argv.slice(2);
writeFileSync(output + '.pid', String(process.pid));
writeFileSync(output, "import { test } from '@playwright/test';\ntest('test', async ({ page }) => { await page.goto(" + JSON.stringify(address) + "); });\n");
process.stdin.resume();
process.stdin.on('end', () => process.exit(0));
`), 0600); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var pair recording.PairRequest
	control := recording.CompanionControl{State: "starting", DeadlineAt: time.Now().UTC().Add(time.Minute)}
	var final recording.CompanionSnapshot
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/recording-companion/sessions/session/pair":
			if err := json.NewDecoder(r.Body).Decode(&pair); err != nil {
				t.Error(err)
				w.WriteHeader(422)
				return
			}
			json.NewEncoder(w).Encode(recording.CompanionPacket{Input: recording.Input{ID: "session", URL: "https://example.test"}, Project: "example", Suite: "browser", Control: control})
		case "/api/v1/recording-companion/sessions/session/control":
			json.NewEncoder(w).Encode(control)
		case "/api/v1/recording-companion/sessions/session/snapshots":
			var input struct {
				Instance string                      `json:"instance"`
				Snapshot recording.CompanionSnapshot `json:"snapshot"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
				w.WriteHeader(422)
				return
			}
			if r.Header.Get("Authorization") != "Bearer "+pair.DeviceToken || input.Instance != pair.Instance || input.Snapshot.Sequence != control.Sequence+1 {
				t.Error("command lost bound device or snapshot sequence")
				w.WriteHeader(403)
				return
			}
			final = input.Snapshot
			control.State, control.Sequence = final.State, final.Sequence
			if final.Code != "" && final.State == "recording" {
				control.State, control.Action = "stopping", "stop"
			}
			json.NewEncoder(w).Encode(control)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	directory := filepath.Join(root, "desktop")
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	var output bytes.Buffer
	err := run(ctx, []string{"--server", server.URL, "--directory", directory, "--playwright-cli", cli, "--helper", helper}, strings.NewReader("session:"+token+"\n"), &output)
	if err != nil {
		t.Fatal("native command did not finish", err)
	}
	mu.Lock()
	complete := final.State == "ready" && strings.Contains(final.Code, "https://example.test") && control.Action == "stop"
	device := pair.DeviceToken
	mu.Unlock()
	if !complete {
		t.Fatal("command did not submit captured source on owner stop")
	}
	if strings.Contains(output.String(), token) || strings.Contains(output.String(), device) {
		t.Fatal("command output disclosed a connection credential")
	}
	pidBytes, err := os.ReadFile(filepath.Join(directory, "captures", "session", "source.spec.ts.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil || syscall.Kill(pid, 0) == nil {
		t.Fatal("owned native helper remains alive after command completion")
	}
	journal, err := companion.NewJournal(directory)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := journal.Read("session")
	if err != nil || saved.PairingToken != "" || saved.Pending != nil || saved.Final == nil {
		t.Fatal("command did not retain private completed connection", err)
	}
	lock, err := journal.Lock()
	if err != nil {
		t.Fatal("command retained its recorder lock", err)
	}
	lock.Close()
}

func TestRecorderCommandRejectsUnsafeConnectionBeforeContactingServer(t *testing.T) {
	for _, args := range [][]string{{"--server", "http://remote.example.test"}, {"--server", "https://studio.example.test", "secret-code"}} {
		if err := run(t.Context(), append(args, "--directory", filepath.Join(t.TempDir(), "desktop")), strings.NewReader("unused"), io.Discard); err == nil {
			t.Fatal("unsafe command accepted")
		}
	}
}
