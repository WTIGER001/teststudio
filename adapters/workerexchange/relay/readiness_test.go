package relay

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerwire"
)

func readyRelay(t *testing.T, lifetime time.Duration, handler http.Handler) (*Relay, *http.Transport, *atomic.Int64) {
	t.Helper()
	ca := testkit.NewCertificateAuthority(t, "Owned relay readiness authority")
	server := testkit.NewTLSServer(t, ca, "backend.example.test", lifetime, handler)
	backend, _ := url.Parse("https://backend.example.test:4443")
	dials := &atomic.Int64{}
	transport := newRelayTransport(backend, "10.40.2.4", func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "10.40.2.4:4443" {
			t.Error("backend dial escaped its selected private address")
		}
		dials.Add(1)
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
	})
	transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	transport.TLSClientConfig.RootCAs.AddCert(ca.Certificate)
	relay, err := NewPinned("https://workers.example.test", backend.String(), "10.40.2.4", transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(relay.Close)
	return relay, transport, dials
}

func probe(t *testing.T, relay *Relay, path string, want int) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	relay.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != want || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("unexpected probe result", path, response.Code, response.Body.String())
	}
	return response
}

func TestWorkerRelayReadinessPinsBackendAndUsesFreshCredentialFreeTLS(t *testing.T) {
	relay, _, dials := readyRelay(t, 5*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "backend.example.test:4443" || r.TLS.ServerName != "backend.example.test" || r.URL.RawQuery != "" {
			t.Error("backend Host or TLS identity changed")
		}
		if r.URL.Path == "/healthz" {
			if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Origin") != "" || r.Header.Get("X-TestStudio-Worker-Instance") != "" || !r.Close {
				t.Error("readiness gained caller authority or retained its TLS connection")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"up","service":"teststudio","role":"worker"}`)
			return
		}
		if r.URL.Path != "/internal/worker/v1/executions/execution-1/input" || r.Header.Get("Authorization") == "" {
			t.Error("worker transport changed while adding readiness")
		}
		json.NewEncoder(w).Encode(workerwire.Packet{Execution: "execution-1", Deadline: time.Now().Add(time.Minute), Input: json.RawMessage(`{"fixture":true}`)})
	}))
	transport := relay.readinessClient.Transport.(*http.Transport)
	if transport.Protocols == nil || !transport.Protocols.HTTP1() || transport.Protocols.HTTP2() || !transport.DisableKeepAlives {
		t.Fatal("readiness transport can reuse or multiplex a TLS connection")
	}
	probe(t, relay, "/readyz", 200)
	probe(t, relay, "/readyz", 200)
	if dials.Load() != 2 {
		t.Fatal("readiness did not verify a fresh TLS connection")
	}
	response := httptest.NewRecorder()
	relay.ServeHTTP(response, request(t, "GET", "/internal/worker/v1/executions/execution-1/input", nil))
	if response.Code != 200 || dials.Load() != 3 {
		t.Fatal("pinned worker forwarding failed", response.Code)
	}
	probe(t, relay, "/livez", 200)
	probe(t, relay, "/healthz", 200)
	if dials.Load() != 3 {
		t.Fatal("local liveness triggered another backend connection")
	}
}

func TestWorkerRelayReadinessRejectsUntrustedResponsesAndCallerCredentials(t *testing.T) {
	var mode atomic.Int32
	relay, _, dials := readyRelay(t, 5*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch mode.Load() {
		case 1:
			w.WriteHeader(503)
			fmt.Fprint(w, "private-backend-canary")
		case 2:
			fmt.Fprint(w, `{"status":"up","service":"foreign","role":"worker"}`)
		case 3:
			fmt.Fprint(w, `{"status":"up","status":"up","service":"teststudio","role":"worker"}`)
		case 4:
			fmt.Fprint(w, `{"status":"up","service":"teststudio","role":"worker","private":"canary"}`)
		case 5:
			fmt.Fprint(w, strings.Repeat("x", 1025))
		case 7:
			fmt.Fprint(w, `{"status":"up","service":"teststudio"}`)
		case 6:
			w.Header().Set("Location", "https://foreign.invalid/private-canary")
			w.WriteHeader(302)
		default:
			fmt.Fprint(w, `{"status":"up","service":"teststudio","role":"worker"}`)
		}
	}))
	for _, header := range []string{"Authorization", "Cookie", "Origin", "X-TestStudio-Worker-Instance"} {
		request := httptest.NewRequest("GET", "/readyz", nil)
		request.Header.Set(header, "private-caller-canary")
		response := httptest.NewRecorder()
		relay.ServeHTTP(response, request)
		if response.Code != 404 || dials.Load() != 0 || strings.Contains(response.Body.String(), "canary") {
			t.Fatal("caller credentials reached readiness")
		}
	}
	for _, path := range []string{"/readyz?ticket=private-canary", "/readyz?"} {
		probe(t, relay, path, 404)
	}
	for value := int32(1); value <= 7; value++ {
		mode.Store(value)
		response := probe(t, relay, "/readyz", 503)
		if strings.Contains(response.Body.String(), "canary") || response.Header().Get("Location") != "" {
			t.Fatal("backend response details escaped readiness")
		}
	}
	if dials.Load() != 7 {
		t.Fatal("readiness retried or followed a redirect")
	}
	probe(t, relay, "/livez", 200)
	mode.Store(0)
	probe(t, relay, "/readyz", 200)
}

func TestWorkerRelayReadinessPreservesCertificateTrustAndLifetime(t *testing.T) {
	health := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"up","service":"teststudio","role":"worker"}`)
	})
	short, _, _ := readyRelay(t, time.Hour, health)
	probe(t, short, "/readyz", 503)
	trusted, _, _ := readyRelay(t, 5*time.Hour, health)
	transport := trusted.readinessClient.Transport.(*http.Transport)
	transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	probe(t, trusted, "/readyz", 503)
	insecure := transport.Clone()
	insecure.TLSClientConfig.InsecureSkipVerify = true
	trusted.readinessClient.Transport = insecure
	probe(t, trusted, "/readyz", 503)
}

