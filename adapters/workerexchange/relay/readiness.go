package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"time"
)

func probeRequest(r *http.Request) bool {
	return r.Method == http.MethodGet && r.URL.RawQuery == "" && !r.URL.ForceQuery && !r.URL.IsAbs() && r.URL.Opaque == "" && r.URL.RawPath == "" && r.ContentLength == 0 && len(r.TransferEncoding) == 0 && len(r.Trailer) == 0
}

func (g *Relay) serveReadiness(w http.ResponseWriter, r *http.Request) {
	for _, header := range []string{"Authorization", "Cookie", "Origin", "X-TestStudio-Worker-Instance"} {
		if len(r.Header.Values(header)) != 0 {
			problem(w, http.StatusNotFound)
			return
		}
	}
	select {
	case g.probeSlot <- struct{}{}:
		defer func() { <-g.probeSlot }()
	default:
		problem(w, http.StatusServiceUnavailable)
		return
	}
	if !g.backendReady(r.Context()) {
		problem(w, http.StatusServiceUnavailable)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "up", "service": "teststudio"})
}

func (g *Relay) backendReady(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	target := *g.backend
	target.Path = "/healthz"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false
	}
	request.Close = true
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "TestStudio-worker-relay-readiness/v1")
	response, err := g.readinessClient.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.StatusCode != http.StatusOK || err != nil || media != "application/json" || response.ContentLength > 1024 || response.TLS == nil || len(response.TLS.VerifiedChains) == 0 || len(response.TLS.PeerCertificates) == 0 || !response.TLS.PeerCertificates[0].NotAfter.After(time.Now().Add(2*time.Hour+5*time.Minute)) {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1025))
	return err == nil && len(data) <= 1024 && validBackendHealth(data)
}

func validBackendHealth(data []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return false
	}
	fields := map[string]string{}
	for decoder.More() {
		key, err := decoder.Token()
		name, ok := key.(string)
		if err != nil || !ok || name != "status" && name != "service" && name != "role" || fields[name] != "" {
			return false
		}
		var value string
		if decoder.Decode(&value) != nil || value == "" {
			return false
		}
		fields[name] = value
	}
	end, err := decoder.Token()
	return err == nil && end == json.Delim('}') && decoder.Decode(new(any)) == io.EOF && fields["status"] == "up" && fields["service"] == "teststudio" && fields["role"] == "worker"
}
