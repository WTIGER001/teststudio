package oidc_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestOIDCSecretResolutionRequiresAnOperatorApprovedReference(t *testing.T) {
	s, db := fixture(t)
	issuer := newIssuer(t)
	register(t, s, issuer, "one")
	changeProvider(t, db, "one", func(provider *authoidc.Provider) { provider.SecretRef = "env:OIDC_APPROVED_SECRET" })
	var resolved atomic.Int32
	s.ResolveSecret = func(context.Context, string) (string, error) {
		resolved.Add(1)
		return "server-only-fixture-secret", nil
	}
	request := httptest.NewRequest("GET", s.Origin+"/auth/login/one", nil)
	if err := s.Login(httptest.NewRecorder(), request, "one"); err == nil {
		t.Fatal("unapproved reference started provider sign-in")
	}
	if resolved.Load() != 0 {
		t.Fatal("unapproved reference reached credential resolution")
	}
	s.SecretRefs = []string{"env:OIDC_APPROVED_SECRET"}
	response := httptest.NewRecorder()
	if err := s.Login(response, request, "one"); err != nil {
		t.Fatal(err)
	}
	if resolved.Load() != 1 || response.Code != 302 {
		t.Fatal("approved reference did not start sign-in")
	}
}

func TestOIDCDifferentSubjectsAtTheSameIssuerDoNotTransferAccessForEqualEmails(t *testing.T) {
	s, db := fixture(t)
	issuer := newIssuer(t)
	register(t, s, issuer, "one")
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), s.Directory)
	if err != nil {
		t.Fatal(err)
	}
	users := []string{}
	for _, subject := range []string{"subject-one", "subject-two"} {
		issuer.subject = subject
		callback, err := login(t, s, issuer, "one")
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		if err := s.Callback(response, callback, "one"); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("GET", s.Origin+"/auth/session", nil)
		for _, cookie := range response.Result().Cookies() {
			if cookie.Name == "teststudio_session" {
				request.AddCookie(cookie)
			}
		}
		session, err := s.Sessions.Read(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, session.UserID)
		if subject == "subject-one" {
			if err := auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: session.UserID, TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if users[0] == users[1] {
		t.Fatal("verified equal email merged different subjects")
	}
	if err := auth.Check(t.Context(), users[0], "alpha", "suite/health", "catalog.read"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), users[1], "alpha", "suite/health", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("different subject acquired another account's permissions", err)
	}
	first, err := s.Directory.User(t.Context(), users[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Directory.User(t.Context(), users[1])
	if err != nil {
		t.Fatal(err)
	}
	if first.Email != second.Email || !first.EmailVerified || !second.EmailVerified {
		t.Fatal("fixture did not provide equal verified emails")
	}
}

func TestHiddenProviderKeepsExplicitRecoverySignIn(t *testing.T) {
	service, store := fixture(t)
	issuer := newIssuer(t)
	register(t, service, issuer, "recovery")
	changeProvider(t, store, "recovery", func(provider *authoidc.Provider) { provider.HiddenFromSignIn = true })
	request := httptest.NewRequest("GET", service.Origin+"/auth/login/recovery?returnTo=/account", nil)
	response := httptest.NewRecorder()
	if err := service.Login(response, request, "recovery"); err != nil {
		t.Fatal(err)
	}
	if response.Code != 302 {
		t.Fatal("explicit recovery sign-in failed")
	}
}
