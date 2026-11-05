package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServiceCommandExposesCompanionOnlyWhenExplicitlyEnabled(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			root := t.TempDir()
			ui := filepath.Join(root, "ui")
			if err := os.Mkdir(ui, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(ui, "index.html"), []byte("<!doctype html><title>Recorder command check</title>"), 0600); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			address := listener.Addr().String()
			listener.Close()
			origin := "http://" + address
			options := []string{"--serve-role=api", "--local", "--addr=" + address, "--public-url=" + origin, "--artifact-origin=http://127.0.0.1:1", "--database=" + filepath.Join(root, "studio.db"), "--artifact-directory=" + filepath.Join(root, "artifacts"), "--ui=" + ui}
			if enabled {
				options = append(options, "--recording-companion=true")
			}
			stop := startOwnedServiceCommand(t, address, options...)
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Jar: jar, Timeout: time.Second}
			ready := false
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				response, err := client.Get(origin + "/healthz")
				if err == nil {
					response.Body.Close()
					if response.StatusCode == 200 {
						ready = true
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !ready {
				t.Fatal("owned API did not become ready")
			}
			csrf := ""
			call := func(method, path string, value any) map[string]any {
				t.Helper()
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				request, err := http.NewRequestWithContext(t.Context(), method, origin+path, bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Origin", origin)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("X-CSRF-Token", csrf)
				if method == "PUT" {
					request.Header.Set("If-Match", "0")
				}
				response, err := client.Do(request)
				if err != nil {
					t.Fatal("owned API request failed")
				}
				defer response.Body.Close()
				if response.StatusCode != 200 {
					t.Fatalf("%s %s: status %d", method, path, response.StatusCode)
				}
				var result map[string]any
				if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				return result
			}
			call("POST", "/auth/local", map[string]any{})
			csrf = call("GET", "/auth/session", nil)["session"].(map[string]any)["csrfToken"].(string)
			call("POST", "/api/v1/projects", map[string]string{"id": "recording", "name": "Recording"})
			call("PUT", "/api/v1/projects/recording/suites/browser", map[string]any{"id": "browser", "name": "Browser", "applications": []string{}})
			base := "/api/v1/projects/recording/suites/browser/browser-recordings"
			settings := call("GET", base+"/options", nil)
			mode := "disabled"
			if enabled {
				mode = "companion"
			}
			if settings["enabled"] != enabled || settings["mode"] != mode {
				t.Fatal("command did not preserve explicit recording mode")
			}
			if enabled {
				row := call("POST", base, map[string]string{"id": "session", "testId": "REC-001", "title": "Desktop test", "url": "https://example.test"})
				value := row["data"].(map[string]any)
				if value["state"] != "pairing" || value["mode"] != "companion" || value["code"] != nil {
					t.Fatal("API tried to open a native browser instead of waiting for a desktop")
				}
				call("POST", base+"/session/actions/cancel", nil)
			}
			stop()
		})
	}
}
