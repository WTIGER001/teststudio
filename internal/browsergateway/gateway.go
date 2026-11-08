package browsergateway

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/diagnostics"
	"github.com/wtiger001/teststudio/internal/networkguard"
)

type Gateway struct {
	credential *apicredential.Value
	private    *strings.Replacer
	origin     string
	authority  string
	scheme     string
	origins    map[string]string
	dial       func(context.Context, string, string) (net.Conn, error)
	transport  *http.Transport
	slots      chan struct{}
	requests   *apipolicy.Spec
	count      atomic.Int32
}

func New(target browserpolicy.Target) (*Gateway, error) {
	return NewAuthenticated(target, nil)
}

func NewAuthenticated(target browserpolicy.Target, credential *apicredential.Value) (*Gateway, error) {
	if err := browserpolicy.ValidateTarget(target); err != nil {
		return nil, err
	}
	if err := apicredential.Validate(target, credential); err != nil {
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
	if credential != nil && credential.Network == networkguard.Public {
		scope = networkguard.Public
	}
	g := &Gateway{origin: target.Origin, authority: net.JoinHostPort(u.Hostname(), port), scheme: u.Scheme, dial: networkguard.Dial(scope), slots: make(chan struct{}, 64), requests: target.Requests}
	g.origins = map[string]string{target.Origin: g.authority}
	for _, origin := range target.AdditionalOrigins {
		parsed, _ := url.Parse(origin)
		port := parsed.Port()
		if port == "" {
			port = "443"
			if parsed.Scheme == "http" {
				port = "80"
			}
		}
		g.origins[origin] = net.JoinHostPort(parsed.Hostname(), port)
	}
	if credential != nil {
		copy := *credential
		g.credential = &copy
		g.private = diagnostics.LiteralReplacer(credential.PrivateValues())
	}
	if target.Requests != nil {
		copy := *target.Requests
		copy.Rules = append([]apipolicy.Rule(nil), target.Requests.Rules...)
		g.requests, g.slots = &copy, make(chan struct{}, 4)
	}
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
	if g.requests != nil {
		g.api(w, r)
		return
	}
	if r.Method == http.MethodConnect {
		g.tunnel(w, r)
		return
	}
	origin, err := browserpolicy.Origin(r.URL.Scheme + "://" + r.URL.Host)
	if err != nil || g.origins[origin] == "" || r.URL.Scheme != "http" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.Header.Get("Upgrade") != "" {
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
	u, err := url.Parse("https://" + r.Host)
	if err != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Port() == "" {
		http.Error(w, "Invalid browser tunnel", http.StatusForbidden)
		return
	}
	origin, err := browserpolicy.Origin("https://" + r.Host)
	if err == nil && g.origins[origin] == "" {
		origin, err = browserpolicy.Origin("http://" + r.Host)
	}
	if err != nil || g.origins[origin] == "" {
		http.Error(w, "Origin is outside the selected browser target", http.StatusForbidden)
		return
	}
	upstream, err := g.dial(r.Context(), "tcp", g.origins[origin])
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
	return ServeAuthenticated(ctx, listener, target, nil)
}

func ServeAuthenticated(ctx context.Context, listener net.Listener, target browserpolicy.Target, credential *apicredential.Value) error {
	g, err := NewAuthenticated(target, credential)
	if err != nil {
		return err
	}
	defer g.Close()
	return serveGateway(ctx, listener, g, target.Requests != nil, nil)
}
