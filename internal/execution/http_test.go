package execution

import (
	"context"
	"crypto/x509"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
)

func definition(path string) catalog.Test {
	return catalog.Test{ID: "HEALTH-1", Title: "Example check", Type: "https", Runner: "http", TimeoutSeconds: 1, Check: &catalog.HTTPCheck{Endpoint: "api", Path: path, Method: "GET", StatusCodes: []int{200}}}
}

func TestHTTPAssertionsAndRedirectIsolation(t *testing.T) {
	var followed atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status":"up","items":[{"id":1}]}`)
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/target", 302) })
	mux.HandleFunc("/target", func(w http.ResponseWriter, r *http.Request) { followed.Store(true); w.WriteHeader(200) })
	server := httptest.NewServer(mux)
	defer server.Close()
	env := projects.Environment{Stage: "local", Endpoints: map[string]string{"api": server.URL}}
	test := definition("/up")
	test.Check.JSON = map[string]any{"status": "up", "items.0.id": 1}
	test.Check.Headers = map[string]string{"Content-Type": "application/json"}
	r := (&HTTP{}).Execute(t.Context(), test, env)
	if r.State != "passed" || r.HTTP.Status != 200 || r.DurationMs <= 0 {
		t.Fatalf("valid JSON check: %+v", r)
	}
	test.Check.JSON["status"] = "down"
	r = (&HTTP{}).Execute(t.Context(), test, env)
	if r.State != "failed" || r.FailureKind != "assertion" {
		t.Fatalf("JSON assertion: %+v", r)
	}
	r = (&HTTP{}).Execute(t.Context(), definition("/redirect"), env)
	if r.State != "failed" || r.FailureKind != "http-status" || followed.Load() {
		t.Fatalf("redirect escaped: %+v followed=%v", r, followed.Load())
	}
	if r.HTTP.DurationMs <= 0 {
		t.Fatal("failed check lost its measured duration")
	}
}

func TestHTTPDeadlineAndTLSVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	env := projects.Environment{Stage: "local", Endpoints: map[string]string{"api": server.URL}}
	r := (&HTTP{}).Execute(t.Context(), definition("/slow"), env)
	if r.FailureKind != "timeout" || r.DurationMs > 3000 {
		t.Fatalf("timeout: %+v", r)
	}
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer secure.Close()
	secure.Config.ErrorLog = log.New(io.Discard, "", 0)
	env.Endpoints["api"] = secure.URL
	r = (&HTTP{}).Execute(t.Context(), definition("/"), env)
	if r.FailureKind != "tls" {
		t.Fatalf("untrusted TLS passed: %+v", r)
	}
	pool := x509.NewCertPool()
	pool.AddCert(secure.Certificate())
	r = (&HTTP{RootCAs: pool}).Execute(t.Context(), definition("/"), env)
	if r.State != "passed" || r.HTTP.CertificateExpires == nil {
		t.Fatalf("explicit trust bundle: %+v", r)
	}
	test := definition("/")
	test.Check.MinCertificateDays = int(time.Until(secure.Certificate().NotAfter)/(24*time.Hour)) + 1
	r = (&HTTP{RootCAs: pool}).Execute(t.Context(), test, env)
	if r.FailureKind != "tls" {
		t.Fatalf("certificate threshold: %+v", r)
	}
}

func TestHTTPRejectsUnsupportedTargetsBeforeExecution(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	env := projects.Environment{Stage: "production", Endpoints: map[string]string{"api": server.URL}}
	r := (&HTTP{}).Execute(t.Context(), definition("/"), env)
	if r.State != "failed" || requests.Load() != 0 {
		t.Fatalf("plain production HTTP was sent: %+v", r)
	}
	env.Stage = "local"
	test := definition("//other.example/path")
	r = (&HTTP{}).Execute(t.Context(), test, env)
	if r.State != "failed" || requests.Load() != 0 {
		t.Fatal("absolute destination was sent")
	}
	test = definition("/")
	test.Check.Method = "HEAD"
	test.Check.RequiredText = "not valid on HEAD"
	r = (&HTTP{}).Execute(t.Context(), test, env)
	if r.State != "failed" || requests.Load() != 0 {
		t.Fatal("invalid HEAD definition reached the server")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r = (&HTTP{}).Execute(ctx, definition("/"), env)
	if r.FailureKind != "cancelled" {
		t.Fatalf("cancelled check: %+v", r)
	}
}

func TestNetworkScopeBlocksMetadataAndPrivateDefaults(t *testing.T) {
	for _, address := range []string{"169.254.169.254", "169.254.170.2", "100.100.100.200", "168.63.129.16", "fd00:ec2::254", "::ffff:169.254.169.254"} {
		if permittedAddress(net.ParseIP(address), true) {
			t.Errorf("metadata address allowed: %s", address)
		}
	}
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "::1"} {
		if permittedAddress(net.ParseIP(address), false) {
			t.Errorf("implicit private target allowed: %s", address)
		}
		if !permittedAddress(net.ParseIP(address), true) {
			t.Errorf("explicit private target rejected: %s", address)
		}
	}
}
