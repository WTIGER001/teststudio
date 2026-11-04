package oci

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) redirectBlob(ctx context.Context, response *http.Response) (*http.Response, error) {
	for count := 0; response.StatusCode >= 300 && response.StatusCode < 400; count++ {
		location := response.Header.Get("Location")
		response.Body.Close()
		if count >= 3 || len(location) > 8192 || response.StatusCode != http.StatusTemporaryRedirect && response.StatusCode != http.StatusFound && response.StatusCode != http.StatusMovedPermanently && response.StatusCode != http.StatusPermanentRedirect {
			return nil, ErrInvalid
		}
		parsed, err := url.Parse(location)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" || !c.blobRedirectOrigins[parsed.Scheme+"://"+parsed.Host] {
			return nil, ErrDenied
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return nil, ErrInvalid
		}
		response, err = c.http.Do(request)
		if err != nil {
			return nil, ErrUnavailable
		}
	}
	return response, nil
}
