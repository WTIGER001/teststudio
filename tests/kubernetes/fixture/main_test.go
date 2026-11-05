package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOIDCRealmProtocol(t *testing.T) {
	first, err := loadIdentity("")
	if err != nil {
		t.Fatal(err)
	}
	second := first
	second.Issuer = "https://identity-two.teststudio.test/review"
	second.ClientID = "teststudio-local-review"
	second.Callback = "https://studio.teststudio.test/auth/callback/local-k3s-review"
	second.Groups = []string{"reviewers"}
	for _, identity := range []identityConfig{first, second} {
		t.Run(identity.ClientID, func(t *testing.T) {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			handler := &fixture{identity: identity, key: key, codes: map[string]authorization{}}
			discovery := httptest.NewRecorder()
			handler.ServeHTTP(discovery, httptest.NewRequest("GET", identity.Issuer+"/.well-known/openid-configuration", nil))
			var document map[string]any
			if discovery.Code != 200 || json.Unmarshal(discovery.Body.Bytes(), &document) != nil || document["issuer"] != identity.Issuer || document["token_endpoint"] != identity.Issuer+"/token" {
				t.Fatal("realm discovery changed its issuer or endpoints")
			}
			verifier := strings.Repeat("p", 43)
			digest := sha256.Sum256([]byte(verifier))
			query := url.Values{"client_id": {identity.ClientID}, "redirect_uri": {identity.Callback}, "response_type": {"code"}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(digest[:])}, "nonce": {"realm-nonce"}, "state": {"realm-state"}}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", identity.Issuer+"/authorize?"+query.Encode(), nil))
			redirect, err := url.Parse(response.Header().Get("Location"))
			if err != nil || response.Code != 302 || redirect.Query().Get("state") != "realm-state" {
				t.Fatal("realm authorization failed")
			}
			form := url.Values{"client_id": {identity.ClientID}, "redirect_uri": {identity.Callback}, "grant_type": {"authorization_code"}, "code": {redirect.Query().Get("code")}, "code_verifier": {verifier}}
			request := httptest.NewRequest("POST", identity.Issuer+"/token", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var token map[string]any
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &token) != nil {
				t.Fatal("realm token exchange failed")
			}
			parts := strings.Split(token["id_token"].(string), ".")
			body, err := base64.RawURLEncoding.DecodeString(parts[1])
			var claims map[string]any
			if err != nil || json.Unmarshal(body, &claims) != nil || claims["iss"] != identity.Issuer || claims["aud"] != identity.ClientID || claims["sub"] != identity.Subject || claims["nonce"] != "realm-nonce" || claims["email"] != identity.Email {
				t.Fatal("realm claims escaped their configured authority")
			}
			rejected := httptest.NewRecorder()
			handler.ServeHTTP(rejected, httptest.NewRequest("GET", "https://unrelated.test/review/keys", nil))
			if rejected.Code != 404 {
				t.Fatal("the fixture served identity keys through another host")
			}
		})
	}
}
