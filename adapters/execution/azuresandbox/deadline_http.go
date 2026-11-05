package azuresandbox

import (
	"bytes"
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

type deadlineHTTP struct {
	base   policy.Transporter
	config DeadlineConfig
	name   *regexp.Regexp
}

func newDeadlineHTTP(c DeadlineConfig, transport policy.Transporter) *deadlineHTTP {
	if transport == nil {
		transport = &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{
			Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}}
	}
	return &deadlineHTTP{base: transport, config: c, name: regexp.MustCompile("^" + regexp.QuoteMeta(c.NamePrefix) + "-stop-[a-f0-9]{24}$")}
}

func (h *deadlineHTTP) Do(r *http.Request) (*http.Response, error) {
	c, u := h.config, r.URL
	prefix := "/subscriptions/" + c.Worker.Subscription + "/resourceGroups/" + c.ResourceGroup + "/providers/Microsoft.Logic/workflows/"
	escaped := "/subscriptions/" + c.Worker.Subscription + "/resourceGroups/" + url.PathEscape(c.ResourceGroup) + "/providers/Microsoft.Logic/workflows/"
	if u == nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || !strings.HasPrefix(u.Path, prefix) || !h.name.MatchString(strings.TrimPrefix(u.Path, prefix)) || u.EscapedPath() != escaped+strings.TrimPrefix(u.Path, prefix) || u.RawQuery != "api-version=2019-05-01" || r.Method != "GET" && r.Method != "PUT" && r.Method != "DELETE" {
		return nil, hostedexecution.ErrOwnership
	}
	response, err := h.base.Do(r)
	if err != nil || response == nil || response.Body == nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return nil, hostedexecution.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode < 400 || response.ContentLength > 1<<20 {
		return nil, hostedexecution.ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, hostedexecution.ErrUnavailable
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	response.ContentLength = int64(len(data))
	return response, nil
}

func (h *deadlineHTTP) Close() {
	if closer, ok := h.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
