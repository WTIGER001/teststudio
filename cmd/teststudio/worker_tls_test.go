package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func workerTLSFixture(t *testing.T, change func(*x509.Certificate)) (workerTLSOptions, *x509.CertPool) {
	t.Helper()
	ca := testkit.NewCertificateAuthority(t, "Owned worker backend authority")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: []string{"backend.example.test", "other-san.example.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if change != nil {
		change(leaf)
	}
	der, err := x509.CreateCertificate(rand.Reader, leaf, ca.Certificate, &key.PublicKey, ca.Key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	options := workerTLSOptions{Origin: "https://backend.example.test", Certificate: filepath.Join(root, "certificate.pem"), Key: filepath.Join(root, "private-key-canary.pem")}
	chain := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), ca.PEM...)
	if err := os.WriteFile(options.Certificate, chain, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.Key, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca.Certificate)
	return options, roots
}

func TestDirectWorkerTLSRejectsInvalidMaterialBeforeOpeningStorage(t *testing.T) {
	valid, _ := workerTLSFixture(t, nil)
	if config, authority, err := valid.load(time.Now()); err != nil || config == nil || config.MinVersion != tls.VersionTLS12 || authority != "backend.example.test" {
		t.Fatal("valid mounted identity rejected", err)
	}
	if config, authority, err := (workerTLSOptions{}).load(time.Now()); err != nil || config != nil || authority != "" {
		t.Fatal("external termination changed", err)
	}
	for name, change := range map[string]func(*x509.Certificate){
		"expired":      func(c *x509.Certificate) { c.NotAfter = time.Now().Add(-time.Minute) },
		"future":       func(c *x509.Certificate) { c.NotBefore = time.Now().Add(time.Minute) },
		"foreign-host": func(c *x509.Certificate) { c.DNSNames = []string{"foreign.example.test"} },
		"client-only":  func(c *x509.Certificate) { c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth} },
		"unknown-purpose": func(c *x509.Certificate) {
			c.ExtKeyUsage = nil
			c.UnknownExtKeyUsage = []asn1.ObjectIdentifier{{1, 2, 3, 4}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			options, _ := workerTLSFixture(t, change)
			if _, _, err := options.load(time.Now()); err != errWorkerTLS {
				t.Fatal("invalid mounted identity accepted", err)
			}
		})
	}
	for _, change := range []func(*workerTLSOptions){
		func(o *workerTLSOptions) { o.Origin = "" },
		func(o *workerTLSOptions) { o.Origin = "http://backend.example.test" },
		func(o *workerTLSOptions) { o.Origin += "/path" },
		func(o *workerTLSOptions) { o.Certificate = "" },
		func(o *workerTLSOptions) { o.Key = "" },
		func(o *workerTLSOptions) { o.Key += "-missing" },
		func(o *workerTLSOptions) { o.Key = o.Certificate },
	} {
		options := valid
		change(&options)
		config, authority, err := options.load(time.Now())
		if err != errWorkerTLS || config != nil || authority != "" || strings.Contains(err.Error(), "private-key-canary") {
			t.Fatal("invalid TLS settings accepted or disclosed", err)
		}
	}
	root := t.TempDir()
	database := filepath.Join(root, "not-created", "backend.db")
	err := runWorkerBackend(t.Context(), "127.0.0.1:0", "azure", database, "", "", "", filepath.Join(root, "artifacts"), workerTLSOptions{Origin: valid.Origin})
	if err != errWorkerTLS {
		t.Fatal("incomplete TLS settings entered service startup", err)
	}
	if _, err := os.Stat(filepath.Dir(database)); !os.IsNotExist(err) {
		t.Fatal("failed TLS startup opened storage", err)
	}
}

