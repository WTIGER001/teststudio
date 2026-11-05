package browsergateway

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func remoteTestSession() RemoteSession {
	return RemoteSession{Execution: "remote-execution", Token: base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 32))), Deadline: time.Now().Add(time.Minute)}
}

func startRemoteTestGateway(t *testing.T, gateway *Remote) (RemoteConnection, context.CancelFunc) {
	t.Helper()
	ca := testkit.NewCertificateAuthority(t, "remote gateway")
	certificate := testkit.NewTLSServer(t, ca, "127.0.0.1", time.Hour, http.NotFoundHandler())
	settings := certificate.TLS.Clone()
	certificate.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- gateway.serve(ctx, listener, settings) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("remote gateway does not stop")
		}
		gateway.Close()
	})
	return RemoteConnection{Origin: "https://" + listener.Addr().String(), CertificateAuthority: ca.PEM, Session: gateway.session}, cancel
}

func startRemoteTestBridge(t *testing.T, target browserpolicy.Target, connection RemoteConnection) (string, context.CancelFunc) {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "studio-bridge-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	path := filepath.Join(root, "gateway.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- ServeRemoteBridge(ctx, listener, target, connection) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("gateway bridge does not stop")
		}
		listener.Close()
	})
	return path, cancel
}

func TestRemoteBridgeKeepsCredentialAndPolicyInsideTrustedGateway(t *testing.T) {
	var hits atomic.Int32
	credentialValue := "Bearer remote-application-canary"
	applicationCA := testkit.NewCertificateAuthority(t, "application")
	upstream := testkit.NewTLSServer(t, applicationCA, "fixture.test", time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != credentialValue || r.Header.Get("Proxy-Authorization") != "" || r.Header.Get(executionHeader) != "" {
			t.Error("application credential missing or gateway capability reached target")
		}
		w.Header().Set("X-Reflected", credentialValue)
		io.WriteString(w, "approved "+credentialValue)
	}))
	defer upstream.Close()
	actual, _ := url.Parse(upstream.URL)
	target := browserpolicy.Target{
		Origin: "https://fixture.test:" + actual.Port(), AllowPrivate: true, Authenticated: true,
		Requests: &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "reader", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/approved", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 4096},
	}
	session := remoteTestSession()
	gateway, err := NewRemote(Configuration{Target: target, Credential: &apicredential.Value{Origin: target.Origin, Header: "Authorization", Value: credentialValue, Network: networkguard.Private}, Remote: &session})
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(applicationCA.PEM)
	gateway.gateway.transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	dial := gateway.gateway.dial
	gateway.gateway.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) { return dial(ctx, network, actual.Host) }
	connection, _ := startRemoteTestGateway(t, gateway)
	if strings.Contains(fmt.Sprintf("%+v %#v %+v %#v", connection, connection, gateway, Configuration{Remote: &session}), session.Token) {
		t.Fatal("gateway diagnostic formatting discloses transport capability")
	}
	path, _ := startRemoteTestBridge(t, target, connection)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", path)
	}}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	for _, check := range []struct {
		method, path string
		status       int
	}{{"GET", "/outside", 403}, {"DELETE", "/approved", 403}, {"GET", "/approved", 200}} {
		request, _ := http.NewRequest(check.method, "http://bridge"+check.path, nil)
		request.Header.Set("Authorization", "spoofed")
		request.Header.Set("Proxy-Authorization", "spoofed")
		request.Header.Set(executionHeader, "other-execution")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != check.status || strings.Contains(string(body)+response.Header.Get("X-Reflected"), "canary") {
			t.Fatal("remote policy or credential redaction fails", response.StatusCode, err)
		}
	}
	proxy, _ := url.Parse("http://bridge")
	transport.Proxy = http.ProxyURL(proxy)
	response, err := client.Get("http://foreign.test/approved")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 || hits.Load() != 1 {
		t.Fatal("foreign origin or denied request reached target")
	}
}

