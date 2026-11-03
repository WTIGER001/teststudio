package oidc_test

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func changeProvider(t *testing.T, db *storage.Store, id string, edit func(*authoidc.Provider)) int64 {
	t.Helper()
	row, err := db.Get(t.Context(), "providers", id)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := storage.Decode[authoidc.Provider](row)
	if err != nil {
		t.Fatal(err)
	}
	edit(&provider)
	change, err := storage.Put("providers", id, "", row.Revision, provider)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	return row.Revision + 1
}

func TestOIDCConfigurationChangesRejectPendingCallbacksBeforeExchange(t *testing.T) {
	for _, field := range []string{"issuer", "client", "disabled", "mapping", "display"} {
		t.Run(field, func(t *testing.T) {
			s, db := fixture(t)
			issuer := newIssuer(t)
			register(t, s, issuer, "one")
			callback, err := login(t, s, issuer, "one")
			if err != nil {
				t.Fatal(err)
			}
			changeProvider(t, db, "one", func(provider *authoidc.Provider) {
				switch field {
				case "issuer":
					provider.Issuer = "https://replacement.example.invalid"
				case "client":
					provider.ClientID = "replacement-client"
				case "disabled":
					provider.Disabled = true
				case "mapping":
					provider.GroupClaim = "groups"
					provider.GroupMappings = map[string]string{"operators": "operators"}
				case "display":
					provider.Name = "Updated display name"
				}
			})
			w := httptest.NewRecorder()
			if err := s.Callback(w, callback, "one"); err == nil {
				t.Fatal("changed provider accepted the earlier callback")
			}
			if issuer.tokens.Load() != 0 {
				t.Fatal("authorization code was sent after provider configuration changed")
			}
			rows, err := db.All(t.Context(), "users", "")
			if err != nil || len(rows) != 0 {
				t.Fatal("rejected callback provisioned an account", err)
			}
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == "teststudio_session" {
					t.Fatal("rejected callback issued a session")
				}
			}
			if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
				t.Fatal("rejected transaction could be reused")
			}
		})
	}
}

func TestOIDCConfigurationChangeDuringExchangeRejectsProvisioning(t *testing.T) {
	testOIDCConfigurationChangeDuringRemoteRequest(t, false)
}

func TestOIDCConfigurationChangeDuringDiscoveryRejectsExchange(t *testing.T) {
	testOIDCConfigurationChangeDuringRemoteRequest(t, true)
}

func testOIDCConfigurationChangeDuringRemoteRequest(t *testing.T, discovery bool) {
	s, db := fixture(t)
	issuer := newIssuer(t)
	register(t, s, issuer, "one")
	callback, err := login(t, s, issuer, "one")
	if err != nil {
		t.Fatal(err)
	}
	changeProvider := func() {
		row, err := db.Get(t.Context(), "providers", "one")
		if err != nil {
			t.Error(err)
			return
		}
		provider, err := storage.Decode[authoidc.Provider](row)
		if err != nil {
			t.Error(err)
			return
		}
		provider.Disabled = true
		change, err := storage.Put("providers", "one", "", row.Revision, provider)
		if err != nil {
			t.Error(err)
			return
		}
		if err := db.Commit(t.Context(), change); err != nil {
			t.Error(err)
		}
	}
	if discovery {
		issuer.onDiscovery = changeProvider
	} else {
		issuer.onToken = changeProvider
	}
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("configuration changed during exchange but sign-in succeeded")
	}
	if discovery && issuer.tokens.Load() != 0 {
		t.Fatal("authorization code was sent after provider changed during discovery")
	}
	for _, bucket := range []string{"users", "external-identities", "sessions", "memberships"} {
		rows, err := db.All(t.Context(), bucket, "")
		if err != nil || len(rows) != 0 {
			t.Fatal("changed callback wrote identity state", bucket, err)
		}
	}
}

