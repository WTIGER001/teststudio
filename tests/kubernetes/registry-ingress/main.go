package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	backend, _ := url.Parse("http://127.0.0.1:5000")
	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "local registry unavailable", http.StatusBadGateway)
	}
	server := &http.Server{Addr: ":5443", ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		if r.Host != "registry.teststudio.test" || r.URL.RawPath != "" || r.URL.Path != "/v2/" && !strings.HasPrefix(r.URL.Path, "/v2/teststudio/") {
			http.NotFound(w, r)
			return
		}
		user, _, present := r.BasicAuth()
		if !present || user != "teststudio-reader" && user != "teststudio-writer" {
			w.Header().Set("WWW-Authenticate", `Basic realm="TestStudio local registry"`)
			http.Error(w, "registry authentication required", http.StatusUnauthorized)
			return
		}
		if user == "teststudio-reader" && r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "registry reader cannot write", http.StatusForbidden)
			return
		}
		proxy.ServeHTTP(w, r)
	})}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServeTLS("/tls/tls.crt", "/tls/tls.key"); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
