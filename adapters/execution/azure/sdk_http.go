package azureworker

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type sdkHTTP struct {
	base          policy.Transporter
	prefix        string
	escapedPrefix string
	name          *regexp.Regexp
}

func newSDKHTTP(c Config, base policy.Transporter) *sdkHTTP {
	if base == nil {
		base = &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}}
	}
	prefix := c.resourceID("Microsoft.App", "jobs", "")
	escaped := strings.Replace(prefix, "/resourceGroups/"+c.ResourceGroup+"/", "/resourceGroups/"+url.PathEscape(c.ResourceGroup)+"/", 1)
	return &sdkHTTP{base: base, prefix: prefix, escapedPrefix: escaped, name: regexp.MustCompile("^" + regexp.QuoteMeta(c.NamePrefix) + "-[a-f0-9]{24}$")}
}

func (c *sdkHTTP) Do(r *http.Request) (*http.Response, error) {
	u := r.URL
	if u == nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || !strings.HasPrefix(u.Path, c.prefix) || u.Query().Get("api-version") != "2025-07-01" {
		return nil, hostedexecution.ErrOwnership
	}
	if u.RawPath != "" && u.EscapedPath() != c.escapedPrefix+strings.TrimPrefix(u.Path, c.prefix) {
		return nil, hostedexecution.ErrOwnership
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, c.prefix), "/")
	if !c.name.MatchString(parts[0]) {
		return nil, hostedexecution.ErrOwnership
	}
	allowed := len(parts) == 1 && (r.Method == "GET" || r.Method == "PUT" || r.Method == "DELETE") || len(parts) == 2 && (parts[1] == "start" && r.Method == "POST" || parts[1] == "executions" && r.Method == "GET") || len(parts) == 4 && parts[1] == "executions" && executionName.MatchString(parts[2]) && parts[3] == "stop" && r.Method == "POST"
	if !allowed {
		return nil, hostedexecution.ErrOwnership
	}
	response, err := c.base.Do(r)
	if err != nil || response == nil || response.Body == nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return nil, hostedexecution.ErrUnavailable
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 || response.ContentLength > 8<<20 {
		response.Body.Close()
		return nil, hostedexecution.ErrUnavailable
	}
	response.Body = &sdkBody{Reader: io.LimitReader(response.Body, 8<<20), Closer: response.Body}
	return response, nil
}

type sdkBody struct {
	io.Reader
	io.Closer
}

func (c *sdkHTTP) CloseIdleConnections() {
	if closer, ok := c.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
