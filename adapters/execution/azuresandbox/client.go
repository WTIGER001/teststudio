package azuresandbox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type Client struct {
	config     Config
	credential azcore.TokenCredential
	http       *http.Client
	owned      *http.Transport
}

func NewClient(c Config, credential azcore.TokenCredential, transport http.RoundTripper) (*Client, error) {
	if Validate(c) != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	var owned *http.Transport
	if transport == nil {
		owned = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}
		transport = owned
	}
	return &Client{config: c, credential: credential, owned: owned, http: &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Close() {
	if c.owned != nil {
		c.owned.CloseIdleConnections()
	}
}

func (c *Client) request(ctx context.Context, method, target string, input any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	var data []byte
	if input != nil {
		var err error
		data, err = json.Marshal(input)
		if err != nil || len(data) > 64<<10 {
			return nil, hostedexecution.ErrConfiguration
		}
	}
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{TokenScope}})
	if err != nil || ctx.Err() != nil || len(token.Token) == 0 || len(token.Token) > 64<<10 || strings.ContainsAny(token.Token, "\x00\r\n") || !token.ExpiresOn.After(time.Now()) {
		return nil, hostedexecution.ErrUnavailable
	}
	r, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(data))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	r.Header.Set("Authorization", "Bearer "+token.Token)
	r.Header.Set("Accept", "application/json")
	if input != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(r)
	if err != nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return nil, hostedexecution.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if method == http.MethodGet && response.StatusCode == http.StatusNotFound {
			return nil, hostedexecution.ErrNotFound
		}
		if method != http.MethodGet {
			switch response.StatusCode {
			case 400, 401, 403, 404, 405, 413, 415, 422:
				return nil, hostedexecution.ErrRejected
			}
		}
		return nil, hostedexecution.ErrUnavailable
	}
	if response.ContentLength > 4<<20 {
		return nil, hostedexecution.ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(body) > 4<<20 {
		return nil, hostedexecution.ErrUnavailable
	}
	return body, nil
}

func (c *Client) target(path string) string {
	return c.config.endpoint() + c.config.path() + path + "?api-version=" + APIVersion
}
