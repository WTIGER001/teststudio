package oidc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/accountlinks"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type projectAccessObserver struct {
	calls int
	last  authoidc.SignInIdentity
}

func (o *projectAccessObserver) ApplyPendingAccess(_ context.Context, proof authoidc.SignInIdentity) error {
	o.calls++
	o.last = proof
	return nil
}

func signIn(t *testing.T, s *authoidc.Service, i *issuer, id string) (sessions.Session, *http.Cookie) {
	t.Helper()
	callback, err := login(t, s, i, id)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	if err := s.Callback(w, callback, id); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "teststudio_session" {
			session, err := s.Sessions.Validate(t.Context(), sessions.Digest(cookie.Value))
			if err != nil {
				t.Fatal(err)
			}
			return session, cookie
		}
	}
	t.Fatal("session cookie missing")
	return sessions.Session{}, nil
}

func authorizeLink(t *testing.T, s *authoidc.Service, i *issuer, id string, cookie *http.Cookie) *http.Request {
	t.Helper()
	r := httptest.NewRequest("GET", s.Origin+"/auth/identity-links/"+id+"/authorize", nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	if err := s.AuthorizeIdentityLink(w, r, id); err != nil {
		t.Fatal(err)
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Query().Get("max_age") != "0" || location.Query().Get("prompt") != "login" || location.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("fresh authentication or PKCE not requested", location)
	}
	i.nonce, i.challenge = location.Query().Get("nonce"), location.Query().Get("code_challenge")
	callback := httptest.NewRequest("GET", location.Query().Get("redirect_uri")+"?code=code&state="+url.QueryEscape(location.Query().Get("state")), nil)
	callback.AddCookie(cookie)
	callback.AddCookie(w.Result().Cookies()[0])
	return callback
}

func TestOIDCLinkProvesCurrentAndTargetBeforeExplicitConfirmation(t *testing.T) {
	s, db := fixture(t)
	source, target := newIssuer(t), newIssuer(t)
	register(t, s, source, "source")
	register(t, s, target, "target")
	s.Links = &accountlinks.Service{Store: db, Sessions: s.Sessions}
	observer := &projectAccessObserver{}
	s.ProjectAccess = observer
	current, cookie := signIn(t, s, source, "source")
	op, err := s.Links.Start(t.Context(), current.ID, accountlinks.StartInput{ProviderID: "target", Reason: "Use another provider"}, "127.0.0.1")
	if err != nil || op.State != "authenticate-source" {
		t.Fatal("source proof missing", err)
	}
	if _, _, err := s.Links.Confirm(t.Context(), current.ID, op.ID, op.Revision); err == nil {
		t.Fatal("unproved link confirmed")
	}
	for _, step := range []struct {
		id    string
		i     *issuer
		state string
	}{{"source", source, "prove-target"}, {"target", target, "review"}} {
		callback := authorizeLink(t, s, step.i, op.ID, cookie)
		w := httptest.NewRecorder()
		if err := s.Callback(w, callback, step.id); err != nil {
			t.Fatal(err)
		}
		if w.Code != 302 || w.Header().Get("Location") != "/account?identityLink="+op.ID {
			t.Fatal("callback did not return to review")
		}
		for _, c := range w.Result().Cookies() {
			if c.Name == "teststudio_session" {
				t.Fatal("proof replaced current session")
			}
		}
		op, err = s.Links.Read(t.Context(), current.ID, op.ID)
		if err != nil || op.State != step.state {
			t.Fatal("wrong proof state", op.State, err)
		}
		if err := s.Callback(httptest.NewRecorder(), callback, step.id); err == nil {
			t.Fatal("proof callback replay accepted")
		}
	}
	if observer.calls != 1 {
		t.Fatal("linking proof applied pending project access", observer.calls)
	}
	users, err := db.All(t.Context(), "users", "")
	if err != nil || len(users) != 1 {
		t.Fatal("proof provisioned a second user", len(users), err)
	}
	if _, err := s.Directory.ExternalLink(t.Context(), target.server.URL, target.subject); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("target linked before confirmation", err)
	}
	completed, newCookie, err := s.Links.Confirm(t.Context(), current.ID, op.ID, op.Revision)
	if err != nil || completed.State != "complete" || newCookie == nil {
		t.Fatal("confirm failed", err)
	}
	next, err := s.Sessions.Validate(t.Context(), sessions.Digest(newCookie.Value))
	if err != nil || next.UserID != current.UserID || next.Provider != "source" {
		t.Fatal("confirmation changed user or active provider", err)
	}
	targetSession, _ := signIn(t, s, target, "target")
	if observer.calls != 2 || observer.last.UserID != current.UserID || observer.last.ProviderID != "target" || !observer.last.EmailVerified || observer.last.Subject != target.subject {
		t.Fatal("normal sign-in did not deliver verified pending access context", observer.calls)
	}
	if targetSession.UserID != current.UserID {
		t.Fatal("linked provider did not sign in to the same account")
	}
}

