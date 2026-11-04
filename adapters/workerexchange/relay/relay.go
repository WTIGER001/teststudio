package relay

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/workerwire"
)

var errConfiguration = errors.New("worker relay requires explicit HTTPS public and backend origins")

type Relay struct {
	public, backend *url.URL
	client          *http.Client
	slots           chan struct{}
}

func New(publicOrigin, backendOrigin string, transport http.RoundTripper) (*Relay, error) {
	public, err := origin(publicOrigin)
	if err != nil {
		return nil, err
	}
	backend, err := origin(backendOrigin)
	if err != nil || authority(public) == authority(backend) {
		return nil, errConfiguration
	}
	if transport == nil {
		transport = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 15 * time.Second, IdleConnTimeout: 30 * time.Second, MaxConnsPerHost: 8, MaxIdleConnsPerHost: 8, MaxIdleConns: 8, DisableCompression: true}
	}
	return &Relay{public: public, backend: backend, slots: make(chan struct{}, 8), client: &http.Client{Transport: transport, Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (g *Relay) Close() { g.client.CloseIdleConnections() }

func (g *Relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.Method == http.MethodGet && (r.URL.Path == "/healthz" || r.URL.Path == "/livez") && r.URL.RawQuery == "" && !r.URL.ForceQuery && !r.URL.IsAbs() && r.URL.RawPath == "" && r.ContentLength == 0 {
		json.NewEncoder(w).Encode(map[string]string{"status": "up", "service": "teststudio"})
		return
	}
	if !strings.EqualFold(r.Host, g.public.Host) {
		problem(w, http.StatusNotFound)
		return
	}
	op, ok := scope(r)
	if !ok {
		problem(w, http.StatusNotFound)
		return
	}
	if r.ContentLength > op.limit || op.input && (r.ContentLength != 0 || len(r.TransferEncoding) != 0) {
		problem(w, http.StatusRequestEntityTooLarge)
		return
	}
	select {
	case g.slots <- struct{}{}:
		defer func() { <-g.slots }()
	default:
		problem(w, http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	var body io.Reader
	if !op.input {
		body = http.MaxBytesReader(w, r.Body, op.limit)
	}
	target := *g.backend
	parsed, _ := url.Parse(op.path)
	target.Path, target.RawPath = parsed.Path, parsed.RawPath
	request, err := http.NewRequestWithContext(ctx, r.Method, target.String(), body)
	if err != nil {
		problem(w, http.StatusServiceUnavailable)
		return
	}
	request.Header.Set("Authorization", "Bearer "+op.token)
	request.Header.Set("X-TestStudio-Worker-Instance", op.instance)
	contentType := "application/json"
	if strings.Contains(op.path, "/attachments/") {
		contentType = "application/octet-stream"
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("User-Agent", "TestStudio-worker-relay/v1")
	if !op.input {
		request.ContentLength = r.ContentLength
	}
	response, err := g.client.Do(request)
	if err != nil {
		problem(w, http.StatusServiceUnavailable)
		return
	}
	defer response.Body.Close()
	if !op.input && response.StatusCode == http.StatusNoContent {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !op.input || response.StatusCode != http.StatusOK {
		problem(w, upstreamStatus(response.StatusCode))
		return
	}
	const limit = workerwire.MaxWorkItemBytes + 8192
	if response.ContentLength > limit {
		problem(w, http.StatusServiceUnavailable)
		return
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(data) > limit {
		problem(w, http.StatusServiceUnavailable)
		return
	}
	var packet workerwire.Packet
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&packet) != nil || decoder.Decode(new(any)) != io.EOF || packet.Execution != op.execution || !time.Now().Before(packet.Deadline) || len(packet.Input) > workerwire.MaxWorkItemBytes || !json.Valid(packet.Input) {
		problem(w, http.StatusServiceUnavailable)
		return
	}
	json.NewEncoder(w).Encode(packet)
}

func upstreamStatus(status int) int {
	switch status {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusInsufficientStorage:
		return status
	default:
		return http.StatusServiceUnavailable
	}
}

func problem(w http.ResponseWriter, status int) {
	message := workerwire.ErrUnavailable.Error()
	switch status {
	case http.StatusNotFound:
		message = workerwire.ErrDenied.Error()
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		message = workerwire.ErrInvalid.Error()
	case http.StatusConflict:
		message = workerwire.ErrConflict.Error()
	case http.StatusInsufficientStorage:
		message = workerwire.ErrQuota.Error()
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
