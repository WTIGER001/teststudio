package oidc_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestOIDCKeyRotationAndRetiredSigningKeys(t *testing.T) {
	s, _ := fixture(t)
	remote := newIssuer(t)
	register(t, s, remote, "one")
	signIn(t, s, remote, "one")
	old := remote.key
	next, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	remote.keyMu.Lock()
	remote.key, remote.keyID = next, "key-two"
	remote.keyMu.Unlock()
	signIn(t, s, remote, "one")
	remote.keyMu.Lock()
	remote.publishedKey, remote.publishedID = next, "key-two"
	remote.key, remote.keyID = old, "key-one"
	remote.keyMu.Unlock()
	callback, err := login(t, s, remote, "one")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("retired signing key was accepted")
	}
	if remote.keys.Load() < 3 || remote.keys.Load() > 6 {
		t.Fatal("key refresh was absent or unbounded", remote.keys.Load())
	}
}

func TestOIDCRejectsWrongTokenUsePartyAndMissingIssueTime(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"header-access", func(c map[string]any) {}},
		{"wrong-issuer", func(c map[string]any) { c["iss"] = "https://wrong.example" }},
		{"access-use", func(c map[string]any) { c["token_use"] = "access" }},
		{"invalid-use", func(c map[string]any) { c["token_use"] = true }},
		{"single-audience-party", func(c map[string]any) { c["azp"] = "other-client" }},
		{"multiple-audiences-missing-party", func(c map[string]any) { c["aud"] = []string{"teststudio", "other-client"} }},
		{"missing-issued", func(c map[string]any) { delete(c, "iat") }},
		{"null-issued", func(c map[string]any) { c["iat"] = nil }},
		{"long-subject", func(c map[string]any) { c["sub"] = strings.Repeat("a", 256) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, db := fixture(t)
			remote := newIssuer(t)
			register(t, s, remote, "one")
			remote.mutate = test.mutate
			if test.name == "header-access" {
				remote.tokenType = "at+jwt"
			}
			callback, err := login(t, s, remote, "one")
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
				t.Fatal("invalid identity token accepted")
			}
			users, _ := db.All(t.Context(), "users", "")
			links, _ := db.All(t.Context(), "external-identities", "")
			stored, _ := db.All(t.Context(), "sessions", "")
			if len(users)+len(links)+len(stored) != 0 {
				t.Fatal("rejected token created identity state")
			}
		})
	}
}

