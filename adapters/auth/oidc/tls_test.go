package oidc_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestOIDCPrivateAuthoritiesKeepCertificateValidationAndDefaultTrust(t *testing.T) {
	ca := testkit.NewCertificateAuthority(t, "Private sign-in authority")
	file := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(file, ca.PEM, 0600); err != nil {
		t.Fatal(err)
	}
	client, err := authoidc.TLSClient(file)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)
	other := testkit.NewCertificateAuthority(t, "Untrusted sign-in authority")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	valid := testkit.NewTLSServer(t, ca, "127.0.0.1", time.Hour, handler)
	wrong := testkit.NewTLSServer(t, ca, "wrong.invalid", time.Hour, handler)
	expired := testkit.NewTLSServer(t, ca, "127.0.0.1", -time.Hour, handler)
	untrusted := testkit.NewTLSServer(t, other, "127.0.0.1", time.Hour, handler)
	for _, scenario := range []struct {
		name, url string
		accepted  bool
	}{{"valid", valid.URL, true}, {"hostname", wrong.URL, false}, {"expired", expired.URL, false}, {"untrusted", untrusted.URL, false}} {
		t.Run(scenario.name, func(t *testing.T) {
			response, err := client.Get(scenario.url)
			if response != nil {
				response.Body.Close()
			}
			if (err == nil) != scenario.accepted {
				t.Fatalf("certificate validation accepted=%v: %v", err == nil, err)
			}
		})
	}
	if response, err := http.DefaultClient.Get(valid.URL); err == nil {
		response.Body.Close()
		t.Fatal("OIDC authorities changed default HTTP trust")
	}
	if client, err := authoidc.TLSClient(""); client != nil || err != nil {
		t.Fatal("unconfigured OIDC trust must preserve the default client")
	}
}

func TestOIDCAuthorityConfigurationRejectsInvalidFilesWithoutPrivatePaths(t *testing.T) {
	for _, data := range []string{"", "private-file-canary", strings.Repeat("x", 65537)} {
		file := filepath.Join(t.TempDir(), "private-path-canary.pem")
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if client, err := authoidc.TLSClient(file); client != nil || err == nil || strings.Contains(err.Error(), "canary") {
			t.Fatal("invalid OIDC authority configuration did not return a fixed failure")
		}
	}
}
