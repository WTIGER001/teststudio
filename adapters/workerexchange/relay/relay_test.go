package relay

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func request(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, "https://workers.example.test"+path, body)
	r.URL.Scheme, r.URL.Host = "", ""
	r.Header.Set("Authorization", "Bearer "+base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	r.Header.Set("X-TestStudio-Worker-Instance", "worker-instance")
	return r
}

func TestWorkerRelayCarriesExistingClientProtocolOverLocalTLS(t *testing.T) {
	var mu sync.Mutex
	paths := []string{}
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		if r.Header.Get("Authorization") != "Bearer "+token || r.Header.Get("X-TestStudio-Worker-Instance") != "worker-instance" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Forwarded-For") != "" {
			t.Error("worker scope changed during relay")
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(workerprotocol.Packet{Execution: "execution-1", Deadline: time.Now().Add(time.Minute), Input: json.RawMessage(`{"test":"own fixture"}`)})
			return
		}
		if strings.Contains(r.URL.Path, "/attachments/") && string(data) != "local screenshot bytes" {
			t.Error("attachment bytes changed")
		}
		w.Header().Set("Set-Cookie", "private-backend-canary")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()
	front := httptest.NewUnstartedServer(nil)
	relay, err := New("https://"+front.Listener.Addr().String(), backend.URL, backend.Client().Transport)
	if err != nil {
		front.Close()
		t.Fatal(err)
	}
	defer relay.Close()
	front.Config.Handler = relay
	front.StartTLS()
	defer front.Close()
	client, err := workerclient.New(front.URL, "execution-1", "worker-instance", token, front.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	packet, err := client.Input(t.Context())
	if err != nil || string(packet.Input) != `{"test":"own fixture"}` {
		t.Fatal("worker input did not survive the relay", err)
	}
	if err := client.Started(t.Context(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := client.Submit(t.Context(), workerprotocol.Submission{ProcessExitCode: 0}); err != nil {
		t.Fatal(err)
	}
	if err := client.Upload(t.Context(), "shot one.png", strings.NewReader("local screenshot bytes")); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	expected := []string{"GET /internal/worker/v1/executions/execution-1/input", "POST /internal/worker/v1/executions/execution-1/started", "PUT /internal/worker/v1/executions/execution-1/report", "PUT /internal/worker/v1/executions/execution-1/attachments/shot%20one.png"}
	if strings.Join(paths, "\n") != strings.Join(expected, "\n") {
		t.Fatal("relay changed the four-operation contract", paths)
	}
}

func TestWorkerRelayRejectsUnscopedRequestsBeforeBackendAccess(t *testing.T) {
	calls := 0
	g, err := New("https://workers.example.test", "https://controller.example.test", transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("backend-private-canary")
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	cases := map[string]func(*http.Request){
		"ui-route":           func(r *http.Request) { r.URL.Path = "/projects" },
		"api-route":          func(r *http.Request) { r.URL.Path = "/api/projects" },
		"unknown-operation":  func(r *http.Request) { r.URL.Path = "/internal/worker/v1/executions/execution-1/resume" },
		"wrong-method":       func(r *http.Request) { r.Method = "DELETE" },
		"foreign-host":       func(r *http.Request) { r.Host = "foreign.invalid" },
		"absolute-form":      func(r *http.Request) { r.URL.Scheme = "https"; r.URL.Host = "foreign.invalid" },
		"query":              func(r *http.Request) { r.URL.RawQuery = "ticket=secret-canary" },
		"empty-query":        func(r *http.Request) { r.URL.ForceQuery = true },
		"cookie":             func(r *http.Request) { r.Header.Set("Cookie", "private-session") },
		"browser-origin":     func(r *http.Request) { r.Header.Set("Origin", "https://browser.invalid") },
		"duplicate-token":    func(r *http.Request) { r.Header.Add("Authorization", r.Header.Get("Authorization")) },
		"duplicate-instance": func(r *http.Request) { r.Header.Add("X-TestStudio-Worker-Instance", "other") },
		"missing-token":      func(r *http.Request) { r.Header.Del("Authorization") },
		"path-token":         func(r *http.Request) { r.URL.Path = "/internal/worker/v1/executions/execution-1/input/secret" },
		"compressed-message": func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") },
		"encoded-selector":   func(r *http.Request) { r.URL.RawPath = "/internal/worker/v1/executions/%65xecution-1/input" },
		"unsafe-file": func(r *http.Request) {
			r.Method = "PUT"
			r.URL.Path = "/internal/worker/v1/executions/execution-1/attachments/.."
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := request(t, "GET", "/internal/worker/v1/executions/execution-1/input", nil)
			mutate(r)
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			if w.Code != 404 || strings.Contains(w.Body.String(), "canary") {
				t.Fatal("unscoped request was accepted", w.Code, w.Body.String())
			}
		})
	}
	if calls != 0 {
		t.Fatal("unscoped request reached backend", calls)
	}
}

func TestWorkerRelayBoundsBodiesAndNeverForwardsBackendDiagnostics(t *testing.T) {
	for _, mode := range []string{"redirect", "private-error", "wrong-execution", "unknown-packet-field", "oversized-input", "large-declared-response", "valid-error"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			g, err := New("https://workers.example.test", "https://controller.example.test", transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				status := 200
				body := `{"executionId":"execution-1","deadline":"` + time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano) + `","input":{}}`
				header := http.Header{"Set-Cookie": []string{"backend-secret-canary"}, "Location": []string{"https://foreign.invalid/secret-canary"}}
				length := int64(len(body))
				switch mode {
				case "redirect":
					status = 307
					body = "private-backend-canary"
				case "private-error":
					return nil, errors.New("private-backend-canary")
				case "wrong-execution":
					body = strings.Replace(body, "execution-1", "foreign", 1)
				case "unknown-packet-field":
					body = strings.TrimSuffix(body, "}") + `,"credential":"backend-secret-canary"}`
				case "oversized-input":
					body = strings.Repeat("x", execution.MaxWorkItemBytes+8193)
					length = -1
				case "large-declared-response":
					length = execution.MaxWorkItemBytes + 8193
				case "valid-error":
					status = 409
					body = "backend-secret-canary"
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), ContentLength: length, Request: r}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			w := httptest.NewRecorder()
			g.ServeHTTP(w, request(t, "GET", "/internal/worker/v1/executions/execution-1/input", nil))
			expected := 503
			if mode == "valid-error" {
				expected = 409
			}
			if w.Code != expected || calls != 1 || strings.Contains(w.Body.String(), "canary") || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Location") != "" {
				t.Fatal("backend detail or redirect escaped", w.Code, calls, w.Body.String())
			}
		})
	}
	calls := 0
	g, _ := New("https://workers.example.test", "https://controller.example.test", transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return nil, errors.New("unused") }))
	defer g.Close()
	r := request(t, "POST", "/internal/worker/v1/executions/execution-1/started", strings.NewReader(strings.Repeat("x", 1025)))
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != 413 || calls != 0 {
		t.Fatal("oversized declared request reached backend")
	}
}

