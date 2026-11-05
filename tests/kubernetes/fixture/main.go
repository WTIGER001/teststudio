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
	"flag"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type identityConfig struct {
	Issuer   string   `json:"issuer"`
	ClientID string   `json:"clientId"`
	Callback string   `json:"callback"`
	Subject  string   `json:"subject"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups,omitempty"`
}

func loadIdentity(path string) (identityConfig, error) {
	value := identityConfig{Issuer: "https://identity.teststudio.test", ClientID: "teststudio-local-k3s", Callback: "https://studio.teststudio.test/auth/callback/local-k3s", Subject: "local-k3s-administrator", Name: "Kubernetes integration administrator", Email: "local-k3s@example.test"}
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return value, err
		}
		defer file.Close()
		decoder := json.NewDecoder(io.LimitReader(file, 8192))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&value); err != nil {
			return value, err
		}
		if decoder.Decode(new(any)) != io.EOF {
			return value, errors.New("invalid identity configuration")
		}
	}
	issuer, err := url.Parse(value.Issuer)
	if err != nil || issuer.Scheme != "https" || (issuer.Host != "identity.teststudio.test" && issuer.Host != "identity-two.teststudio.test") || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" || issuer.RawPath != "" || strings.HasSuffix(issuer.Path, "/") || strings.Contains(issuer.Path, "..") {
		return value, errors.New("identity issuer must use the owned HTTPS host and a fixed optional realm path")
	}
	callback, err := url.Parse(value.Callback)
	if err != nil || callback.Scheme != "https" || callback.Host != "studio.teststudio.test" || callback.User != nil || callback.RawQuery != "" || callback.Fragment != "" || !strings.HasPrefix(callback.Path, "/auth/callback/") || value.ClientID == "" || value.Subject == "" || value.Email == "" {
		return value, errors.New("identity callback and claims are incomplete")
	}
	return value, nil
}

type authorization struct {
	Nonce, Challenge string
	ExpiresAt        time.Time
}

type fixture struct {
	identity identityConfig
	key      *rsa.PrivateKey
	mu       sync.Mutex
	codes    map[string]authorization
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(value)
}

func (f *fixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method == "GET" && r.URL.Path == "/health" {
		w.Header().Set("X-TestStudio-Fixture", "local-k3s")
		writeJSON(w, map[string]any{"status": "ready", "source": "new-local-kubernetes-fixture"})
		return
	}
	origin, _ := url.Parse(f.identity.Issuer)
	if r.Host != origin.Host || !strings.HasPrefix(r.URL.Path, origin.Path+"/") {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, origin.Path)
	switch r.Method + " " + path {
	case "GET /.well-known/openid-configuration":
		writeJSON(w, map[string]any{"issuer": f.identity.Issuer, "authorization_endpoint": f.identity.Issuer + "/authorize", "token_endpoint": f.identity.Issuer + "/token", "jwks_uri": f.identity.Issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	case "GET /keys":
		writeJSON(w, map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "local-k3s", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(f.key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.E)).Bytes())}}})
	case "GET /authorize":
		f.authorize(w, r)
	case "POST /token":
		f.token(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (f *fixture) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != f.identity.ClientID || q.Get("redirect_uri") != f.identity.Callback || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || len(q.Get("code_challenge")) != 43 || q.Get("nonce") == "" || q.Get("state") == "" {
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
	redirect, _ := url.Parse(f.identity.Callback)
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
	if !present || time.Now().After(v.ExpiresAt) || r.PostForm.Get("grant_type") != "authorization_code" || client != f.identity.ClientID || r.PostForm.Get("redirect_uri") != f.identity.Callback || base64.RawURLEncoding.EncodeToString(challenge[:]) != v.Challenge {
		http.Error(w, "invalid local token grant", 400)
		return
	}
	now := time.Now().Unix()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "local-k3s"})
	values := map[string]any{"iss": f.identity.Issuer, "sub": f.identity.Subject, "aud": f.identity.ClientID, "iat": now, "auth_time": now, "exp": now + 900, "nonce": v.Nonce, "name": f.identity.Name, "email": f.identity.Email, "email_verified": true}
	if len(f.identity.Groups) > 0 {
		values["groups"] = f.identity.Groups
	}
	claims, _ := json.Marshal(values)
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
	config := flag.String("identity-config", "", "Owned local identity fixture configuration")
	flag.Parse()
	identity, err := loadIdentity(*config)
	if err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	handler := &fixture{identity: identity, key: key, codes: map[string]authorization{}}
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