func TestOIDCGroupClaimGapsWithdrawOnlyTheProviderMembership(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
		valid  bool
	}{
		{"missing", func(c map[string]any) { delete(c, "groups") }, false},
		{"null", func(c map[string]any) { c["groups"] = nil }, false},
		{"wrong-type", func(c map[string]any) { c["groups"] = "operators" }, false},
		{"mixed-values", func(c map[string]any) { c["groups"] = []any{"operators", 7} }, false},
		{"overflow", func(c map[string]any) { c["hasgroups"] = true }, false},
		{"distributed", func(c map[string]any) { c["_claim_names"] = map[string]any{"groups": "src"} }, false},
		{"invalid-distributed", func(c map[string]any) { c["_claim_names"] = "incomplete" }, false},
		{"next-page", func(c map[string]any) { c["groups@odata.nextLink"] = "https://groups.example/next" }, false},
		{"empty-identifier", func(c map[string]any) { c["groups"] = []string{"operators", ""} }, false},
		{"too-many", func(c map[string]any) {
			values := make([]string, 4097)
			for i := range values {
				values[i] = "operators"
			}
			c["groups"] = values
		}, false},
		{"complete-empty", func(c map[string]any) { c["groups"] = []string{} }, true},
		{"unmapped", func(c map[string]any) { c["groups"] = []string{"builtin.teststudio.installation-admin", "unknown"} }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, db := fixture(t)
			remote := newIssuer(t)
			register(t, s, remote, "one")
			for _, group := range []identity.Group{{ID: "external", Name: "External", ManagedBy: "oidc:one"}, {ID: "local", Name: "Local"}} {
				if err := s.Directory.SaveGroup(t.Context(), group, 0); err != nil {
					t.Fatal(err)
				}
			}
			row, _ := db.Get(t.Context(), "providers", "one")
			provider, _ := storage.Decode[map[string]any](row)
			provider["groupClaim"] = "groups"
			provider["groupMappings"] = map[string]string{"operators": "external"}
			provider["membershipTtlSeconds"] = 10
			change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, provider)
			if err := db.Commit(t.Context(), change); err != nil {
				t.Fatal(err)
			}
			current, _ := signIn(t, s, remote, "one")
			membership := identity.Membership{UserID: current.UserID, GroupID: "local", Source: "local"}
			local, _ := storage.Put("memberships", identity.MembershipID(current.UserID, "local", "local"), "", 0, membership)
			if err := db.Commit(t.Context(), local); err != nil {
				t.Fatal(err)
			}
			remote.mutate = test.mutate
			callback, err := login(t, s, remote, "one")
			if err != nil {
				t.Fatal(err)
			}
			err = s.Callback(httptest.NewRecorder(), callback, "one")
			if test.valid && err != nil {
				t.Fatal("complete claim rejected", err)
			}
			if !test.valid && err == nil {
				t.Fatal("incomplete claim accepted")
			}
			groups, err := s.Directory.GetUserGroups(t.Context(), current.UserID)
			if err != nil || len(groups) != 1 || groups[0] != "local" {
				t.Fatal("stale external access or lost local membership", groups, err)
			}
			valid, err := s.Sessions.Validate(t.Context(), current.ID)
			if err != nil || valid.AuthenticationUntil.After(time.Now().Add(11*time.Second)) {
				t.Fatal("membership lifetime did not bound session", err)
			}
		})
	}
}

func TestOIDCSignInRotatesThePriorBrowserSession(t *testing.T) {
	s, _ := fixture(t)
	remote := newIssuer(t)
	register(t, s, remote, "one")
	old, cookie := signIn(t, s, remote, "one")
	callback, err := login(t, s, remote, "one")
	if err != nil {
		t.Fatal(err)
	}
	callback.AddCookie(cookie)
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sessions.Validate(t.Context(), old.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("replaced browser session remained valid", err)
	}
}

func TestInvalidDistributedClaimsCannotWithdrawMembershipWithoutIdentityBinding(t *testing.T) {
	s, db := fixture(t)
	remote := newIssuer(t)
	register(t, s, remote, "one")
	if err := s.Directory.SaveGroup(t.Context(), identity.Group{ID: "external", Name: "External", ManagedBy: "oidc:one"}, 0); err != nil {
		t.Fatal(err)
	}
	row, _ := db.Get(t.Context(), "providers", "one")
	provider, _ := storage.Decode[map[string]any](row)
	provider["groupClaim"] = "groups"
	provider["groupMappings"] = map[string]string{"operators": "external"}
	change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, provider)
	if err := db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	current, _ := signIn(t, s, remote, "one")
	remote.mutate = func(claims map[string]any) { claims["_claim_names"] = "invalid"; claims["nonce"] = "wrong" }
	callback, err := login(t, s, remote, "one")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("unbound callback accepted")
	}
	groups, err := s.Directory.GetUserGroups(t.Context(), current.UserID)
	if err != nil || len(groups) != 1 || groups[0] != "external" {
		t.Fatal("unbound callback changed authoritative membership", groups, err)
	}
}

