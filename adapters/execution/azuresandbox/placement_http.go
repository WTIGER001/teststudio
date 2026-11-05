package azuresandbox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type placementHTTP struct {
	base    policy.Transporter
	owned   *http.Transport
	allowed map[string]string
	closed  atomic.Bool
}

func placementVersions(s Settings) map[string]string {
	c, p := s.Worker, s.Placement
	return map[string]string{c.groupID(): APIVersion, c.connectionID(): APIVersion, p.Subnet: "2025-03-01", s.PullIdentity: "2024-11-30", p.Registry: "2025-11-01", p.PullAssignment: "2022-04-01", "/subscriptions/" + c.Subscription + pullRole: "2022-05-01-preview"}
}

func newPlacementHTTP(s Settings, base policy.Transporter) *placementHTTP {
	h := &placementHTTP{allowed: placementVersions(s), base: base}
	if base == nil {
		h.owned = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}
		h.base = &http.Client{Timeout: 5 * time.Second, Transport: h.owned, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return h
}

func (h *placementHTTP) Do(r *http.Request) (*http.Response, error) {
	u := r.URL
	if h.closed.Load() {
		return nil, hostedexecution.ErrUnavailable
	}
	if u == nil || r.Method != "GET" || r.Body != nil && r.Body != http.NoBody || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || h.allowed[u.Path] == "" || u.RawQuery != "api-version="+h.allowed[u.Path] {
		return nil, hostedexecution.ErrOwnership
	}
	parts := strings.Split(u.Path, "/")
	escaped := u.Path
	if len(parts) > 4 && parts[3] == "resourceGroups" {
		escaped = strings.Replace(u.Path, "/resourceGroups/"+parts[4]+"/", "/resourceGroups/"+url.PathEscape(parts[4])+"/", 1)
	}
	if u.EscapedPath() != escaped {
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

func (h *placementHTTP) Close() {
	h.closed.Store(true)
	if h.owned != nil {
		h.owned.CloseIdleConnections()
	}
}

func placementClient(s Settings, credential azcore.TokenCredential, h *placementHTTP) (*arm.Client, error) {
	options := &arm.ClientOptions{DisableRPRegistration: true, ClientOptions: policy.ClientOptions{Transport: h, Retry: policy.RetryOptions{MaxRetries: -1}, Cloud: cloud.Configuration{ActiveDirectoryAuthorityHost: "https://login.microsoftonline.com/", Services: map[cloud.ServiceName]cloud.ServiceConfiguration{cloud.ResourceManager: {Endpoint: "https://management.azure.com/", Audience: "https://management.core.windows.net/"}}}}}
	return arm.NewClient("github.com/wtiger001/teststudio/azuresandbox", "v1.0.0", credential, options)
}

func (c *SDKPlacement) get(ctx context.Context, id string) (placementResource, error) {
	version := c.http.allowed[id]
	if version == "" {
		return placementResource{}, hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || c.http.closed.Load() {
		return placementResource{}, hostedexecution.ErrUnavailable
	}
	parts := strings.Split(id, "/")
	escaped := id
	if len(parts) > 4 && parts[3] == "resourceGroups" {
		escaped = strings.Replace(id, "/resourceGroups/"+parts[4]+"/", "/resourceGroups/"+url.PathEscape(parts[4])+"/", 1)
	}
	request, err := runtime.NewRequest(ctx, http.MethodGet, "https://management.azure.com"+escaped+"?api-version="+version)
	if err != nil {
		return placementResource{}, hostedexecution.ErrConfiguration
	}
	response, err := c.client.Pipeline().Do(request)
	if err != nil {
		return placementResource{}, hostedexecution.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return placementResource{}, hostedexecution.ErrUnavailable
	}
	var value placementResource
	data, err := io.ReadAll(response.Body)
	if err != nil || json.Unmarshal(data, &value) != nil {
		return value, hostedexecution.ErrUnavailable
	}
	return value, nil
}
