package oidc_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type issuer struct {
	server      *httptest.Server
	key         *rsa.PrivateKey
	nonce       string
	challenge   string
	subject     string
	mutate      func(map[string]any)
	onToken     func()
	onDiscovery func()
	tokens      atomic.Int32
}

func newIssuer(t *testing.T) *issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	i := &issuer{key: key, subject: "subject-one"}
	i.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			if i.onDiscovery != nil {
				i.onDiscovery()
			}
			json.NewEncoder(w).Encode(map[string]any{"issuer": i.server.URL, "authorization_endpoint": i.server.URL + "/authorize", "token_endpoint": i.server.URL + "/token", "jwks_uri": i.server.URL + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key-one", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/token":
			i.tokens.Add(1)
			r.ParseForm()
			hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("code") != "code" || base64.RawURLEncoding.EncodeToString(hash[:]) != i.challenge {
				http.Error(w, "invalid PKCE verifier", 400)
				return
			}
			if i.onToken != nil {
				i.onToken()
			}
			claims := map[string]any{"iss": i.server.URL, "sub": i.subject, "aud": "teststudio", "iat": time.Now().Add(-time.Second).Unix(), "exp": time.Now().Add(time.Hour).Unix(), "nonce": i.nonce, "name": "Example User", "email": "same@example.invalid", "email_verified": true, "groups": []string{"operators"}}
			if i.mutate != nil {
				i.mutate(claims)
			}
			header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "key-one"})
			payload, _ := json.Marshal(claims)
			unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
			digest := sha256.Sum256([]byte(unsigned))
			sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
			if err != nil {
				t.Error(err)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "opaque", "token_type": "Bearer", "id_token": unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(i.server.Close)
	return i
}

func fixture(t *testing.T) (*authoidc.Service, *storage.Store) {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	sessions := &sessions.Service{Store: db, Directory: dir, Origin: "http://127.0.0.1:8040"}
	return &authoidc.Service{Store: db, Directory: dir, Sessions: sessions, Origin: sessions.Origin}, db
}

func register(t *testing.T, s *authoidc.Service, i *issuer, id string) {
	t.Helper()
	p := authoidc.Provider{ID: id, Name: id, Issuer: i.server.URL, ClientID: "teststudio", AutoProvision: true}
	c, err := storage.Put("providers", id, "", 0, p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Store.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
}

func login(t *testing.T, s *authoidc.Service, i *issuer, id string) (*http.Request, error) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", s.Origin+"/auth/login/"+id+"?returnTo=%2Fruns", nil)
	if err := s.Login(w, r, id); err != nil {
		return nil, err
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	i.nonce = location.Query().Get("nonce")
	i.challenge = location.Query().Get("code_challenge")
	if location.Query().Get("code_challenge_method") != "S256" || i.nonce == "" {
		t.Fatal("PKCE or nonce is missing")
	}
	callback := httptest.NewRequest("GET", s.Origin+"/auth/callback/"+id+"?code=code&state="+location.Query().Get("state"), nil)
	callback.AddCookie(w.Result().Cookies()[0])
	return callback, nil
}

func TestOIDCSignInReplayAndDistinctIssuers(t *testing.T) {
	s, db := fixture(t)
	a, b := newIssuer(t), newIssuer(t)
	register(t, s, a, "one")
	register(t, s, b, "two")
	users := map[string]bool{}
	for id, i := range map[string]*issuer{"one": a, "two": b} {
		callback, err := login(t, s, i, id)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		if err := s.Callback(w, callback, id); err != nil {
			t.Fatal(err)
		}
		if w.Code != 302 || w.Header().Get("Location") != "/runs" {
			t.Fatalf("callback response: %d %s", w.Code, w.Header().Get("Location"))
		}
		r := httptest.NewRequest("GET", s.Origin+"/auth/session", nil)
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == "teststudio_session" {
				r.AddCookie(cookie)
			}
		}
		session, err := s.Sessions.Read(t.Context(), r)
		if err != nil {
			t.Fatal(err)
		}
		users[session.UserID] = true
		if err := s.Callback(httptest.NewRecorder(), callback, id); err == nil {
			t.Fatal("callback replay was accepted")
		}
	}
	if len(users) != 2 {
		t.Fatal("same email merged two issuer identities")
	}
	rows, err := db.All(t.Context(), "external-identities", "")
	if err != nil || len(rows) != 2 {
		t.Fatalf("identity links: %+v %v", rows, err)
	}
}

func TestOIDCRejectsStateNonceAudienceAndExpiry(t *testing.T) {
	s, _ := fixture(t)
	i := newIssuer(t)
	register(t, s, i, "one")
	callback, err := login(t, s, i, "one")
	if err != nil {
		t.Fatal(err)
	}
	query := callback.URL.Query()
	query.Set("state", "wrong")
	callback.URL.RawQuery = query.Encode()
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("accepted wrong state")
	}
	for _, field := range []string{"nonce", "aud", "exp", "iss"} {
		t.Run(field, func(t *testing.T) {
			callback, err := login(t, s, i, "one")
			if err != nil {
				t.Fatal(err)
			}
			i.mutate = func(claims map[string]any) {
				if field == "exp" {
					claims[field] = time.Now().Add(-time.Hour).Unix()
				} else {
					claims[field] = "wrong"
				}
			}
			if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
				t.Fatalf("accepted wrong %s", field)
			}
		})
	}
}

func TestIncompleteGroupsRemovePreviousExternalMembership(t *testing.T) {
	s, db := fixture(t)
	i := newIssuer(t)
	register(t, s, i, "one")
	if err := s.Directory.SaveGroup(t.Context(), identity.Group{ID: "operators", Name: "Operators", ManagedBy: "oidc:one"}, 0); err != nil {
		t.Fatal(err)
	}
	r, err := db.Get(t.Context(), "providers", "one")
	if err != nil {
		t.Fatal(err)
	}
	p, err := storage.Decode[authoidc.Provider](r)
	if err != nil {
		t.Fatal(err)
	}
	p.GroupClaim = "groups"
	p.GroupMappings = map[string]string{"operators": "operators"}
	c, err := storage.Put(r.Bucket, r.ID, "", r.Revision, p)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	callback, err := login(t, s, i, "one")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err != nil {
		t.Fatal(err)
	}
	users, err := db.All(t.Context(), "users", "")
	if err != nil || len(users) != 1 {
		t.Fatal(err)
	}
	user, err := storage.Decode[identity.User](users[0])
	if err != nil {
		t.Fatal(err)
	}
	groups, err := s.Directory.GetUserGroups(t.Context(), user.ID)
	if err != nil || len(groups) != 1 {
		t.Fatalf("initial membership: %v %v", groups, err)
	}
	callback, err = login(t, s, i, "one")
	if err != nil {
		t.Fatal(err)
	}
	i.mutate = func(claims map[string]any) { delete(claims, "groups") }
	if err := s.Callback(httptest.NewRecorder(), callback, "one"); err == nil {
		t.Fatal("accepted incomplete group claims")
	}
	groups, err = s.Directory.GetUserGroups(t.Context(), user.ID)
	if err != nil || len(groups) != 0 {
		t.Fatalf("preserved stale membership: %v %v", groups, err)
	}
}