func TestExpiredProviderMembershipImmediatelyRemovesProjectPermission(t *testing.T) {
	s, db := fixture(t)
	remote := newIssuer(t)
	register(t, s, remote, "one")
	if err := s.Directory.SaveGroup(t.Context(), identity.Group{ID: "external", Name: "External", Project: "alpha", ManagedBy: "oidc:one"}, 0); err != nil {
		t.Fatal(err)
	}
	row, _ := db.Get(t.Context(), "providers", "one")
	provider, _ := storage.Decode[map[string]any](row)
	provider["groupClaim"] = "groups"
	provider["groupMappings"] = map[string]string{"operators": "external"}
	change, _ := storage.Put(row.Bucket, row.ID, "", row.Revision, provider)
	if err := db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	current, _ := signIn(t, s, remote, "one")
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), s.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: "external"}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), current.UserID, "alpha", "", "projects.read"); err != nil {
		t.Fatal("verified group did not grant its project role", err)
	}
	member, err := db.Get(t.Context(), "memberships", identity.MembershipID(current.UserID, "external", "oidc:one"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[identity.Membership](member)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Second)
	value.ExpiresAt = &past
	change, _ = storage.Put(member.Bucket, member.ID, member.Project, member.Revision, value)
	if err := db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), current.UserID, "alpha", "", "projects.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("expired group retained elevated access", err)
	}
	if _, err := s.Sessions.Validate(t.Context(), current.ID); err != nil {
		t.Fatal("permission revocation required ending an otherwise valid session", err)
	}
}

func TestUnresolvedDistributedEmailCannotActivatePendingAccess(t *testing.T) {
	for _, claim := range []string{"email", "email_verified"} {
		t.Run(claim, func(t *testing.T) {
			s, _ := fixture(t)
			remote := newIssuer(t)
			register(t, s, remote, "one")
			observer := &projectAccessObserver{}
			s.ProjectAccess = observer
			remote.mutate = func(claims map[string]any) { claims["_claim_names"] = map[string]string{claim: "unavailable"} }
			session, _ := signIn(t, s, remote, "one")
			if observer.calls != 1 || observer.last.EmailVerified {
				t.Fatal("unresolved email was presented as provider-verified")
			}
			user, err := s.Directory.User(t.Context(), session.UserID)
			if err != nil || user.EmailVerified {
				t.Fatal("unresolved verification entered profile metadata", err)
			}
		})
	}
}

func TestOIDCAccessTokenHashMustBindTheExchangedToken(t *testing.T) {
	for _, mode := range []string{"valid", "mismatch", "empty", "null"} {
		t.Run(mode, func(t *testing.T) {
			s, _ := fixture(t)
			remote := newIssuer(t)
			register(t, s, remote, "one")
			remote.mutate = func(claims map[string]any) {
				sum := sha256.Sum256([]byte("opaque"))
				claims["at_hash"] = base64.RawURLEncoding.EncodeToString(sum[:16])
				switch mode {
				case "mismatch":
					claims["at_hash"] = "other"
				case "empty":
					claims["at_hash"] = ""
				case "null":
					claims["at_hash"] = nil
				}
			}
			callback, err := login(t, s, remote, "one")
			if err != nil {
				t.Fatal(err)
			}
			err = s.Callback(httptest.NewRecorder(), callback, "one")
			if (err == nil) != (mode == "valid") {
				t.Fatal("unexpected token binding result", mode, err)
			}
		})
	}
}

func TestOIDCDiscoveryRejectsInsecureProtocolEndpoints(t *testing.T) {
	for _, name := range []string{"authorization_endpoint", "token_endpoint", "jwks_uri"} {
		t.Run(name, func(t *testing.T) {
			s, db := fixture(t)
			remote := newIssuer(t)
			register(t, s, remote, "one")
			remote.metadata = func(metadata map[string]any) { metadata[name] = "http://outside.example/endpoint" }
			if _, err := login(t, s, remote, "one"); err == nil {
				t.Fatal("insecure discovered endpoint accepted")
			}
			transactions, _ := db.All(t.Context(), "login-transactions", "")
			if len(transactions) != 0 {
				t.Fatal("rejected discovery created an authorization transaction")
			}
		})
	}
}