func TestRemoteGatewayRequiresScopedAuthorizationAndVerifiedTLS(t *testing.T) {
	session := remoteTestSession()
	target := browserpolicy.Target{Origin: "http://fixture.test"}
	gateway, err := NewRemote(Configuration{Target: target, Remote: &session})
	if err != nil {
		t.Fatal(err)
	}
	var dials atomic.Int32
	gateway.gateway.transport.DialContext = func(context.Context, string, string) (net.Conn, error) { dials.Add(1); return nil, net.ErrClosed }
	connection, _ := startRemoteTestGateway(t, gateway)
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(connection.CertificateAuthority)
	for _, check := range []struct {
		name, token, execution string
		duplicate              bool
	}{
		{"missing", "", "", false}, {"wrong token", "other", session.Execution, false}, {"wrong execution", session.Token, "other", false}, {"duplicate", session.Token, session.Execution, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			address, _ := url.Parse(connection.Origin)
			socket, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, "tcp", address.Host, &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12})
			if err != nil {
				t.Fatal(err)
			}
			defer socket.Close()
			socket.SetDeadline(time.Now().Add(2 * time.Second))
			request, _ := http.NewRequest("GET", target.Origin+"/check", nil)
			if check.token != "" {
				request.Header.Set("Proxy-Authorization", "Bearer "+check.token)
			}
			if check.execution != "" {
				request.Header.Set(executionHeader, check.execution)
			}
			if check.duplicate {
				request.Header.Add("Proxy-Authorization", "Bearer "+check.token)
			}
			if err := request.WriteProxy(socket); err != nil {
				t.Fatal(err)
			}
			response, err := http.ReadResponse(bufio.NewReader(socket), request)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != 407 || strings.Contains(string(body), session.Token) {
				t.Fatal("invalid gateway authority accepted or disclosed")
			}
		})
	}
	request := httptest.NewRequest("GET", target.Origin+"/check", nil)
	request.Header.Set("Proxy-Authorization", "Bearer "+session.Token)
	request.Header.Set(executionHeader, session.Execution)
	plain := httptest.NewRecorder()
	gateway.ServeHTTP(plain, request)
	if plain.Code != 403 {
		t.Fatal("plaintext capability accepted")
	}
	otherCA := testkit.NewCertificateAuthority(t, "unrelated authority")
	connection.CertificateAuthority = otherCA.PEM
	bridge, err := newRemoteBridge(target, connection)
	if err != nil {
		t.Fatal(err)
	}
	result := httptest.NewRecorder()
	bridge.ServeHTTP(result, httptest.NewRequest("GET", target.Origin+"/check", nil))
	if result.Code != 502 || dials.Load() != 0 {
		t.Fatal("untrusted TLS or invalid authorization reached target")
	}
}

func TestRemoteGatewayLifecycleClosesActiveBrowserTunnel(t *testing.T) {
	for _, mode := range []string{"gateway cancellation", "bridge cancellation", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			upstream, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer upstream.Close()
			session := remoteTestSession()
			if mode == "deadline" {
				session.Deadline = time.Now().Add(2 * time.Second)
			}
			target := browserpolicy.Target{Origin: "https://fixture.test", AllowPrivate: true}
			gateway, err := NewRemote(Configuration{Target: target, Remote: &session})
			if err != nil {
				t.Fatal(err)
			}
			dial := gateway.gateway.dial
			gateway.gateway.dial = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return dial(ctx, network, upstream.Addr().String())
			}
			connection, stop := startRemoteTestGateway(t, gateway)
			path, stopBridge := startRemoteTestBridge(t, target, connection)
			client, err := net.DialTimeout("unix", path, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			client.SetDeadline(time.Now().Add(3 * time.Second))
			fmt.Fprint(client, "CONNECT fixture.test:443 HTTP/1.1\r\nHost: fixture.test:443\r\n\r\n")
			reader := bufio.NewReader(client)
			response, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
			if err != nil || response.StatusCode != 200 {
				t.Fatal("tunnel unavailable", err)
			}
			upstream.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second))
			accepted, err := upstream.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer accepted.Close()
			accepted.SetDeadline(time.Now().Add(3 * time.Second))
			client.Write([]byte("probe"))
			payload := make([]byte, 5)
			if _, err := io.ReadFull(accepted, payload); err != nil || string(payload) != "probe" {
				t.Fatal("tunnel did not forward", err)
			}
			switch mode {
			case "gateway cancellation":
				stop()
			case "bridge cancellation":
				stopBridge()
			}
			if _, err := reader.ReadByte(); err == nil {
				t.Fatal("canceled gateway retains client tunnel")
			} else if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				t.Fatal("client tunnel only closed at test timeout")
			}
			if _, err := accepted.Read(payload); err == nil {
				t.Fatal("canceled gateway retains upstream tunnel")
			} else if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				t.Fatal("upstream tunnel only closed at test timeout")
			}
		})
	}
}
