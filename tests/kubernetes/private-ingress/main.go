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
	backend, _ := url.Parse("http://teststudio-worker.teststudio-local.svc.cluster.local")
	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "private exchange unavailable", http.StatusBadGateway)
	}
	server := &http.Server{Addr: ":8443", ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Host != "worker.teststudio.test" || !strings.HasPrefix(r.URL.Path, "/internal/worker/v1/executions/") {
			http.NotFound(w, r)
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
