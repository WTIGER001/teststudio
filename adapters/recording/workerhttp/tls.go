package workerhttp

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"regexp"
)

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func material(filename, environment string) ([]byte, error) {
	if (filename == "") == (environment == "") {
		return nil, errors.New("recording transport requires one file or environment reference for each certificate input")
	}
	var data []byte
	if environment != "" {
		if !environmentName.MatchString(environment) {
			return nil, errors.New("recording certificate environment reference is invalid")
		}
		value, ok := os.LookupEnv(environment)
		if !ok {
			return nil, errors.New("recording certificate environment is unavailable")
		}
		data = []byte(value)
	} else {
		var err error
		data, err = os.ReadFile(filename)
		if err != nil {
			return nil, errors.New("recording certificate file is unavailable")
		}
	}
	if len(data) == 0 || len(data) > 64<<10 {
		return nil, errors.New("recording certificate material exceeds its limit")
	}
	return data, nil
}

func roots(filename string) (*x509.CertPool, error) {
	data, err := os.ReadFile(filename)
	pool := x509.NewCertPool()
	if err != nil || len(data) > 64<<10 || !pool.AppendCertsFromPEM(data) {
		return nil, errors.New("recording transport requires its dedicated certificate authority")
	}
	return pool, nil
}

func ServerTLS(certificate, key, clientCA, clientName string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil || clientName == "" {
		return nil, errors.New("recording worker requires its server certificate and API client identity")
	}
	pool, err := roots(clientCA)
	if err != nil {
		return nil, err
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert, VerifyConnection: func(state tls.ConnectionState) error {
		if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 || state.PeerCertificates[0].VerifyHostname(clientName) != nil {
			return errors.New("recording API client identity rejected")
		}
		return nil
	}}, nil
}
