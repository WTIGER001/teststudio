package azureworker

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type topologyHTTP struct {
	base      policy.Transporter
	resources map[string]string
}

func newTopologyHTTP(c Config, s TopologySettings, base policy.Transporter) *topologyHTTP {
	if base == nil {
		base = newSDKHTTP(c, nil).base
	}
	return &topologyHTTP{base: base, resources: map[string]string{c.WorkerEnvironment: "2025-07-01", c.ControllerEnvironment: "2025-07-01", s.WorkerVNet: "2025-03-01", s.ControllerVNet: "2025-03-01", s.WorkerSubnet: "2025-03-01", s.ControllerSubnet: "2025-03-01"}}
}

func (h *topologyHTTP) allows(id, version string) bool {
	return version != "" && h.resources[id] == version
}

func (h *topologyHTTP) Do(r *http.Request) (*http.Response, error) {
	u := r.URL
	if u == nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || r.Method != "GET" {
		return nil, hostedexecution.ErrOwnership
	}
	version, ok := h.resources[u.Path]
	if !ok || u.RawQuery != "api-version="+version {
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

func (h *topologyHTTP) CloseIdleConnections() {
	if closer, ok := h.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
