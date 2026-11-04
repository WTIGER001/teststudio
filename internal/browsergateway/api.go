package browsergateway

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

func (g *Gateway) api(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect || r.Header.Get("Upgrade") != "" || len(r.Trailer) > 0 {
		http.Error(w, "API policy does not permit tunnels or protocol upgrades", http.StatusForbidden)
		return
	}
	if encoding := strings.TrimSpace(r.Header.Get("Content-Encoding")); encoding != "" && !strings.EqualFold(encoding, "identity") {
		http.Error(w, "API request policy requires uncompressed payloads", http.StatusUnsupportedMediaType)
		return
	}
	if r.URL.IsAbs() {
		origin, err := browserpolicy.Origin(r.URL.Scheme + "://" + r.URL.Host)
		if err != nil || origin != g.origin {
			http.Error(w, "API origin is outside the selected target", http.StatusForbidden)
			return
		}
	}
	rule, err := apipolicy.Match(g.requests, r.Method, r.URL)
	if err != nil {
		http.Error(w, "API method or path is outside the reviewed policy", http.StatusForbidden)
		return
	}
	if rule.ReadOnly {
		for _, header := range []string{"X-HTTP-Method-Override", "X-Method-Override", "X-HTTP-Method"} {
			if r.Header.Get(header) != "" {
				http.Error(w, "Read-only API rules do not permit method overrides", http.StatusForbidden)
				return
			}
		}
	}
	if r.ContentLength > int64(rule.MaxBodyBytes) {
		http.Error(w, "API request exceeds its reviewed body limit", http.StatusRequestEntityTooLarge)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, int64(rule.MaxBodyBytes)+1))
	if err != nil || len(data) > rule.MaxBodyBytes {
		http.Error(w, "API request exceeds its reviewed body limit", http.StatusRequestEntityTooLarge)
		return
	}
	if g.count.Add(1) > int32(g.requests.MaxRequests) {
		http.Error(w, "API attempt exceeds its reviewed request limit", http.StatusTooManyRequests)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	request := r.Clone(ctx)
	request.URL, _ = url.Parse(g.origin)
	request.URL.Path, request.URL.RawQuery = r.URL.Path, r.URL.RawQuery
	request.Host, request.RequestURI = request.URL.Host, ""
	request.Body = io.NopCloser(bytes.NewReader(data))
	request.ContentLength = int64(len(data))
	request.TransferEncoding = nil
	request.Header.Del("Content-Length")
	request.Header.Set("Accept-Encoding", "identity")
	stripHopHeaders(request.Header)
	if g.credential != nil {
		request.Header.Set(g.credential.Header, g.credential.Value)
	}
	response, err := g.transport.RoundTrip(request)
	if err != nil {
		http.Error(w, "Selected API target is unreachable or fails TLS validation", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if encoding := strings.TrimSpace(response.Header.Get("Content-Encoding")); response.StatusCode == http.StatusSwitchingProtocols || encoding != "" && !strings.EqualFold(encoding, "identity") {
		http.Error(w, "API response encoding or protocol is outside the reviewed policy", http.StatusBadGateway)
		return
	}
	data, err = io.ReadAll(io.LimitReader(response.Body, int64(g.requests.MaxResponseBytes)+1))
	if err != nil || len(data) > g.requests.MaxResponseBytes {
		http.Error(w, "API response exceeds its reviewed limit or cannot be read", http.StatusBadGateway)
		return
	}
	stripHopHeaders(response.Header)
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		response.Header.Del("Location")
		response.Header.Set("Content-Type", "text/plain; charset=utf-8")
		data = []byte("Automatic API redirects are disabled by the reviewed request policy.\n")
	}
	response.Header.Del("Content-Length")
	if g.private != nil {
		data = []byte(g.private.Replace(string(data)))
		if len(data) > g.requests.MaxResponseBytes {
			http.Error(w, "Redacted API response exceeds its reviewed limit", http.StatusBadGateway)
			return
		}
	}
	for key, values := range response.Header {
		if g.private != nil && g.private.Replace(key) != key {
			continue
		}
		for _, value := range values {
			if g.private != nil {
				value = g.private.Replace(value)
			}
			w.Header().Add(key, value)
		}
	}
	if r.Method == http.MethodHead && response.ContentLength >= 0 && (response.StatusCode < 300 || response.StatusCode >= 400) {
		w.Header().Set("Content-Length", strconv.FormatInt(response.ContentLength, 10))
	} else {
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	}
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(data)
}
