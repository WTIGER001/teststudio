package oidc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestProviderLogoutBindsBrowserStateAndNeverClearsANewerSession(t *testing.T) {
	s, _ := fixture(t)
	remote := newIssuer(t)
	register(t, s, remote, "one")
	row, _ := s.Store.Get(t.Context(), "providers", "one")
	p, _ := storage.Decode[authoidc.Provider](row)
	p.LogoutEndpoint = remote.server.URL + "/end-session"
	change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, p)
	if err := s.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	current, _ := signIn(t, s, remote, "one")
	if _, _, err := s.OfferLogout(t.Context(), current); err == nil {
		t.Fatal("provider logout bypassed local revocation")
	}
	if err := s.Sessions.Revoke(t.Context(), current.ID); err != nil {
		t.Fatal(err)
	}
	offer, cookie, err := s.OfferLogout(t.Context(), current)
	if err != nil {
		t.Fatal(err)
	}
	if offer == nil || cookie == nil || !cookie.HttpOnly || cookie.Path != "/auth" || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("logout offer lacks browser protection")
	}
	launch := httptest.NewRequest("GET", s.Origin+offer.URL, nil)
	if err := s.LaunchLogout(httptest.NewRecorder(), launch, strings.TrimPrefix(offer.URL, "/auth/logout/")); err == nil {
		t.Fatal("URL possession bypassed browser binding")
	}
	launch.AddCookie(cookie)
	w := httptest.NewRecorder()
	token := strings.TrimPrefix(offer.URL, "/auth/logout/")
	if err := s.LaunchLogout(w, launch, token); err != nil {
		t.Fatal(err)
	}
	destination, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	query := destination.Query()
	if destination.String() == "" || destination.Path != "/end-session" || query.Get("client_id") != "teststudio" || query.Get("state") == "" || query.Get("id_token_hint") != "" || query.Get("post_logout_redirect_uri") != s.Origin+"/auth/logout/callback/one" {
		t.Fatal("invalid provider logout request")
	}
	if err := s.LaunchLogout(httptest.NewRecorder(), launch, token); err == nil {
		t.Fatal("logout launch replay accepted")
	}
	callback := httptest.NewRequest("GET", query.Get("post_logout_redirect_uri")+"?state=wrong", nil)
	callback.AddCookie(cookie)
	if err := s.CompleteLogout(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("wrong logout state accepted")
	}
	newer, err := s.Sessions.Create(t.Context(), httptest.NewRecorder(), current.UserID, "local", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	callback.URL.RawQuery = "state=" + url.QueryEscape(query.Get("state"))
	w = httptest.NewRecorder()
	if err := s.CompleteLogout(w, callback, "one"); err != nil {
		t.Fatal(err)
	}
	if w.Header().Get("Location") != "/projects?providerLogout=complete" {
		t.Fatal("logout returned to an unconfigured destination")
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "teststudio_session" {
			t.Fatal("logout callback cleared newer session")
		}
	}
	if _, err := s.Sessions.Validate(t.Context(), newer.ID); err != nil {
		t.Fatal("newer session was revoked", err)
	}
	if _, err := s.Sessions.Validate(t.Context(), current.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("original session revived", err)
	}
	if err := s.CompleteLogout(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("logout callback replay accepted")
	}
}

func TestProviderLogoutFailureLeavesLocalSessionRevoked(t *testing.T) {
	for _, mode := range []string{"discovery-failure", "metadata-mismatch", "configuration-change", "expiry"} {
		t.Run(mode, func(t *testing.T) {
			s, _ := fixture(t)
			remote := newIssuer(t)
			register(t, s, remote, "one")
			row, _ := s.Store.Get(t.Context(), "providers", "one")
			p, _ := storage.Decode[authoidc.Provider](row)
			p.LogoutEndpoint = remote.server.URL + "/end-session"
			change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, p)
			if err := s.Store.Commit(t.Context(), change); err != nil {
				t.Fatal(err)
			}
			current, _ := signIn(t, s, remote, "one")
			if err := s.Sessions.Revoke(t.Context(), current.ID); err != nil {
				t.Fatal(err)
			}
			offer, cookie, err := s.OfferLogout(t.Context(), current)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "discovery-failure":
				remote.server.Close()
			case "metadata-mismatch":
				remote.metadata = func(data map[string]any) { data["end_session_endpoint"] = "https://different.example/end-session" }
			case "configuration-change":
				row, _ := s.Store.Get(t.Context(), "providers", "one")
				p.Name = "Changed provider"
				change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, p)
				if err := s.Store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
			case "expiry":
				rows, _ := s.Store.All(t.Context(), "logout-transactions", "")
				value, _ := storage.Decode[map[string]any](rows[0])
				value["expiresAt"] = time.Now().Add(-time.Second).Format(time.RFC3339Nano)
				change, _ := storage.Put(rows[0].Bucket, rows[0].ID, "", rows[0].Revision, value)
				if err := s.Store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
			}
			request := httptest.NewRequest("GET", s.Origin+offer.URL, nil)
			request.AddCookie(cookie)
			if err := s.LaunchLogout(httptest.NewRecorder(), request, strings.TrimPrefix(offer.URL, "/auth/logout/")); err == nil {
				t.Fatal("invalid provider logout launched")
			}
			if _, err := s.Sessions.Validate(t.Context(), current.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
				t.Fatal("failed provider logout restored local session", err)
			}
		})
	}
}