func TestOIDCLinkRejectsStaleMissingOrInvalidFreshAuthentication(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing", func(c map[string]any) { delete(c, "auth_time") }},
		{"stale", func(c map[string]any) { c["auth_time"] = time.Now().Add(-10 * time.Minute).Unix() }},
		{"future", func(c map[string]any) { c["auth_time"] = time.Now().Add(5 * time.Minute).Unix() }},
		{"string", func(c map[string]any) { c["auth_time"] = "123" }},
		{"wrong-subject", func(c map[string]any) { c["sub"] = "another-user" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, db := fixture(t)
			source, target := newIssuer(t), newIssuer(t)
			register(t, s, source, "source")
			register(t, s, target, "target")
			s.Links = &accountlinks.Service{Store: db, Sessions: s.Sessions}
			current, cookie := signIn(t, s, source, "source")
			op, err := s.Links.Start(t.Context(), current.ID, accountlinks.StartInput{ProviderID: "target", Reason: "Add sign-in"}, "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			callback := authorizeLink(t, s, source, op.ID, cookie)
			source.mutate = test.mutate
			w := httptest.NewRecorder()
			if err := s.Callback(w, callback, "source"); err != nil {
				t.Fatal("valid bound failure should return to account", err)
			}
			op, err = s.Links.Read(t.Context(), current.ID, op.ID)
			if err != nil || op.State != "failed" || op.Identity != nil {
				t.Fatal("invalid fresh proof accepted", op, err)
			}
			users, _ := db.All(t.Context(), "users", "")
			if len(users) != 1 {
				t.Fatal("source proof provisioned a different account")
			}
		})
	}
}

func TestOIDCLinkCallbackCannotMoveAcrossSessions(t *testing.T) {
	s, db := fixture(t)
	target := newIssuer(t)
	register(t, s, target, "target")
	user, err := s.Directory.ResolveExternal(t.Context(), "local-fixture", "owner", "Owner", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	current, err := s.Sessions.Create(t.Context(), w, user.ID, "local", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	cookie := w.Result().Cookies()[0]
	s.Links = &accountlinks.Service{Store: db, Sessions: s.Sessions, LocalUser: user.ID}
	op, err := s.Links.Start(t.Context(), current.ID, accountlinks.StartInput{ProviderID: "target", Reason: "Link identity", ConfirmLocal: true}, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	callback := authorizeLink(t, s, target, op.ID, cookie)
	otherW := httptest.NewRecorder()
	if _, err := s.Sessions.Create(t.Context(), otherW, user.ID, "local", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	savedLogin, _ := callback.Cookie("teststudio_login")
	callback.Header.Del("Cookie")
	callback.AddCookie(savedLogin)
	callback.AddCookie(otherW.Result().Cookies()[0])
	if err := s.Callback(httptest.NewRecorder(), callback, "target"); err == nil {
		t.Fatal("callback accepted a different session")
	}
	if target.tokens.Load() != 0 {
		t.Fatal("cross-session callback exchanged code")
	}
	op, err = s.Links.Read(t.Context(), current.ID, op.ID)
	if err != nil || op.State != "prove-target" {
		t.Fatal("cross-session callback changed original operation", err)
	}
	if _, err := s.Directory.ExternalLink(t.Context(), target.server.URL, target.subject); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("cross-session callback linked identity", err)
	}
}
