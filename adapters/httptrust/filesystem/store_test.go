package filesystem_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httptrust "github.com/wtiger001/teststudio/adapters/httptrust/filesystem"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestHTTPTrustResolutionPreservesTimeoutAndCancellation(t *testing.T) {
	store, err := httptrust.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"timeout", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			if kind == "timeout" {
				cancel()
				ctx, cancel = context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
			} else {
				cancel()
			}
			defer cancel()
			test := catalog.Test{ID: "CANCEL-001", TimeoutSeconds: 2, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/", Method: "GET", StatusCodes: []int{200}}}
			result := (&execution.HTTP{Trust: store}).Execute(ctx, "alpha", test, projects.Environment{Stage: "local", Endpoints: map[string]string{"api": "https://example.invalid"}})
			if result.FailureKind != kind || kind == "timeout" && result.State != "timed-out" {
				t.Fatal("trust resolution rewrites cancellation as configuration failure", result)
			}
		})
	}
}

func trustFile(t *testing.T, data []byte) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "authority.pem")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestProjectOriginTrustKeepsTLSValidationAndNetworkGuards(t *testing.T) {
	ca := testkit.NewCertificateAuthority(t, "Owned HTTP trust authority")
	var hits atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(200) })
	server := testkit.NewTLSServer(t, ca, "127.0.0.1", 72*time.Hour, handler)
	other := testkit.NewTLSServer(t, ca, "127.0.0.1", 72*time.Hour, handler)
	wrong := testkit.NewTLSServer(t, ca, "wrong.invalid", 72*time.Hour, handler)
	expired := testkit.NewTLSServer(t, ca, "127.0.0.1", -time.Hour, handler)
	file := trustFile(t, ca.PEM)
	bindings := []httptrust.Binding{}
	for index, origin := range []string{server.URL, wrong.URL, expired.URL} {
		bindings = append(bindings, httptrust.Binding{ID: []string{"valid", "wrong-host", "expired"}[index], Name: "Owned project CA", Project: "alpha", Origin: origin, File: file})
	}
	store, err := httptrust.New(bindings)
	if err != nil {
		t.Fatal(err)
	}
	executor := &execution.HTTP{Trust: store}
	definition := catalog.Test{ID: "TLS-001", Type: "https", Runner: "http", TimeoutSeconds: 2, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/health", Method: "GET", StatusCodes: []int{200}, MinCertificateDays: 1}}
	environment := projects.Environment{Stage: "local", Endpoints: map[string]string{"api": server.URL}}
	passed := executor.Execute(t.Context(), "alpha", definition, environment)
	if passed.State != "passed" || passed.HTTP.TrustBundle == nil || passed.HTTP.TrustBundle.ID != "valid" || passed.HTTP.CertificateExpires == nil {
		t.Fatal("registered project trust does not validate its private endpoint", passed)
	}
	for _, scenario := range []struct{ name, project, origin, stage, kind string }{
		{"other-project", "beta", server.URL, "local", "tls"},
		{"other-origin", "alpha", other.URL, "local", "tls"},
		{"hostname", "alpha", wrong.URL, "local", "tls"},
		{"expired", "alpha", expired.URL, "local", "tls"},
		{"network-policy", "alpha", server.URL, "production", "configuration"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			target := projects.Environment{Stage: scenario.stage, Endpoints: map[string]string{"api": scenario.origin}}
			result := executor.Execute(t.Context(), scenario.project, definition, target)
			if result.State != "failed" || result.FailureKind != scenario.kind {
				t.Fatal("custom trust bypasses project, origin, certificate, or network validation", result)
			}
		})
	}
	definition.Check.MinCertificateDays = 4
	threshold := executor.Execute(t.Context(), "alpha", definition, environment)
	if threshold.FailureKind != "tls" || threshold.HTTP.CertificateExpires == nil || hits.Load() != 2 {
		t.Fatal("certificate lifetime or request isolation is lost", threshold, hits.Load())
	}
}

func TestTrustConfigurationRejectsInvalidFilesAndKeepsStableCertificateDigests(t *testing.T) {
	a, b := testkit.NewCertificateAuthority(t, "Owned A"), testkit.NewCertificateAuthority(t, "Owned B")
	file := trustFile(t, append(append([]byte{}, a.PEM...), b.PEM...))
	binding := httptrust.Binding{ID: "private", Name: "Private service CA", Project: "alpha", Origin: "https://EXAMPLE.invalid:443/", File: file}
	first, err := httptrust.New([]httptrust.Binding{binding})
	if err != nil {
		t.Fatal(err)
	}
	_, original, err := first.Resolve(t.Context(), "alpha", "https://example.invalid")
	if err != nil || original == nil {
		t.Fatal("canonical origin is not matched", err)
	}
	if err := os.WriteFile(file, append(append(append([]byte{}, b.PEM...), a.PEM...), a.PEM...), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := httptrust.New([]httptrust.Binding{binding})
	if err != nil {
		t.Fatal(err)
	}
	_, reordered, _ := second.Resolve(t.Context(), "alpha", "https://example.invalid")
	if original.Digest != reordered.Digest {
		t.Fatal("ordering or duplicate certificates change trust identity")
	}
	for _, data := range [][]byte{nil, []byte("private-file-canary"), append(append([]byte{}, a.PEM...), []byte("trailing-private-canary")...), []byte("-----BEGIN PRIVATE KEY-----\nprivate-key-canary\n-----END PRIVATE KEY-----"), []byte(strings.Repeat("x", 65537))} {
		binding.File = trustFile(t, data)
		_, err := httptrust.New([]httptrust.Binding{binding})
		if err == nil || strings.Contains(err.Error(), "canary") || strings.Contains(err.Error(), binding.File) {
			t.Fatal("invalid certificate source is accepted or exposed", err)
		}
	}
	binding.File = file
	if _, err := httptrust.New([]httptrust.Binding{binding, binding}); err == nil {
		t.Fatal("duplicate project origin is accepted")
	}
	config := filepath.Join(t.TempDir(), "trust.json")
	encoded, _ := json.Marshal([]httptrust.Binding{binding})
	if err := os.WriteFile(config, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := httptrust.Load(config); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, append(encoded, []byte("{}")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := httptrust.Load(config); err == nil {
		t.Fatal("trailing configuration is accepted")
	}
}
