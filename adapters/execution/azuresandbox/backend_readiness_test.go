package azuresandbox

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestSandboxBackendEndpointBindsExistingRelayTLSReadinessContract(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKBackendEndpoint(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := reader.Check(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	b := s.Placement.Services.Backend
	template := s.relayDefinition().Properties.Template
	if *template.Containers[0].Args[3] != "--backend-address="+backendIP(w)["privateIPAddress"].(string) {
		t.Fatal("readiness and forwarding address differs from the observed endpoint")
	}
	readiness := 0
	for _, probe := range template.Containers[0].Probes {
		if *probe.Type == a.TypeReadiness {
			readiness++
			if *probe.HTTPGet.Path != "/readyz" || *probe.TimeoutSeconds != 4 {
				t.Fatal("platform readiness no longer checks the backend")
			}
		}
	}
	if readiness != 1 {
		t.Fatal("trusted relay lacks one backend readiness probe")
	}
	backend, _ := url.Parse(b.Origin)
	ca := testkit.NewCertificateAuthority(t, "Owned sandbox backend readiness")
	var unavailable atomic.Bool
	var dials atomic.Int32
	server := testkit.NewTLSServer(t, ca, backend.Hostname(), 5*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != backend.Host || r.TLS == nil || r.TLS.ServerName != backend.Hostname() || r.URL.Path != "/healthz" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("X-TestStudio-Worker-Instance") != "" || !r.Close {
			t.Error("readiness lost its selected host, TLS identity, or credential-free contract")
		}
		w.Header().Set("Content-Type", "application/json")
		if unavailable.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"up","service":"teststudio","role":"worker"}`))
	}))
	roots := x509.NewCertPool()
	roots.AddCert(ca.Certificate)
	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: backend.Hostname(), RootCAs: roots}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != net.JoinHostPort(backend.Hostname(), "443") {
			t.Error("owned readiness transport received a foreign authority")
		}
		dials.Add(1)
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
	}}
	defer transport.CloseIdleConnections()
	workerRelay, err := relay.NewPinned(s.Worker.ExchangeOrigin, b.Origin, b.Address, transport)
	if err != nil {
		t.Fatal(err)
	}
	defer workerRelay.Close()
	probe := func(path string, expected int) {
		t.Helper()
		response := httptest.NewRecorder()
		workerRelay.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != expected || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("unexpected relay health result", path, response.Code)
		}
	}
	probe("/readyz", http.StatusOK)
	unavailable.Store(true)
	probe("/readyz", http.StatusServiceUnavailable)
	probe("/livez", http.StatusOK)
	unavailable.Store(false)
	probe("/readyz", http.StatusOK)
	if dials.Load() != 3 {
		t.Fatal("readiness reused a TLS connection or liveness contacted the backend")
	}
}
