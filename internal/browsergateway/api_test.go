package browsergateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestAPIGatewayChecksMethodsPathsPayloadsRedirectsAndTLSBeforeTransmission(t *testing.T) {
	ca := testkit.NewCertificateAuthority(t, "Owned API gateway authority")
	var hits atomic.Int32
	upstream := testkit.NewTLSServer(t, ca, "fixture.test", time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Proxy-Authorization") != "" {
			t.Error("proxy credentials reach the API")
		}
		switch r.URL.Path {
		case "/v1/compressed":
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = io.WriteString(w, "encoded-fixture")
		case "/v1/redirect":
			http.Redirect(w, r, "https://unapproved.invalid/private-canary", 302)
		case "/v1/large":
			_, _ = io.WriteString(w, strings.Repeat("x", 1025))
		default:
			if r.Method == "HEAD" {
				w.Header().Set("Content-Length", "123")
				w.WriteHeader(200)
				return
			}
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write([]byte(r.Method + " " + r.URL.RequestURI() + " " + string(body)))
		}
	}))
	actual, _ := url.Parse(upstream.URL)
	selected := "https://fixture.test:" + actual.Port()
	spec := &apipolicy.Spec{Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/v1", ReadOnly: true}, {Method: "HEAD", PathPrefix: "/v1", ReadOnly: true}, {Method: "POST", PathPrefix: "/v1/search", ReadOnly: true, MaxBodyBytes: 8}}, MaxRequests: 10, MaxResponseBytes: 1024}
	g, err := New(browserpolicy.Target{Origin: selected, AllowPrivate: true, Requests: spec})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(g.Close)
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	g.transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	original := g.dial
	g.transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "fixture.test:"+actual.Port() {
			t.Error("unexpected API destination")
		}
		return original(ctx, network, actual.Host)
	}
	for _, scenario := range []struct {
		method, target, body string
		code                 int
	}{
		{"GET", "/v1/items/7?limit=5", "", 200},
		{"HEAD", "/v1/items/7", "", 200},
		{"POST", "/v1/search", "filter=7", 200},
		{"POST", "/v1/search", "filter=77", 413},
		{"DELETE", "/v1/items/7", "", 403},
		{"GET", "/v10/private", "", 403},
		{"GET", "/v1/../private", "", 403},
		{"GET", "/v1/%2fprivate", "", 403},
		{"GET", "https://unapproved.invalid/v1/items", "", 403},
		{"CONNECT", "https://fixture.test:" + actual.Port(), "", 403},
		{"GET", "/v1/redirect", "", 302},
		{"GET", "/v1/large", "", 502},
		{"GET", "/v1/compressed", "", 502},
	} {
		request := httptest.NewRequest(scenario.method, scenario.target, strings.NewReader(scenario.body))
		request.Header.Set("Proxy-Authorization", "owned-private-proxy-canary")
		response := httptest.NewRecorder()
		g.ServeHTTP(response, request)
		if response.Code != scenario.code || response.Header().Get("Location") != "" || strings.Contains(response.Body.String(), "private-canary") {
			t.Fatal("API policy decision differs", scenario.method, scenario.target, response.Code, response.Body.String())
		}
		if scenario.method == "HEAD" && response.Header().Get("Content-Length") != "123" {
			t.Fatal("HEAD loses its upstream content length")
		}
	}
	if hits.Load() != 6 {
		t.Fatal("rejected request reaches the target", hits.Load())
	}
	request := httptest.NewRequest("GET", "/v1/items", nil)
	request.Header.Set("X-HTTP-Method-Override", "DELETE")
	response := httptest.NewRecorder()
	g.ServeHTTP(response, request)
	if response.Code != 403 || hits.Load() != 6 {
		t.Fatal("method override bypasses read-only review")
	}
	request = httptest.NewRequest("POST", "/v1/search", strings.NewReader("data"))
	request.Header.Set("Content-Encoding", "gzip")
	response = httptest.NewRecorder()
	g.ServeHTTP(response, request)
	if response.Code != 415 || hits.Load() != 6 {
		t.Fatal("compressed request bypasses payload admission")
	}
	g.requests.MaxRequests = int(g.count.Load())
	response = httptest.NewRecorder()
	g.ServeHTTP(response, httptest.NewRequest("GET", "/v1/items", nil))
	if response.Code != 429 || hits.Load() != 6 {
		t.Fatal("attempt request budget is not enforced before dispatch")
	}
	g.requests.MaxRequests += 10
	g.transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	response = httptest.NewRecorder()
	g.ServeHTTP(response, httptest.NewRequest("GET", "/v1/items", nil))
	if response.Code != 502 || hits.Load() != 6 {
		t.Fatal("API gateway skips TLS trust validation")
	}
}
