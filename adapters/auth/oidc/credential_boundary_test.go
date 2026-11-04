package oidc_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
)

type credentialTransport func(*http.Request) (*http.Response, error)

func (f credentialTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOIDCProviderAndResolverErrorsExcludePrivateText(t *testing.T) {
	for _, mode := range []string{"discovery-body", "discovery-transport", "resolver-error", "resolver-empty"} {
		t.Run(mode, func(t *testing.T) {
			s, db := fixture(t)
			issuer := newIssuer(t)
			register(t, s, issuer, "one")
			const private = "teststudio-owned-private-error-canary"
			switch mode {
			case "discovery-body", "discovery-transport":
				s.Client = &http.Client{Transport: credentialTransport(func(r *http.Request) (*http.Response, error) {
					if mode == "discovery-transport" {
						return nil, errors.New(private)
					}
					return &http.Response{StatusCode: 502, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(private)), Request: r}, nil
				})}
			default:
				changeProvider(t, db, "one", func(p *authoidc.Provider) { p.SecretRef = "env:OIDC_BOUNDARY_SECRET" })
				s.SecretRefs = []string{"env:OIDC_BOUNDARY_SECRET"}
				s.ResolveSecret = func(context.Context, string) (string, error) {
					if mode == "resolver-error" {
						return private, fmt.Errorf("credential lookup failed: %s", private)
					}
					return "", nil
				}
			}
			response := httptest.NewRecorder()
			err := s.Login(response, httptest.NewRequest("GET", s.Origin+"/auth/login/one", nil), "one")
			if err == nil || strings.Contains(fmt.Sprintf("%+v", err), private) || strings.Contains(fmt.Sprintf("%#v", err), private) {
				t.Fatal("provider failure is missing or exposes private upstream text")
			}
			if response.Header().Get("Location") != "" || len(response.Result().Cookies()) != 0 {
				t.Fatal("failed provider setup creates a browser sign-in")
			}
			rows, err := db.All(t.Context(), "login-transactions", "")
			if err != nil || len(rows) != 0 {
				t.Fatal("failed provider setup retains a login transaction", err)
			}
		})
	}
}

func TestOIDCClientCredentialStaysInServerTokenExchange(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejected=%t", rejected), func(t *testing.T) {
			s, db := fixture(t)
			issuer := newIssuer(t)
			register(t, s, issuer, "one")
			changeProvider(t, db, "one", func(p *authoidc.Provider) { p.SecretRef = "env:OIDC_BOUNDARY_SECRET" })
			s.SecretRefs = []string{"env:OIDC_BOUNDARY_SECRET"}
			const secret = "teststudio-owned-token-exchange-canary"
			s.ResolveSecret = func(context.Context, string) (string, error) { return secret, nil }
			var exchanged bool
			s.Client = &http.Client{Transport: credentialTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/token" {
					client, password, ok := r.BasicAuth()
					if !ok && r.GetBody != nil {
						body, err := r.GetBody()
						if err != nil {
							return nil, err
						}
						data, err := io.ReadAll(body)
						body.Close()
						if err != nil {
							return nil, err
						}
						values, err := url.ParseQuery(string(data))
						if err != nil {
							return nil, errors.New("token request encoding is invalid")
						}
						client, password, ok = values.Get("client_id"), values.Get("client_secret"), true
					}
					if !ok || client != "teststudio" || password != secret || strings.Contains(r.URL.String(), secret) {
						t.Error("credential does not use the server-only token exchange")
					}
					exchanged = true
					if rejected {
						return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_client","error_description":"` + secret + `"}`)), Request: r}, nil
					}
				} else if strings.Contains(r.URL.String(), secret) || strings.Contains(r.Header.Get("Authorization"), secret) {
					t.Error("credential enters a request outside token exchange")
				}
				return http.DefaultTransport.RoundTrip(r)
			})}
			callback, err := login(t, s, issuer, "one")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(callback.URL.String(), secret) || strings.Contains(fmt.Sprint(callback.Cookies()), secret) {
				t.Fatal("credential enters browser callback or cookie")
			}
			response := httptest.NewRecorder()
			err = s.Callback(response, callback, "one")
			if rejected {
				if err == nil || strings.Contains(fmt.Sprintf("%+v", err), secret) {
					t.Fatal("rejected token exchange exposes private diagnostics or succeeds")
				}
				for _, cookie := range response.Result().Cookies() {
					if cookie.Name == "teststudio_session" {
						t.Fatal("rejected token exchange issues an authenticated session")
					}
				}
			} else if err != nil || response.Code != 302 {
				t.Fatal("authenticated server token exchange does not complete", err)
			}
			if !exchanged {
				t.Fatal("server token exchange is missing")
			}
			if strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), secret) {
				t.Fatal("credential enters browser response")
			}
			rows, err := db.All(t.Context(), "login-transactions", "")
			if err != nil || len(rows) != 0 {
				t.Fatal("completed exchange retains login transaction", err)
			}
			for _, bucket := range []string{"providers", "sessions", "audit", "users", "external-identities"} {
				rows, err := db.All(t.Context(), bucket, "")
				if err != nil {
					t.Fatal(err)
				}
				encoded, err := json.Marshal(rows)
				if err != nil || strings.Contains(string(encoded), secret) {
					t.Fatal("resolved credential enters stored application metadata", bucket, err)
				}
			}
		})
	}
}
