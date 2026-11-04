package azureworker

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type armReadHTTP struct {
	base      policy.Transporter
	resources map[string]string
	pages     map[string]bool
}

func newTopologyHTTP(c Config, s TopologySettings, base policy.Transporter) *armReadHTTP {
	return newARMReadHTTP(c, map[string]string{c.WorkerEnvironment: "2025-07-01", c.ControllerEnvironment: "2025-07-01", s.WorkerVNet: "2025-03-01", s.ControllerVNet: "2025-03-01", s.WorkerSubnet: "2025-03-01", s.ControllerSubnet: "2025-03-01"}, base)
}

func newARMReadHTTP(c Config, resources map[string]string, base policy.Transporter) *armReadHTTP {
	if base == nil {
		base = newSDKHTTP(c, nil).base
	}
	owned := make(map[string]string, len(resources))
	for id, version := range resources {
		owned[id] = version
	}
	return &armReadHTTP{base: base, resources: owned}
}

func (h *armReadHTTP) allows(id, version string) bool {
	return version != "" && h.resources[id] == version
}

func (h *armReadHTTP) Do(r *http.Request) (*http.Response, error) {
	u := r.URL
	if u == nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || r.Method != "GET" {
		return nil, hostedexecution.ErrOwnership
	}
	version, ok := h.resources[u.Path]
	if !ok || !h.queryAllowed(u, version) {
		return nil, hostedexecution.ErrOwnership
	}
	group, _ := resourceParts(u.Path)
	escaped := strings.Replace(u.Path, "/resourceGroups/"+group+"/", "/resourceGroups/"+url.PathEscape(group)+"/", 1)
	if u.RawPath != "" && u.EscapedPath() != escaped {
		return nil, hostedexecution.ErrOwnership
	}
	response, err := h.base.Do(r)
	if err != nil || response == nil || response.Body == nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return nil, hostedexecution.ErrUnavailable
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 || response.ContentLength > 4<<20 {
		response.Body.Close()
		return nil, hostedexecution.ErrUnavailable
	}
	response.Body = &sdkBody{Reader: io.LimitReader(response.Body, 4<<20), Closer: response.Body}
	return response, nil
}

func (h *armReadHTTP) CloseIdleConnections() {
	if closer, ok := h.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (h *armReadHTTP) queryAllowed(u *url.URL, version string) bool {
	if u.RawQuery == "api-version="+version {
		return true
	}
	if !h.pages[u.Path] || len(u.RawQuery) > 4096 {
		return false
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query) != 2 || len(query["api-version"]) != 1 || query.Get("api-version") != version || len(query["$skipToken"]) != 1 {
		return false
	}
	token := query.Get("$skipToken")
	if len(token) == 0 || len(token) > 2048 {
		return false
	}
	for _, r := range token {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
