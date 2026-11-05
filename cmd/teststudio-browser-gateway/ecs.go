package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/ecsidentity"
)

var metadataPath = regexp.MustCompile(`^/v4/[A-Za-z0-9_-]{1,256}$`)

func ecsGatewayLookup(ctx context.Context, lookup func(string) (string, bool), transport http.RoundTripper) (func(string) (string, bool), error) {
	if _, found := lookup("TESTSTUDIO_GATEWAY_INSTANCE"); found {
		return nil, ecsidentity.ErrIdentity
	}
	raw, found := lookup("ECS_CONTAINER_METADATA_URI_V4")
	location, err := url.Parse(raw)
	if !found || err != nil || location.Scheme != "http" || location.Host != "169.254.170.2" || location.User != nil || location.RawPath != "" || !metadataPath.MatchString(location.Path) || location.RawQuery != "" || location.ForceQuery || location.Fragment != "" || location.Opaque != "" || location.String() != raw {
		return nil, ecsidentity.ErrIdentity
	}
	if transport == nil {
		transport = &http.Transport{ResponseHeaderTimeout: 2 * time.Second, IdleConnTimeout: time.Second}
	}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	location.Path += "/task"
	request, err := http.NewRequestWithContext(bounded, http.MethodGet, location.String(), nil)
	if err != nil {
		return nil, ecsidentity.ErrIdentity
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, ecsidentity.ErrIdentity
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > 64<<10 {
		return nil, ecsidentity.ErrIdentity
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(data) > 64<<10 {
		return nil, ecsidentity.ErrIdentity
	}
	var document struct{ TaskARN string }
	if json.Unmarshal(data, &document) != nil {
		return nil, ecsidentity.ErrIdentity
	}
	instance, err := ecsidentity.TaskInstance(document.TaskARN)
	if err != nil {
		return nil, err
	}
	return func(name string) (string, bool) {
		if name == "TESTSTUDIO_GATEWAY_INSTANCE" {
			return instance, true
		}
		return lookup(name)
	}, nil
}

func fetchECSRemoteConfiguration(ctx context.Context, origin string, lookup func(string) (string, bool)) (browsergateway.Configuration, error) {
	bound, err := ecsGatewayLookup(ctx, lookup, nil)
	if err != nil {
		return browsergateway.Configuration{}, err
	}
	return fetchRemoteConfiguration(ctx, origin, bound)
}
