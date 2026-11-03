package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/sessions"
)

func TestOIDCLogoutClearsLocalSessionBeforeProviderFailure(t *testing.T) {
	c := fixture(t)
	c.signIn()
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("logout preparation contacted the provider")
		w.WriteHeader(503)
	}))
	input := providerInput("logout-provider", false)
	input.Issuer = remote.URL
	input.LogoutEndpoint = remote.URL + "/end-session"
	accessCall(t, c, "POST", "/api/v1/identity-providers", input, "", 200)
	writer := httptest.NewRecorder()
	current, err := c.api.Sessions.CreateOIDC(t.Context(), writer, "admin", input.ID, 1, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := url.Parse(c.origin)
	oidc := passwordClient(t, c)
	oidc.http.Jar.SetCookies(origin, writer.Result().Cookies())
	oidc.csrf = current.CSRF
	for _, headers := range []map[string]string{{"Origin": "https://other.example"}, {"X-CSRF-Token": "wrong"}} {
		if status, _ := oidc.call("POST", "/auth/logout", map[string]any{}, headers); status != 403 {
			t.Fatal("logout forgery accepted", status)
		}
	}
	remote.Close()
	response := accessCall(t, oidc, "POST", "/auth/logout", map[string]any{}, "", 200)
	offer := response["providerLogout"].(map[string]any)
	if response["signedOut"] != true || !strings.HasPrefix(offer["url"].(string), "/auth/logout/") {
		t.Fatal("optional logout offer missing", response)
	}
	if _, err := c.api.Sessions.Validate(t.Context(), current.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("provider offer retained local session", err)
	}
	if session := accessCall(t, oidc, "GET", "/auth/session", nil, "", 200); session["user"] != nil {
		t.Fatal("local session cookie remains active")
	}
	oidc.http.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	result, err := oidc.http.Get(c.origin + offer["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	result.Body.Close()
	if result.StatusCode != 302 || result.Header.Get("Location") != "/projects?providerLogout=unconfirmed" {
		t.Fatal("provider outage did not return a readable local outcome", result.StatusCode, result.Header.Get("Location"))
	}
	if session := accessCall(t, oidc, "GET", "/auth/session", nil, "", 200); session["user"] != nil {
		t.Fatal("provider failure restored local session")
	}
	if session := accessCall(t, c, "GET", "/auth/session", nil, "", 200); session["user"] == nil {
		t.Fatal("current-session logout revoked another browser")
	}
	encoded, _ := json.Marshal(response)
	for _, secret := range []string{current.ID, current.CSRF, "id_token", "access_token", "refresh_token", "clientSecret"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("logout response contains private credentials", secret)
		}
	}
}

func TestOIDCCallbackErrorsReturnReadableSignInOutcomes(t *testing.T) {
	c := fixture(t)
	c.http.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	for _, path := range []string{"/auth/callback/missing?code=untrusted&state=unknown", "/auth/login/missing"} {
		response, err := c.http.Get(c.origin + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		expected := "/projects?signIn=expired"
		if strings.HasPrefix(path, "/auth/login/") {
			expected = "/projects?signIn=unavailable"
		}
		if response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("authentication redirect can be cached")
		}
		if response.StatusCode != 302 || response.Header.Get("Location") != expected {
			t.Fatal("authentication error returned raw provider diagnostics", response.StatusCode, response.Header.Get("Location"))
		}
	}
	if status, _ := c.call("GET", "/api/v1/projects", nil, map[string]string{"Authorization": "Bearer provider-id-token"}); status != 401 {
		t.Fatal("provider token accepted as API session", status)
	}
}

func TestCancelledOIDCSignInPreservesItsApprovedReturnRoute(t *testing.T) {
	c := fixture(t)
	c.signIn()
	var remote *httptest.Server
	remote = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"issuer": remote.URL, "authorization_endpoint": remote.URL + "/authorize", "token_endpoint": remote.URL + "/token", "jwks_uri": remote.URL + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	}))
	defer remote.Close()
	input := providerInput("return-provider", false)
	input.Issuer = remote.URL
	accessCall(t, c, "POST", "/api/v1/identity-providers", input, "", 200)
	c.http.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	destination := "/projects/alpha/runs/one?test=read#evidence"
	response, err := c.http.Get(c.origin + "/auth/login/return-provider?returnTo=" + url.QueryEscape(destination))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	authorize, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	callback := c.origin + "/auth/callback/return-provider?error=access_denied&state=" + url.QueryEscape(authorize.Query().Get("state"))
	response, err = c.http.Get(callback)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	returned, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if returned.Path != "/projects/alpha/runs/one" || returned.Query().Get("test") != "read" || returned.Query().Get("signIn") != "cancelled" || returned.Fragment != "evidence" {
		t.Fatal("cancelled sign-in lost its local destination", returned)
	}
}
