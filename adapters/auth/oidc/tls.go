package oidc

import (
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/certificates"
)

func TLSClient(caFile string) (*http.Client, error) {
	if caFile == "" {
		return nil, nil
	}
	roots, _, err := certificates.LoadAuthorities(caFile)
	if err != nil {
		return nil, errors.New("OIDC CA file must contain only valid PEM certificate authorities within 64 KiB")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second}, nil
}
