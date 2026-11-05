package main

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	credential, err := os.ReadFile("/credentials/reader")
	if err != nil || len(strings.TrimSpace(string(credential))) < 32 {
		log.Fatal("The local target requires its fixture credential")
	}
	expected := "Bearer " + strings.TrimSpace(string(credential))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html lang="en"><meta charset="utf-8"><title>Gateway workspace</title><h1>Gateway workspace</h1><button type="button">Load availability</button><output aria-live="polite"></output><script>document.querySelector('button').addEventListener('click', async () => { const response = await fetch('/api/status'); const status = await response.json(); document.querySelector('output').textContent = status.status + ': ' + status.itemCount + ' items'; });</script></html>`))
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ready", "itemCount": 3})
	})
	mux.HandleFunc("GET /api/private/status", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			http.Error(w, "A TLS fixture credential is required", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Credential-Echo", expected)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready", "receivedCredential": expected})
	})
	server := func(address string) *http.Server {
		return &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 10 * time.Second}
	}
	go func() { log.Fatal(server(":8080").ListenAndServe()) }()
	log.Fatal(server(":8443").ListenAndServeTLS("/tls/tls.crt", "/tls/tls.key"))
}