func TestWorkerRelayReadinessBoundsConcurrencyAndPropagatesCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	relay, _, dials := readyRelay(t, 5*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	for range 8 {
		relay.slots <- struct{}{}
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		relay.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/readyz", nil).WithContext(ctx))
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("owned readiness call did not stop")
		}
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker saturation blocked the separate readiness connection")
	}
	probe(t, relay, "/readyz", 503)
	probe(t, relay, "/livez", 200)
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("cancellation did not reach the readiness client")
	}
	if dials.Load() != 1 {
		t.Fatal("saturated readiness started another backend connection")
	}
}

func TestWorkerRelayPinnedConfigurationRejectsForeignDestinations(t *testing.T) {
	for _, address := range []string{"", "8.8.8.8", "127.0.0.1", "::ffff:10.40.2.4", "10.40.2.4:443", "private-canary"} {
		_, err := NewPinned("https://workers.example.test", "https://backend.example.test", address, nil)
		if err == nil || strings.Contains(err.Error(), "canary") {
			t.Fatal("invalid backend pin accepted or disclosed", err)
		}
	}
	backend, _ := url.Parse("https://backend.example.test")
	count := 0
	transport := newRelayTransport(backend, "10.40.2.4", func(_ context.Context, _, address string) (net.Conn, error) {
		count++
		if address != "10.40.2.4:443" {
			t.Fatal("configured origin did not dial its pinned private address")
		}
		return nil, errors.New("owned fake dial")
	})
	if transport.Proxy != nil || transport.TLSClientConfig.ServerName != "backend.example.test" || transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("pinned production transport lost normal TLS verification")
	}
	if _, err := transport.DialContext(t.Context(), "tcp", "foreign.example.test:443"); err != errConfiguration || count != 0 {
		t.Fatal("foreign dial escaped its origin")
	}
	transport.DialContext(t.Context(), "tcp", "backend.example.test:443")
	if count != 1 {
		t.Fatal("selected private dial was not exercised")
	}
}
