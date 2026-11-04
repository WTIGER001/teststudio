package acr

import (
	"crypto/tls"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
)

type transport struct {
	registry, prefix string
	publish          bool
	blobs            map[string]bool
	base             http.RoundTripper
	closed           *atomic.Bool
}

var digest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var uploadID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,255}$`)

func newTransport(options Options, blobs map[string]bool) *transport {
	base := options.Transport
	if base == nil {
		base = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 5 * time.Second, IdleConnTimeout: 30 * time.Second, MaxResponseHeaderBytes: 64 << 10}
	}
	return &transport{registry: options.Registry, prefix: options.RepositoryPrefix, publish: options.Publish, blobs: blobs, base: base}
}

func (t *transport) permitted(r *http.Request) bool {
	u := r.URL
	if t.closed != nil && t.closed.Load() || u == nil || u.Scheme != "https" || u.Host != u.Hostname() || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || r.Host != "" && r.Host != u.Host || r.Header.Get("Cookie") != "" || r.Header.Get("Proxy-Authorization") != "" {
		return false
	}
	if u.Host != t.registry {
		return t.blobs["https://"+u.Host] && r.Method == http.MethodGet && r.Header.Get("Authorization") == ""
	}
	if u.Path == "/oauth2/exchange" || u.Path == "/oauth2/token" {
		return r.Method == http.MethodPost && u.RawQuery == "" && !u.ForceQuery && r.Header.Get("Authorization") == "" && r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
	}
	if auth := r.Header.Get("Authorization"); auth != "" && !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	if !strings.HasPrefix(u.Path, "/v2/") {
		return false
	}
	rest := strings.TrimPrefix(u.Path, "/v2/")
	for _, kind := range []string{"manifests", "blobs"} {
		marker := "/" + kind + "/"
		i := strings.LastIndex(rest, marker)
		if i < 0 {
			continue
		}
		repo, tail := rest[:i], rest[i+len(marker):]
		if !oci.RepositoryWithinPrefix(repo, t.prefix) {
			continue
		}
		if digest.MatchString(tail) && u.RawQuery == "" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return true
		}
		if !t.publish {
			continue
		}
		if kind == "manifests" && r.Method == http.MethodPut && digest.MatchString(tail) && u.RawQuery == "" {
			return true
		}
		if kind == "blobs" && strings.HasPrefix(tail, "uploads/") {
			id := strings.TrimPrefix(tail, "uploads/")
			if id == "" && r.Method == http.MethodPost {
				return true
			}
			if uploadID.MatchString(id) && (r.Method == http.MethodPatch || r.Method == http.MethodPut || r.Method == http.MethodDelete) {
				if at := u.Query().Get("digest"); at != "" && !digest.MatchString(at) {
					return false
				}
				return true
			}
		}
	}
	return false
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !t.permitted(r) {
		return nil, oci.ErrDenied
	}
	response, err := t.base.RoundTrip(r)
	if err != nil || response == nil || response.Body == nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return nil, oci.ErrUnavailable
	}
	return response, nil
}

func (t *transport) CloseIdleConnections() {
	if c, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		c.CloseIdleConnections()
	}
}
