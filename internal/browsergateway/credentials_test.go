package browsergateway

import (
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/networkguard"
)

func TestAPIGatewayInjectsCredentialsAfterPolicyAndScrubsReflections(t *testing.T) {
	for _, scheme := range []string{"bearer", "basic", "header"} {
		t.Run(scheme, func(t *testing.T) {
			raw := "bound-private-canary"
			header, value := "Authorization", "Bearer "+raw
			if scheme == "basic" {
				raw = "reader:bound-private-canary"
				value = "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
			}
			if scheme == "header" {
				header, value = "X-Api-Key", raw
			}
			var hits atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if r.Header.Get(header) != value {
					w.WriteHeader(401)
					return
				}
				w.Header().Set("X-Reflected", base64.StdEncoding.EncodeToString([]byte(raw)))
				_, _ = io.WriteString(w, `{"status":"authorized","echo":"`+raw+`","header":"`+value+`"}`)
			}))
			defer upstream.Close()
			address, _ := url.Parse(upstream.URL)
			auth := &apipolicy.Authentication{Binding: "reader", Scheme: scheme}
			if scheme == "header" {
				auth.Header = "X-API-Key"
			}
			target := browserpolicy.Target{Origin: "http://host.docker.internal:" + address.Port(), AllowPrivate: true, Authenticated: true, AllowCredentialHTTP: true, Requests: &apipolicy.Spec{Authentication: auth, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/approved", ReadOnly: true}}, MaxRequests: 3, MaxResponseBytes: 4096}}
			credential := &apicredential.Value{Origin: target.Origin, Header: header, Value: value, Network: networkguard.Private}
			if _, err := New(target); err == nil {
				t.Fatal("authenticated gateway starts without a credential")
			}
			g, err := NewAuthenticated(target, credential)
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			dial := g.dial
			g.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return dial(ctx, network, address.Host)
			}
			for _, path := range []string{"/outside", "https://foreign.test/approved"} {
				response := httptest.NewRecorder()
				g.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
				if response.Code != 403 {
					t.Fatal("credential reaches an unreviewed origin or path")
				}
			}
			if hits.Load() != 0 {
				t.Fatal("denied requests reach upstream")
			}
			request := httptest.NewRequest("GET", "/approved", nil)
			request.Header.Set(header, "spoofed")
			request.Header.Set("Connection", header)
			response := httptest.NewRecorder()
			g.ServeHTTP(response, request)
			if response.Code != 200 || !strings.Contains(response.Body.String(), "authorized") || strings.Contains(response.Body.String(), "canary") || strings.Contains(response.Header().Get("X-Reflected"), base64.StdEncoding.EncodeToString([]byte(raw))) {
				t.Fatal("credential injection or response scrubbing fails", response.Code)
			}
			public := *credential
			public.Network = networkguard.Public
			limited, err := NewAuthenticated(target, &public)
			if err != nil {
				t.Fatal(err)
			}
			defer limited.Close()
			limitedDial := limited.dial
			limited.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return limitedDial(ctx, network, address.Host)
			}
			blocked := httptest.NewRecorder()
			limited.ServeHTTP(blocked, httptest.NewRequest("GET", "/approved", nil))
			if blocked.Code != 502 || hits.Load() != 1 {
				t.Fatal("environment private access widens a public-only credential scope")
			}
			target.Authenticated, target.AllowCredentialHTTP = false, false
			negative, err := New(target)
			if err != nil {
				t.Fatal(err)
			}
			defer negative.Close()
			negative.transport.DialContext = g.transport.DialContext
			response = httptest.NewRecorder()
			negative.ServeHTTP(response, httptest.NewRequest("GET", "/approved", nil))
			if response.Code != 401 {
				t.Fatal("missing-credential scenario unexpectedly authenticates")
			}
		})
	}
}
