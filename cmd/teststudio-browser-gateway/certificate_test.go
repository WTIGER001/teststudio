package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestRemoteGatewayAcceptsPrivateCertificateEnvironmentWithoutSecretDiagnostics(t *testing.T) {
	authority := testkit.NewCertificateAuthority(t, "gateway certificate fixture")
	server := testkit.NewTLSServer(t, authority, "gateway.fixture.test", time.Hour, http.NotFoundHandler())
	expected := server.TLS.Certificates[0]
	var certificate strings.Builder
	for _, der := range expected.Certificate {
		certificate.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	}
	der, err := x509.MarshalPKCS8PrivateKey(expected.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	key := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	for _, mode := range []string{"valid", "missing key", "missing certificate", "empty key", "invalid key", "oversized certificate"} {
		t.Run(mode, func(t *testing.T) {
			values := map[string]string{"TESTSTUDIO_GATEWAY_TLS_CERTIFICATE": certificate.String(), "TESTSTUDIO_GATEWAY_TLS_KEY": key}
			switch mode {
			case "missing key":
				delete(values, "TESTSTUDIO_GATEWAY_TLS_KEY")
			case "missing certificate":
				delete(values, "TESTSTUDIO_GATEWAY_TLS_CERTIFICATE")
			case "empty key":
				values["TESTSTUDIO_GATEWAY_TLS_KEY"] = ""
			case "invalid key":
				values["TESTSTUDIO_GATEWAY_TLS_KEY"] = "private-key-diagnostic-canary"
			case "oversized certificate":
				values["TESTSTUDIO_GATEWAY_TLS_CERTIFICATE"] = strings.Repeat("x", 1+(64<<10))
			}
			pair, err := remoteCertificate(func(name string) (string, bool) { value, ok := values[name]; return value, ok })
			if mode == "valid" {
				if err != nil || len(pair.Certificate) != len(expected.Certificate) || pair.Leaf == nil || pair.Leaf.VerifyHostname("gateway.fixture.test") != nil {
					t.Fatal("private certificate material was not accepted", err)
				}
			} else if err != errRemoteCertificate || strings.Contains(err.Error(), "canary") || strings.Contains(err.Error(), "BEGIN") {
				t.Fatal("invalid private material did not produce its bounded diagnostic")
			}
		})
	}
}
