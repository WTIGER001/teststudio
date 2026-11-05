package main

import (
	"crypto/tls"
	"errors"
)

var errRemoteCertificate = errors.New("remote gateway TLS certificate is unavailable or invalid")

func remoteCertificate(lookup func(string) (string, bool)) (tls.Certificate, error) {
	certificate, hasCertificate := lookup("TESTSTUDIO_GATEWAY_TLS_CERTIFICATE")
	key, hasKey := lookup("TESTSTUDIO_GATEWAY_TLS_KEY")
	if hasCertificate || hasKey {
		if !hasCertificate || !hasKey || len(certificate) < 1 || len(certificate) > 64<<10 || len(key) < 1 || len(key) > 64<<10 {
			return tls.Certificate{}, errRemoteCertificate
		}
		pair, err := tls.X509KeyPair([]byte(certificate), []byte(key))
		if err != nil {
			return tls.Certificate{}, errRemoteCertificate
		}
		return pair, nil
	}
	pair, err := tls.LoadX509KeyPair("/tls/tls.crt", "/tls/tls.key")
	if err != nil {
		return tls.Certificate{}, errRemoteCertificate
	}
	return pair, nil
}
