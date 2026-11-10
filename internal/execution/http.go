package execution

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/projects"
)

type HTTP struct {
	RootCAs *x509.CertPool
	Trust   HTTPTrustResolver
}

func (e *HTTP) Execute(ctx context.Context, project string, test catalog.Test, environment projects.Environment) Result {
	start := time.Now()
	r := Result{TestID: test.ID, Attempt: 1, State: "failed", StartedAt: start.UTC()}
	var finalize func()
	fail := func(kind string, err error) Result {
		r.FailureKind = kind
		if kind == "timeout" {
			r.State = "timed-out"
		}
		r.Error = err.Error()
		r.FinishedAt = time.Now().UTC()
		r.DurationMs = float64(time.Since(start)) / float64(time.Millisecond)
		if r.HTTP != nil {
			r.HTTP.DurationMs = r.DurationMs
		}
		if finalize != nil {
			finalize()
		}
		return r
	}
	resolved, err := catalog.ResolveHTTPParameters(test, environment.Parameters)
	if err != nil {
		return fail("configuration", err)
	}
	test = resolved
	if issues := catalog.ValidateHTTPCheck(test.Check); len(issues) > 0 {
		return fail("configuration", errors.New(issues[0].Message))
	}
	check := test.Check
	base, err := url.Parse(environment.Endpoints[check.Endpoint])
	if err != nil || base.Hostname() == "" || base.User != nil {
		return fail("configuration", errors.New("registered endpoint is invalid or missing"))
	}
	if base.Scheme != "https" && !(base.Scheme == "http" && (environment.Stage == "local" || environment.AllowHTTP)) {
		return fail("configuration", errors.New("plain HTTP requires an explicit permitted environment"))
	}
	roots, bundle, err := e.trust(ctx, project, test, environment)
	if err != nil {
		if ctx.Err() != nil {
			return fail(failureKind(ctx.Err()), ctx.Err())
		}
		return fail("configuration", err)
	}
	relative, _ := url.Parse(check.Path)
	rawPath := strings.TrimRight(base.EscapedPath(), "/") + relative.EscapedPath()
	base.Path = strings.TrimRight(base.Path, "/") + relative.Path
	base.RawPath = rawPath
	base.RawQuery = relative.RawQuery
	base.Fragment = ""
	limit := time.Duration(test.TimeoutSeconds) * time.Second
	if limit <= 0 || limit > time.Hour {
		return fail("configuration", errors.New("test timeout is invalid"))
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	scope := networkguard.Public
	if environment.AllowPrivate || environment.Stage == "local" {
		scope = networkguard.Private
	}
	transport := &http.Transport{DialContext: networkguard.Dial(scope), TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: limit, MaxResponseHeaderBytes: 32 << 10}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	observation := &HTTPObservation{URL: base.Scheme + "://" + base.Host + base.EscapedPath(), Timings: map[string]float64{}, TrustBundle: bundle}
	r.HTTP = observation
	var mu sync.Mutex
	starts := map[string]time.Time{}
	timings := map[string]float64{}
	finalize = func() {
		mu.Lock()
		defer mu.Unlock()
		snapshot := map[string]float64{}
		for key, value := range timings {
			snapshot[key] = value
		}
		observation.Timings = snapshot
	}
	begin := func(name string) { mu.Lock(); starts[name] = time.Now(); mu.Unlock() }
	end := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		if at, ok := starts[name]; ok {
			timings[name] = float64(time.Since(at)) / float64(time.Millisecond)
		}
	}
	trace := &httptrace.ClientTrace{DNSStart: func(httptrace.DNSStartInfo) { begin("dns") }, DNSDone: func(httptrace.DNSDoneInfo) { end("dns") }, ConnectStart: func(string, string) { begin("connect") }, ConnectDone: func(string, string, error) { end("connect") }, TLSHandshakeStart: func() { begin("tls") }, TLSHandshakeDone: func(tls.ConnectionState, error) { end("tls") }, GotFirstResponseByte: func() {
		mu.Lock()
		timings["firstByte"] = float64(time.Since(start)) / float64(time.Millisecond)
		mu.Unlock()
	}}
	request, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), check.Method, base.String(), nil)
	if err != nil {
		return fail("configuration", errors.New("request could not be constructed"))
	}
	request.Header.Set("User-Agent", "TestStudio/1.0 health-check")
	response, err := client.Do(request)
	if err != nil {
		if e, ok := err.(*url.Error); ok {
			err = e.Err
		}
		kind := failureKind(err)
		return fail(kind, err)
	}
	defer response.Body.Close()
	observation.Status = response.StatusCode
	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		expires := response.TLS.PeerCertificates[0].NotAfter
		observation.CertificateExpires = &expires
		if time.Until(expires) < time.Duration(check.MinCertificateDays)*24*time.Hour {
			return fail("tls", errors.New("certificate expires before the configured minimum lifetime"))
		}
	} else if check.MinCertificateDays > 0 {
		return fail("tls", errors.New("TLS certificate information is unavailable for the endpoint"))
	}
	if !slices.Contains(check.StatusCodes, response.StatusCode) {
		return fail("http-status", fmt.Errorf("unexpected HTTP status %d", response.StatusCode))
	}
	for name, expected := range check.Headers {
		if response.Header.Get(name) != expected {
			return fail("assertion", fmt.Errorf("response header %s did not match", name))
		}
	}
	if check.RequiredText != "" || check.ForbiddenText != "" || len(check.JSON) > 0 {
		body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		if err != nil {
			return fail(failureKind(err), err)
		}
		if len(body) > 1<<20 {
			return fail("assertion", errors.New("response exceeds the bounded assertion size"))
		}
		if err := assertBody(body, check.RequiredText, check.ForbiddenText, check.JSON); err != nil {
			return fail("assertion", err)
		}
	}
	observation.DurationMs = float64(time.Since(start)) / float64(time.Millisecond)
	if check.MaxResponseMs > 0 && observation.DurationMs > float64(check.MaxResponseMs) {
		return fail("response-time", errors.New("response exceeds the configured time limit"))
	}
	finalize()
	r.State = "passed"
	r.FinishedAt = time.Now().UTC()
	r.DurationMs = float64(time.Since(start)) / float64(time.Millisecond)
	return r
}
