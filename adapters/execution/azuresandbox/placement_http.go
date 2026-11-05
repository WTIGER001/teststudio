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
	pages   map[string]bool
	posts   map[string]bool
	closed  atomic.Bool
}

type metadataReader struct {
	client *arm.Client
	http   *placementHTTP
}

func placementVersions(s Settings) map[string]string {
	c, p := s.Worker, s.Placement
	return map[string]string{c.groupID(): APIVersion, c.connectionID(): APIVersion, p.Subnet: "2025-03-01", s.PullIdentity: "2024-11-30", p.Registry: "2025-11-01", p.PullAssignment: "2022-04-01", "/subscriptions/" + c.Subscription + pullRole: "2022-05-01-preview"}
}

func newPlacementHTTP(s Settings, base policy.Transporter) *placementHTTP {
	return newMetadataHTTP(placementVersions(s), base)
}

func newMetadataHTTP(versions map[string]string, base policy.Transporter) *placementHTTP {
	allowed := map[string]string{}
	for id, version := range versions {
		allowed[id] = version
	}
	h := &placementHTTP{allowed: allowed, base: base}
	if base == nil {
		h.owned = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}
		h.base = &http.Client{Timeout: 5 * time.Second, Transport: h.owned, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return h
}

func (h *placementHTTP) accepts(r *http.Request) bool {
	u := r.URL
	method := http.MethodGet
	if u != nil && h.posts[u.Path] {
		method = http.MethodPost
	}
	if u == nil || r.Method != method || r.ContentLength > 0 || r.Body != nil && r.Body != http.NoBody || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || r.Host != "" && r.Host != u.Host || h.allowed[u.Path] == "" || !h.queryAllowed(u) {
		return false
	}
	parts := strings.Split(u.Path, "/")
	escaped := u.Path
	if len(parts) > 4 && parts[3] == "resourceGroups" {
		escaped = strings.Replace(u.Path, "/resourceGroups/"+parts[4]+"/", "/resourceGroups/"+url.PathEscape(parts[4])+"/", 1)
	}
	if u.EscapedPath() != escaped {
		return false
	}
	return true
}

func (h *placementHTTP) Do(r *http.Request) (*http.Response, error) {
	if h.closed.Load() {
		return nil, hostedexecution.ErrUnavailable
	}
	if !h.accepts(r) {
		return nil, hostedexecution.ErrOwnership
	}
	return h.readBounded(r)
}

func (h *placementHTTP) readBounded(r *http.Request) (*http.Response, error) {
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

func newMetadataReader(versions map[string]string, credential azcore.TokenCredential, base policy.Transporter) (*metadataReader, error) {
	h := newMetadataHTTP(versions, base)
	options := &arm.ClientOptions{DisableRPRegistration: true, ClientOptions: policy.ClientOptions{Transport: h, Retry: policy.RetryOptions{MaxRetries: -1}, Cloud: cloud.Configuration{ActiveDirectoryAuthorityHost: "https://login.microsoftonline.com/", Services: map[cloud.ServiceName]cloud.ServiceConfiguration{cloud.ResourceManager: {Endpoint: "https://management.azure.com/", Audience: "https://management.core.windows.net/"}}}}}
	client, err := arm.NewClient("github.com/wtiger001/teststudio/azuresandbox", "v1.0.0", credential, options)
	if err != nil {
		h.Close()
		return nil, hostedexecution.ErrConfiguration
	}
	return &metadataReader{client: client, http: h}, nil
}

func (c *metadataReader) get(ctx context.Context, id string) (placementResource, error) {
	var value placementResource
	err := c.documentID(ctx, id, &value)
	return value, err
}

func (c *metadataReader) documentID(ctx context.Context, id string, value any) error {
	version := c.http.allowed[id]
	if version == "" {
		return hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || c.http.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	parts := strings.Split(id, "/")
	escaped := id
	if len(parts) > 4 && parts[3] == "resourceGroups" {
		escaped = strings.Replace(id, "/resourceGroups/"+parts[4]+"/", "/resourceGroups/"+url.PathEscape(parts[4])+"/", 1)
	}
	return c.document(ctx, "https://management.azure.com"+escaped+"?api-version="+version, value)
}

func (c *metadataReader) document(ctx context.Context, target string, value any) error {
	if ctx.Err() != nil || c.http.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return hostedexecution.ErrOwnership
	}
	method := http.MethodGet
	if c.http.posts[parsed.Path] {
		method = http.MethodPost
	}
	request, err := runtime.NewRequest(ctx, method, target)
	if err != nil || !c.http.accepts(request.Raw()) {
		return hostedexecution.ErrOwnership
	}
	response, err := c.client.Pipeline().Do(request)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return hostedexecution.ErrUnavailable
	}
	data, err := io.ReadAll(response.Body)
	if err != nil || json.Unmarshal(data, value) != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