func TestDirectWorkerTLSCommandCarriesRelayProtocolAndRejectsForeignAuthority(t *testing.T) {
	f := newBackendFixture(t)
	ticket := f.executionTicket(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_, port, _ := net.SplitHostPort(address)
	listener.Close()
	options, roots := workerTLSFixture(t, nil)
	options.Origin += ":" + port
	startWorkerBackendCommand(t, address, "--serve-role=worker", "--worker-addr="+address, "--execution-location=azure", "--database="+f.database, "--artifact-directory="+f.files, "--worker-origin="+options.Origin, "--worker-tls-certificate="+options.Certificate, "--worker-tls-key="+options.Key)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", address)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: time.Second}
	ready := false
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		response, err := client.Get(options.Origin + "/healthz")
		if err == nil {
			response.Body.Close()
			ready = response.StatusCode == 200
		}
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !ready {
		t.Fatal("direct TLS backend did not become ready")
	}
	front := httptest.NewUnstartedServer(nil)
	t.Cleanup(front.Close)
	forwarder, err := relay.New("https://"+front.Listener.Addr().String(), options.Origin, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(forwarder.Close)
	front.Config.Handler = forwarder
	front.StartTLS()
	worker, err := workerclient.New(front.URL, ticket.Execution, "tls-worker", ticket.Token, front.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(worker.Close)
	packet, err := worker.Input(t.Context())
	if err != nil || packet.Execution != ticket.Execution {
		t.Fatal("real relay could not fetch authorized input", err)
	}
	now := time.Now().UTC()
	if err := worker.Started(t.Context(), now); err != nil {
		t.Fatal(err)
	}
	body := []byte("owned direct TLS diagnostic")
	sum := sha256.Sum256(body)
	report := workerexchange.Submission{Report: execution.CodedReport{Schema: "teststudio.playwright/v1", TestID: "PAGE-1", Selector: "@PAGE-1", Result: execution.Result{TestID: "PAGE-1", State: "passed", StartedAt: now, FinishedAt: now.Add(time.Millisecond), DurationMs: 1}, Attachments: []execution.Attachment{{Path: "tls.log", Kind: "log", Filename: "tls.log", ContentType: "text/plain"}}}, Files: []workerexchange.File{{Path: "tls.log", Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])}}}
	if err := worker.Submit(t.Context(), report); err != nil {
		t.Fatal("report did not cross verified backend TLS", err)
	}
	if err := worker.Upload(t.Context(), "tls.log", bytes.NewReader(body)); err != nil {
		t.Fatal("attachment did not cross verified backend TLS", err)
	}
	row, err := f.db.Get(t.Context(), workerexchange.Bucket, ticket.Execution)
	if err != nil {
		t.Fatal(err)
	}
	record, err := storage.Decode[workerexchange.Record](row)
	if err != nil || record.Submission == nil || !record.Uploaded["tls.log"] || record.Instance != "tls-worker" {
		t.Fatal("TLS backend did not persist scoped evidence", err)
	}
	request, _ := http.NewRequest("GET", options.Origin+"/livez", nil)
	request.Host = "foreign.example.test"
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 404 || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("foreign HTTP authority was accepted")
	}
	for _, config := range []*tls.Config{
		{RootCAs: roots, MinVersion: tls.VersionTLS12, ServerName: "other-san.example.test"},
		{RootCAs: x509.NewCertPool(), MinVersion: tls.VersionTLS12, ServerName: "backend.example.test"},
		{RootCAs: roots, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11, ServerName: "backend.example.test"},
	} {
		connection, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, "tcp", address, config)
		if err == nil {
			connection.Close()
			t.Fatal("foreign SNI, untrusted issuer, or legacy TLS was accepted")
		}
	}
	plain := &http.Client{Timeout: time.Second, Transport: &http.Transport{}}
	t.Cleanup(plain.CloseIdleConnections)
	response, err = plain.Get("http://" + address + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatal("TLS endpoint accepted plaintext HTTP")
	}
	row, _ = f.db.Get(t.Context(), "users", "owner")
	owner, _ := storage.Decode[identity.User](row)
	owner.Disabled = true
	if err := f.auth.Directory.SaveUser(t.Context(), owner, row.Revision); err != nil {
		t.Fatal(err)
	}
	if err := worker.Started(t.Context(), now); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("relay retained write access after live permission loss", err)
	}
}
