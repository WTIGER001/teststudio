package browsergateway

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

type remoteBridge struct {
	target     *url.URL
	api        bool
	connection RemoteConnection
	address    string
	tls        *tls.Config
	slots      chan struct{}
}

func (b *remoteBridge) String() string   { return "private remote gateway bridge" }
func (b *remoteBridge) GoString() string { return b.String() }

func newRemoteBridge(target browserpolicy.Target, connection RemoteConnection) (*remoteBridge, error) {
	if browserpolicy.ValidateTarget(target) != nil || connection.validate() != nil {
		return nil, ErrRemoteConfiguration
	}
	origin, _ := url.Parse(connection.Origin)
	port := origin.Port()
	if port == "" {
		port = "443"
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, ErrRemoteConfiguration
	}
	if len(connection.CertificateAuthority) > 0 && !roots.AppendCertsFromPEM(connection.CertificateAuthority) {
		return nil, ErrRemoteConfiguration
	}
	selected, _ := url.Parse(target.Origin)
	return &remoteBridge{
		target: selected, api: target.Requests != nil, connection: connection,
		address: net.JoinHostPort(origin.Hostname(), port), slots: make(chan struct{}, 64),
		tls: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: origin.Hostname(), RootCAs: roots, NextProtos: []string{"http/1.1"}},
	}, nil
}

func ServeRemoteBridge(ctx context.Context, listener net.Listener, target browserpolicy.Target, connection RemoteConnection) error {
	bridge, err := newRemoteBridge(target, connection)
	if err != nil {
		return err
	}
	bounded, cancel := context.WithDeadline(ctx, connection.Session.Deadline)
	defer cancel()
	return serveGateway(bounded, listener, bridge, target.Requests != nil, nil)
}

func (b *remoteBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
	default:
		http.Error(w, "Gateway bridge connection limit", http.StatusServiceUnavailable)
		return
	}
	if !time.Now().Before(b.connection.Session.Deadline) {
		http.Error(w, "Gateway execution has expired", http.StatusGatewayTimeout)
		return
	}
	if r.Header.Get("Upgrade") != "" || len(r.Trailer) > 0 || r.Method == http.MethodConnect && (r.ContentLength != 0 || len(r.TransferEncoding) != 0) {
		http.Error(w, "Gateway bridge does not permit this request framing", http.StatusForbidden)
		return
	}
	if r.ContentLength > 16<<20 {
		http.Error(w, "Gateway request exceeds its body limit", http.StatusRequestEntityTooLarge)
		return
	}
	setup, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	dialer := tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: b.tls}
	connection, err := dialer.DialContext(setup, "tcp", b.address)
	if err != nil {
		http.Error(w, "Trusted gateway TLS connection is unavailable", http.StatusBadGateway)
		return
	}
	defer connection.Close()
	stop := context.AfterFunc(r.Context(), func() { connection.Close() })
	defer stop()
	deadline, _ := setup.Deadline()
	if b.connection.Session.Deadline.Before(deadline) {
		deadline = b.connection.Session.Deadline
	}
	_ = connection.SetDeadline(deadline)
	request := b.request(w, r)
	if err := request.WriteProxy(connection); err != nil {
		http.Error(w, "Trusted gateway request transfer failed", http.StatusBadGateway)
		return
	}
	limited := &io.LimitedReader{R: connection, N: 64 << 10}
	reader := bufio.NewReader(limited)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		http.Error(w, "Trusted gateway response is unavailable", http.StatusBadGateway)
		return
	}
	limited.N = math.MaxInt64
	defer response.Body.Close()
	if r.Method == http.MethodConnect && response.StatusCode == http.StatusOK {
		_ = connection.SetDeadline(b.connection.Session.Deadline)
		b.tunnel(w, connection, reader)
		return
	}
	stripHopHeaders(response.Header)
	response.Header.Del(executionHeader)
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func (b *remoteBridge) request(w http.ResponseWriter, r *http.Request) *http.Request {
	request := r.Clone(r.Context())
	request.RequestURI = ""
	if r.Method == http.MethodConnect {
		request.URL = &url.URL{Host: r.Host}
	} else {
		if b.api && request.URL.Scheme == "" && request.URL.Host == "" && request.URL.Opaque == "" {
			request.URL.Scheme, request.URL.Host = b.target.Scheme, b.target.Host
		}
		request.Host = request.URL.Host
		request.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	}
	stripHopHeaders(request.Header)
	request.Header.Del("Expect")
	request.Header.Set("Proxy-Authorization", "Bearer "+b.connection.Session.Token)
	request.Header.Set(executionHeader, b.connection.Session.Execution)
	request.Close = true
	return request
}

func (b *remoteBridge) tunnel(w http.ResponseWriter, upstream net.Conn, reader *bufio.Reader) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Gateway bridge tunneling is unavailable", http.StatusInternalServerError)
		return
	}
	client, buffered, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()
	_ = client.SetDeadline(b.connection.Session.Deadline)
	if _, err := buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil || buffered.Flush() != nil {
		return
	}
	done := make(chan struct{})
	go func() { _, _ = io.Copy(upstream, buffered); upstream.Close(); close(done) }()
	_, _ = io.Copy(client, reader)
	client.Close()
	upstream.Close()
	<-done
}
