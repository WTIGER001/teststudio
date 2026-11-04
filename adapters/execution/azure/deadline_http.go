package azureworker

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type deadlineHTTP struct {
	base          policy.Transporter
	prefix        string
	escapedPrefix string
	name          *regexp.Regexp
}

func newDeadlineHTTP(c DeadlineConfig, base policy.Transporter) *deadlineHTTP {
	if base == nil {
		base = newSDKHTTP(c.Worker, nil).base
	}
	worker := c.Worker
	worker.ResourceGroup = c.ResourceGroup
	prefix := worker.resourceID("Microsoft.Logic", "workflows", "")
	escaped := strings.Replace(prefix, "/resourceGroups/"+c.ResourceGroup+"/", "/resourceGroups/"+url.PathEscape(c.ResourceGroup)+"/", 1)
	return &deadlineHTTP{base: base, prefix: prefix, escapedPrefix: escaped, name: regexp.MustCompile("^" + regexp.QuoteMeta(c.Worker.NamePrefix) + "-stop-[a-f0-9]{24}$")}
}

func (h *deadlineHTTP) Do(r *http.Request) (*http.Response, error) {
	u := r.URL
	if u == nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || !strings.HasPrefix(u.Path, h.prefix) || u.RawQuery != "api-version=2019-05-01" || !h.name.MatchString(strings.TrimPrefix(u.Path, h.prefix)) || u.RawPath != "" && u.EscapedPath() != h.escapedPrefix+strings.TrimPrefix(u.Path, h.prefix) || r.Method != "GET" && r.Method != "PUT" && r.Method != "DELETE" {
		return nil, hostedexecution.ErrOwnership
	}
	response, err := h.base.Do(r)
	if err != nil || response == nil || response.Body == nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return nil, hostedexecution.ErrUnavailable
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 || response.ContentLength > 1<<20 {
		response.Body.Close()
		return nil, hostedexecution.ErrUnavailable
	}
	response.Body = &sdkBody{Reader: io.LimitReader(response.Body, 1<<20), Closer: response.Body}
	return response, nil
}

func (h *deadlineHTTP) CloseIdleConnections() {
	if closer, ok := h.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