func TestOIDCProviderRevisionsWithdrawSessionsAndGroupAccessUntilFreshSignIn(t *testing.T) {
	s, db := fixture(t)
	issuer := newIssuer(t)
	register(t, s, issuer, "one")
	if err := s.Directory.SaveGroup(t.Context(), identity.Group{ID: "operators", Name: "Operators", ManagedBy: "oidc:one"}, 0); err != nil {
		t.Fatal(err)
	}
	revision := changeProvider(t, db, "one", func(provider *authoidc.Provider) {
		provider.GroupClaim = "groups"
		provider.GroupMappings = map[string]string{"operators": "operators"}
	})
	callback, err := login(t, s, issuer, "one")
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	if err := s.Callback(w, callback, "one"); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", s.Origin+"/auth/session", nil)
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "teststudio_session" {
			request.AddCookie(cookie)
		}
	}
	session, err := s.Sessions.Read(t.Context(), request)
	if err != nil || session.AuthenticationMethod != "oidc" || session.ProviderRevision != revision {
		t.Fatal("session is not bound to its provider", session.AuthenticationMethod, err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), s.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: "operators"}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), session.UserID, "alpha", "suite/health", "catalog.read"); err != nil {
		t.Fatal(err)
	}
	nextRevision := changeProvider(t, db, "one", func(provider *authoidc.Provider) { provider.Name = "Changed provider" })
	if _, err := s.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("earlier browser session survived provider change", err)
	}
	if _, err := s.Sessions.Validate(t.Context(), session.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("earlier evidence session survived provider change", err)
	}
	if err := auth.Check(t.Context(), session.UserID, "alpha", "suite/health", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("stale provider membership still grants access", err)
	}
	if members, err := s.Directory.GetGroupMembers(t.Context(), "operators"); err != nil || len(members) != 0 {
		t.Fatal("stale group enumeration remained active", members, err)
	}
	before, err := s.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Directory.SyncProviderGroups(t.Context(), session.UserID, "one", revision, []string{"operators"}, time.Now().Add(time.Hour)); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("earlier callback restored stale membership", err)
	}
	after, err := s.Directory.Revision(t.Context())
	if err != nil || before.Revision != after.Revision {
		t.Fatal("rejected synchronization changed directory", err)
	}
	if _, err := s.Sessions.CreateOIDC(t.Context(), httptest.NewRecorder(), session.UserID, "one", revision, time.Now().Add(time.Hour)); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("earlier callback created a new session", err)
	}
	callback, err = login(t, s, issuer, "one")
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	if err := s.Callback(w, callback, "one"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), session.UserID, "alpha", "suite/health", "catalog.read"); err != nil {
		t.Fatal("fresh validated membership did not restore access", err)
	}
	if _, err := s.Sessions.Validate(t.Context(), session.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("fresh sign-in revived an older cookie", err)
	}
	row, err := db.Get(t.Context(), "memberships", identity.MembershipID(session.UserID, "operators", "oidc:one"))
	if err != nil {
		t.Fatal(err)
	}
	membership, err := storage.Decode[identity.Membership](row)
	if err != nil || membership.ProviderRevision != nextRevision {
		t.Fatal("membership provenance was not refreshed", err)
	}
	freshRequest := httptest.NewRequest("GET", s.Origin+"/auth/session", nil)
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "teststudio_session" {
			freshRequest.AddCookie(cookie)
		}
	}
	fresh, err := s.Sessions.Read(t.Context(), freshRequest)
	if err != nil {
		t.Fatal(err)
	}
	changeProvider(t, db, "one", func(provider *authoidc.Provider) { provider.Disabled = true })
	if _, err := s.Sessions.Validate(t.Context(), fresh.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("disabled provider retained an evidence session", err)
	}
	if groups, err := s.Directory.GetUserGroups(t.Context(), session.UserID); err != nil || len(groups) != 0 {
		t.Fatal("disabled provider retained group membership", groups, err)
	}
	changeProvider(t, db, "one", func(provider *authoidc.Provider) { provider.Disabled = false })
	if _, err := s.Sessions.Read(t.Context(), freshRequest); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("provider restoration revived an old cookie", err)
	}
	if groups, err := s.Directory.GetUserGroups(t.Context(), session.UserID); err != nil || len(groups) != 0 {
		t.Fatal("provider restoration revived stale membership", groups, err)
	}
}

