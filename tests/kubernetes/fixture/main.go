package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const issuer = "https://identity.teststudio.test"
const clientID = "teststudio-local-k3s"
const callback = "https://studio.teststudio.test/auth/callback/local-k3s"

type authorization struct {
	Nonce, Challenge string
	ExpiresAt        time.Time
}

type fixture struct {
	key   *rsa.PrivateKey
	mu    sync.Mutex
	codes map[string]authorization
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(value)
}

func (f *fixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	switch r.Method + " " + r.URL.Path {
	case "GET /.well-known/openid-configuration":
		writeJSON(w, map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	case "GET /keys":
		writeJSON(w, map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "local-k3s", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(f.key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.E)).Bytes())}}})
	case "GET /authorize":
		f.authorize(w, r)
	case "POST /token":
		f.token(w, r)
	case "GET /health":
		w.Header().Set("X-TestStudio-Fixture", "local-k3s")
		writeJSON(w, map[string]any{"status": "ready", "source": "new-local-kubernetes-fixture"})
	default:
		http.NotFound(w, r)
	}
}

func (f *fixture) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != clientID || q.Get("redirect_uri") != callback || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || len(q.Get("code_challenge")) != 43 || q.Get("nonce") == "" || q.Get("state") == "" {
		http.Error(w, "invalid local authorization", 400)
		return
	}
	code := rand.Text()
	f.mu.Lock()
	for key, value := range f.codes {
		if time.Now().After(value.ExpiresAt) {
			delete(f.codes, key)
		}
	}
	f.codes[code] = authorization{q.Get("nonce"), q.Get("code_challenge"), time.Now().Add(time.Minute)}
	f.mu.Unlock()
	redirect, _ := url.Parse(callback)
	redirect.RawQuery = url.Values{"code": {code}, "state": {q.Get("state")}}.Encode()
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, redirect.String(), 302)
}

func (f *fixture) token(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if r.ParseForm() != nil {
		http.Error(w, "invalid local token request", 400)
		return
	}
	f.mu.Lock()
	v, present := f.codes[r.PostForm.Get("code")]
	delete(f.codes, r.PostForm.Get("code"))
	f.mu.Unlock()
	challenge := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	client := r.PostForm.Get("client_id")
	if name, _, ok := r.BasicAuth(); ok {
		client = name
	}
	if !present || time.Now().After(v.ExpiresAt) || r.PostForm.Get("grant_type") != "authorization_code" || client != clientID || r.PostForm.Get("redirect_uri") != callback || base64.RawURLEncoding.EncodeToString(challenge[:]) != v.Challenge {
		http.Error(w, "invalid local token grant", 400)
		return
	}
	now := time.Now().Unix()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "local-k3s"})
	claims, _ := json.Marshal(map[string]any{"iss": issuer, "sub": "local-k3s-administrator", "aud": clientID, "iat": now, "auth_time": now, "exp": now + 900, "nonce": v.Nonce, "name": "Kubernetes integration administrator", "email": "local-k3s@example.test", "email_verified": true})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, digest[:])
	if err != nil {
		http.Error(w, "local token signing failed", 500)
		return
	}
	writeJSON(w, map[string]any{"access_token": rand.Text(), "token_type": "Bearer", "expires_in": 900, "id_token": unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)})
}

func run() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	handler := &fixture{key: key, codes: map[string]authorization{}}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	plain := &http.Server{Addr: ":8080", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	secure := &http.Server{Addr: ":8443", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	errors := make(chan error, 2)
	go func() { errors <- plain.ListenAndServe() }()
	go func() { errors <- secure.ListenAndServeTLS("/tls/tls.crt", "/tls/tls.key") }()
	select {
	case <-ctx.Done():
		err = nil
	case err = <-errors:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	plain.Shutdown(shutdown)
	secure.Shutdown(shutdown)
	return err
}

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
