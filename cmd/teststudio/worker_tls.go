package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
)

type workerTLSOptions struct {
	Origin      string
	Certificate string
	Key         string
}

var errWorkerTLS = errors.New("direct worker TLS requires an HTTPS origin, a matching current certificate chain, and its mounted private key")

func (o workerTLSOptions) load(now time.Time) (*tls.Config, string, error) {
	if o == (workerTLSOptions{}) {
		return nil, "", nil
	}
	origin, err := relay.ParseHTTPSOrigin(o.Origin)
	if err != nil || o.Certificate == "" || o.Key == "" {
		return nil, "", errWorkerTLS
	}
	certificate, err := tls.LoadX509KeyPair(o.Certificate, o.Key)
	if err != nil || len(certificate.Certificate) == 0 {
		return nil, "", errWorkerTLS
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil || leaf.VerifyHostname(origin.Hostname()) != nil || now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) {
		return nil, "", errWorkerTLS
	}
	if len(leaf.ExtKeyUsage) != 0 || len(leaf.UnknownExtKeyUsage) != 0 {
		server := false
		for _, usage := range leaf.ExtKeyUsage {
			server = server || usage == x509.ExtKeyUsageServerAuth || usage == x509.ExtKeyUsageAny
		}
		if !server {
			return nil, "", errWorkerTLS
		}
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}
	config.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
		if strings.EqualFold(hello.ServerName, origin.Hostname()) || hello.ServerName == "" && net.ParseIP(origin.Hostname()) != nil {
			return nil, nil
		}
		return nil, errWorkerTLS
	}
	return config, origin.Host, nil
}

func workerAuthorityHandler(next http.Handler, authority string) http.Handler {
	if authority == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Host, authority) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "worker request unavailable"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