func TestWorkerRelayIsFixedOriginAndBoundsConcurrentForwarding(t *testing.T) {
	for _, pair := range [][2]string{{"http://workers.example.test", "https://controller.example.test"}, {"https://workers.example.test", "https://user:private@controller.example.test"}, {"https://workers.example.test", "https://controller.example.test/path"}, {"https://workers.example.test", "https://controller.example.test?secret=canary"}, {"https://workers.example.test", "https://WORKERS.example.test:0443"}, {"https://workers.example.test", "https://controller.example.test:0"}} {
		if _, err := New(pair[0], pair[1], nil); err == nil || strings.Contains(err.Error(), "private") {
			t.Fatal("invalid or recursive relay origin accepted", err)
		}
	}
	g, err := New("https://workers.example.test", "https://controller.example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	transport := g.client.Transport.(*http.Transport)
	if transport.Proxy != nil || !transport.DisableCompression || transport.TLSClientConfig.InsecureSkipVerify || transport.MaxConnsPerHost != 8 {
		t.Fatal("relay enabled implicit proxy or unbounded transport")
	}
	for range cap(g.slots) {
		g.slots <- struct{}{}
	}
	w := httptest.NewRecorder()
	g.ServeHTTP(w, request(t, "GET", "/internal/worker/v1/executions/execution-1/input", nil))
	if w.Code != 503 {
		t.Fatal("relay saturation did not reject promptly")
	}
	health := httptest.NewRecorder()
	probe := httptest.NewRequest("GET", "/healthz", nil)
	probe.Host = "127.0.0.1"
	g.ServeHTTP(health, probe)
	if health.Code != 200 || !strings.Contains(health.Body.String(), `"status":"up"`) {
		t.Fatal("local process probe depends on public Host or backend")
	}
	for range cap(g.slots) {
		<-g.slots
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r := request(t, "GET", "/internal/worker/v1/executions/execution-1/input", nil).WithContext(ctx)
	g.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() == nil {
			t.Error("caller cancellation was lost")
		}
		return nil, r.Context().Err()
	})
	w = httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatal("cancelled relay request did not fail closed")
	}
}

func TestWorkerRelayStreamsBoundedWritesWithoutForwardingOtherAuthority(t *testing.T) {
	calls := 0
	g, err := New("https://workers.example.test", "https://backend.example.test:8443", transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Scheme != "https" || r.URL.Host != "backend.example.test:8443" || r.Host != "backend.example.test:8443" || r.URL.Path != "/internal/worker/v1/executions/execution-1/started" || r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-TestStudio-Project") != "" || r.Header.Get("X-MSI-IDENTITY") != "" || r.Header.Get("User-Agent") != "TestStudio-worker-relay/v1" {
			t.Error("relay widened destination or caller authority")
		}
		data, err := io.ReadAll(r.Body)
		if err == nil || len(data) != 1024 {
			t.Error("chunked write exceeded the operation bound", len(data), err)
		}
		return nil, errors.New("private-upload-canary")
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	r := request(t, "POST", "/internal/worker/v1/executions/execution-1/started", strings.NewReader(strings.Repeat("x", 1025)))
	r.ContentLength = -1
	r.Header.Set("X-Forwarded-For", "private-client-canary")
	r.Header.Set("X-TestStudio-Project", "foreign-project")
	r.Header.Set("X-MSI-IDENTITY", "private-identity-canary")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	if w.Code != 503 || calls != 1 || strings.Contains(w.Body.String(), "canary") {
		t.Fatal("failed upload was replayed or exposed diagnostics", w.Code, calls, w.Body.String())
	}
}
