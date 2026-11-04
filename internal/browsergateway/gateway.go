package browsergateway

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/networkguard"
)

type Gateway struct {
	origin    string
	authority string
	scheme    string
	dial      func(context.Context, string, string) (net.Conn, error)
	transport *http.Transport
	slots     chan struct{}
}

func New(target browserpolicy.Target) (*Gateway, error) {
	if err := browserpolicy.ValidateTarget(target); err != nil {
		return nil, err
	}
	u, _ := url.Parse(target.Origin)
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	scope := networkguard.Public
	if target.AllowPrivate {
		scope = networkguard.Private
	}
	g := &Gateway{origin: target.Origin, authority: net.JoinHostPort(u.Hostname(), port), scheme: u.Scheme, dial: networkguard.Dial(scope), slots: make(chan struct{}, 64)}
	g.transport = &http.Transport{Proxy: nil, DialContext: g.dial, DisableKeepAlives: true, ResponseHeaderTimeout: 20 * time.Second, MaxResponseHeaderBytes: 64 << 10}
	return g, nil
}

func (g *Gateway) Close() { g.transport.CloseIdleConnections() }

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case g.slots <- struct{}{}:
		defer func() { <-g.slots }()
	default:
		http.Error(w, "Browser gateway connection limit", http.StatusServiceUnavailable)
		return
	}
	if r.Method == http.MethodConnect {
		g.tunnel(w, r)
		return
	}
	origin, err := browserpolicy.Origin(r.URL.Scheme + "://" + r.URL.Host)
	if err != nil || origin != g.origin || g.scheme != "http" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.Header.Get("Upgrade") != "" {
		http.Error(w, "Origin is outside the selected browser target", http.StatusForbidden)
		return
	}
	request := r.Clone(r.Context())
	request.RequestURI = ""
	request.Host = r.URL.Host
	request.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	stripHopHeaders(request.Header)
	response, err := g.transport.RoundTrip(request)
	if err != nil {
		http.Error(w, "Selected browser target is unreachable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	stripHopHeaders(response.Header)
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func stripHopHeaders(header http.Header) {
	for _, key := range strings.Split(header.Get("Connection"), ",") {
		header.Del(strings.TrimSpace(key))
	}
	for _, key := range []string{"Connection", "Proxy-Connection", "Proxy-Authorization", "Proxy-Authenticate", "Keep-Alive", "TE", "Trailer", "Transfer-Encoding", "Upgrade"} {
		header.Del(key)
	}
}

func (g *Gateway) tunnel(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(g.scheme + "://" + r.Host)
	if err != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Port() == "" {
		http.Error(w, "Invalid browser tunnel", http.StatusForbidden)
		return
	}
	origin, err := browserpolicy.Origin(g.scheme + "://" + r.Host)
	if err != nil || origin != g.origin {
		http.Error(w, "Origin is outside the selected browser target", http.StatusForbidden)
		return
	}
	upstream, err := g.dial(r.Context(), "tcp", g.authority)
	if err != nil {
		http.Error(w, "Selected browser target is unreachable", http.StatusBadGateway)
		return
	}
	defer upstream.Close()
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Browser tunneling is unavailable", http.StatusInternalServerError)
		return
	}
	client, buffered, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()
	deadline := time.Now().Add(65 * time.Minute)
	_ = upstream.SetDeadline(deadline)
	_ = client.SetDeadline(deadline)
	if _, err := buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if buffered.Flush() != nil {
		return
	}
	done := make(chan struct{})
	go func() { _, _ = io.Copy(upstream, buffered); _ = upstream.Close(); close(done) }()
	_, _ = io.Copy(client, upstream)
	_ = client.Close()
	_ = upstream.Close()
	<-done
}

func Serve(ctx context.Context, listener net.Listener, target browserpolicy.Target) error {
	g, err := New(target)
	if err != nil {
		return err
	}
	defer g.Close()
	server := &http.Server{Handler: g, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 64 << 10}
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
		case <-finished:
			return
		}
		_ = server.Close()
		_ = listener.Close()
	}()
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
