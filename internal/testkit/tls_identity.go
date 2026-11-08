package testkit

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type TLSIdentity struct{ Certificate, Key, Authority string }

func NewTLSIdentity(t *testing.T, ca CertificateAuthority, name string, client bool) TLSIdentity {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		t.Fatal(err)
	}
	usage := x509.ExtKeyUsageServerAuth
	if client {
		usage = x509.ExtKeyUsageClientAuth
	}
	leaf := &x509.Certificate{SerialNumber: serial, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage}, DNSNames: []string{name}}
	if ip := net.ParseIP(name); ip != nil {
		leaf.IPAddresses = []net.IP{ip}
	}
	certificate, err := x509.CreateCertificate(rand.Reader, leaf, ca.Certificate, &key.PublicKey, ca.Key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	identity := TLSIdentity{Certificate: filepath.Join(root, "identity.pem"), Key: filepath.Join(root, "private.pem"), Authority: filepath.Join(root, "authority.pem")}
	for filename, data := range map[string][]byte{identity.Certificate: append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate}), ca.PEM...), identity.Key: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), identity.Authority: ca.PEM} {
		if err := os.WriteFile(filename, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return identity
}
