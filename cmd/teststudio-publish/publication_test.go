package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPublisherManagedLoginReviewAndPublicationOverLocalHTTP(t *testing.T) {
	cookie := strings.Repeat("c", 43)
	csrf := "private-request-token"
	origin := ""
	published := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "GET" && r.Header.Get("Origin") != origin {
			t.Error("mutation origin missing")
		}
		if r.URL.Path == "/auth/local" {
			http.SetCookie(w, &http.Cookie{Name: "teststudio_session", Value: cookie, Path: "/", HttpOnly: true})
			w.Write([]byte(`{"signedIn":true}`))
			return
		}
		stored, err := r.Cookie("teststudio_session")
		if err != nil || stored.Value != cookie {
			t.Error("managed session cookie missing")
		}
		if r.URL.Path == "/auth/session" {
			_ = json.NewEncoder(w).Encode(map[string]any{"session": map[string]any{"csrfToken": csrf, "expiresAt": time.Now().Add(time.Hour)}})
			return
		}
		if r.Method != "GET" && r.Header.Get("X-CSRF-Token") != csrf {
			t.Error("managed request token missing")
		}
		base := "/api/v1/projects/example/suites/code"
		switch r.URL.Path {
		case base + "/source-preparations", base + "/source-preparations/job":
			if r.Method == "POST" && r.Header.Get("Idempotency-Key") != "stable-key" {
				t.Error("preparation retry key missing")
			}
			state := "ready"
			if published {
				state = "published"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "job", "revision": 2, "data": map[string]string{"id": "job", "state": state, "phase": "ready", "uploadId": "upload"}})
		case base + "/source-uploads/upload":
			w.Write([]byte(`{"upload":{"id":"upload","revision":1,"data":{"id":"upload","digest":"source-digest"}},"plan":{"changes":[{"action":"create","test":{"id":"CASE-1","title":"Managed code","type":"unit","safety":"safe"}}],"issues":[]}}`))
		case base + "/source-uploads/upload/publish":
			if r.Header.Get("If-Match") != "\"1\"" {
				t.Error("source review revision missing")
			}
			var body struct {
				Activate bool `json:"activate"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil || !body.Activate {
				t.Error("explicit activation choice missing")
			}
			published = true
			w.Write([]byte(`{"id":"release","activatedOnPublication":true}`))
		case "/auth/logout":
			w.Write([]byte(`{"signedOut":true}`))
		default:
			t.Error("unexpected CLI request", r.URL.Path)
			http.Error(w, "unexpected", 404)
		}
	}))
	defer server.Close()
	origin = server.URL
	filename := filepath.Join(t.TempDir(), "session.json")
	var output bytes.Buffer
	commands := [][]string{
		{"login", "--server", server.URL, "--local", "--session-file", filename},
		{"prepare", "--project", "example", "--suite", "code", "--image", "sha256:" + strings.Repeat("a", 64), "--key", "stable-key", "--session-file", filename},
		{"status", "--project", "example", "--suite", "code", "--job", "job", "--review", "--session-file", filename},
		{"publish", "--project", "example", "--suite", "code", "--job", "job", "--activate", "--session-file", filename},
		{"logout", "--session-file", filename},
	}
	for _, args := range commands {
		if err := run(t.Context(), args, &output); err != nil {
			t.Fatalf("%s: %v", args[0], err)
		}
	}
	if !published || !strings.Contains(output.String(), "Published release release") || !strings.Contains(output.String(), "CASE-1") {
		t.Fatal("CLI did not review and publish the prepared catalog")
	}
	if strings.Contains(output.String(), cookie) || strings.Contains(output.String(), csrf) {
		t.Fatal("CLI printed managed credentials")
	}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatal("logout retained the session file")
	}
}