func TestOIDCProviderNamesDoNotBypassSessionBindingAndLegacySessionsRequireSignIn(t *testing.T) {
	for _, id := range []string{"local", "password"} {
		t.Run(id, func(t *testing.T) {
			s, db := fixture(t)
			issuer := newIssuer(t)
			register(t, s, issuer, id)
			callback, err := login(t, s, issuer, id)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			if err := s.Callback(w, callback, id); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", s.Origin+"/auth/session", nil)
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == "teststudio_session" {
					request.AddCookie(cookie)
				}
			}
			session, err := s.Sessions.Read(t.Context(), request)
			if err != nil || session.AuthenticationMethod != "oidc" || session.ProviderRevision != 1 {
				t.Fatal("provider name selected another authentication method", err)
			}
			row, err := db.Get(t.Context(), "sessions", session.ID)
			if err != nil {
				t.Fatal(err)
			}
			session.AuthenticationMethod = ""
			session.ProviderRevision = 0
			change, err := storage.Put("sessions", row.ID, "", row.Revision, session)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Commit(t.Context(), change); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
				t.Fatal("ambiguous legacy browser session remained active", err)
			}
			if _, err := s.Sessions.Validate(t.Context(), session.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
				t.Fatal("ambiguous legacy evidence session remained active", err)
			}
		})
	}
}

func TestLegacyProviderMembershipRequiresValidatedRefreshAndPreservesLocalMembership(t *testing.T) {
	s, db := fixture(t)
	issuer := newIssuer(t)
	register(t, s, issuer, "one")
	if err := s.Directory.SaveUser(t.Context(), identity.User{ID: "member", Name: "Member"}, 0); err != nil {
		t.Fatal(err)
	}
	for _, group := range []identity.Group{{ID: "operators", Name: "Operators", ManagedBy: "oidc:one"}, {ID: "local", Name: "Local"}} {
		if err := s.Directory.SaveGroup(t.Context(), group, 0); err != nil {
			t.Fatal(err)
		}
	}
	expires := time.Now().Add(time.Hour)
	for _, membership := range []identity.Membership{{UserID: "member", GroupID: "operators", Source: "oidc:one", ExpiresAt: &expires}, {UserID: "member", GroupID: "local", Source: "local"}} {
		if err := s.Directory.AddMember(t.Context(), membership); err != nil {
			t.Fatal(err)
		}
	}
	if groups, err := s.Directory.GetUserGroups(t.Context(), "member"); err != nil || len(groups) != 1 || groups[0] != "local" {
		t.Fatal("legacy provider membership granted access", groups, err)
	}
	if err := s.Directory.SyncExternalGroups(t.Context(), "member", "oidc:one", []string{"operators"}, expires); err == nil {
		t.Fatal("unbound synchronization accepted an OIDC source")
	}
	if err := s.Directory.SyncProviderGroups(t.Context(), "member", "one", 1, []string{"operators"}, expires); err != nil {
		t.Fatal(err)
	}
	if groups, err := s.Directory.GetUserGroups(t.Context(), "member"); err != nil || len(groups) != 2 {
		t.Fatal("validated refresh did not activate provider membership", groups, err)
	}
	changeProvider(t, db, "one", func(provider *authoidc.Provider) { provider.Disabled = true })
	if groups, err := s.Directory.GetUserGroups(t.Context(), "member"); err != nil || len(groups) != 1 || groups[0] != "local" {
		t.Fatal("provider change removed local membership or retained provider access", groups, err)
	}
}

func TestOIDCIdentityProvisioningRequiresTheCurrentProviderAndExactIssuer(t *testing.T) {
	s, db := fixture(t)
	issuer := newIssuer(t)
	register(t, s, issuer, "one")
	for _, input := range []struct {
		revision int64
		issuer   string
	}{{0, issuer.server.URL}, {2, issuer.server.URL}, {1, "https://another-issuer.example.invalid"}} {
		if _, err := s.Directory.ResolveProviderIdentity(t.Context(), "one", input.revision, input.issuer, "verified-subject", "Subject", "same@example.invalid", true, true); err == nil {
			t.Fatal("unbound provider proof provisioned an identity")
		}
	}
	for _, bucket := range []string{"users", "external-identities"} {
		rows, err := db.All(t.Context(), bucket, "")
		if err != nil || len(rows) != 0 {
			t.Fatal("rejected provisioning changed identity state", bucket, err)
		}
	}
	user, err := s.Directory.ResolveProviderIdentity(t.Context(), "one", 1, issuer.server.URL, "verified-subject", "Subject", "same@example.invalid", true, true)
	if err != nil {
		t.Fatal(err)
	}
	link, err := s.Directory.ExternalLink(t.Context(), issuer.server.URL, "verified-subject")
	if err != nil || link.UserID != user.ID {
		t.Fatal("validated provider identity did not retain exact ownership", err)
	}
}
