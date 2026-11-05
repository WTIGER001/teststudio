package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *Client) Inventory(ctx context.Context) ([]Snapshot, error) {
	values, err := c.pages(ctx, "/sandboxes")
	if err != nil {
		return nil, err
	}
	result := []Snapshot{}
	seen := map[string]bool{}
	for _, item := range values {
		actual, err := snapshot(item)
		if err != nil || seen[actual.ID] {
			return nil, hostedexecution.ErrUnavailable
		}
		seen[actual.ID] = true
		result = append(result, actual)
	}
	return result, nil
}

func (c *Client) Images(ctx context.Context) ([]DiskImage, error) {
	values, err := c.pages(ctx, "/diskimages")
	if err != nil {
		return nil, err
	}
	result := []DiskImage{}
	seen := map[string]bool{}
	for _, item := range values {
		actual, err := c.image(item)
		if err != nil || seen[actual.ID] {
			return nil, hostedexecution.ErrUnavailable
		}
		seen[actual.ID] = true
		result = append(result, actual)
	}
	return result, nil
}

func (c *Client) pages(ctx context.Context, path string) ([]json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	target := c.target(path)
	seenPages := map[string]bool{}
	result := []json.RawMessage{}
	for pages := 0; target != "" && pages < 8; pages++ {
		if !c.validContinuation(target, path) || seenPages[target] {
			return nil, hostedexecution.ErrUnavailable
		}
		seenPages[target] = true
		data, err := c.request(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		var page sandboxPage
		if strings.HasPrefix(strings.TrimSpace(string(data)), "[") {
			err = json.Unmarshal(data, &page.Value)
		} else {
			err = json.Unmarshal(data, &page)
			var required map[string]json.RawMessage
			if json.Unmarshal(data, &required) != nil || required["value"] == nil || string(required["value"]) == "null" {
				err = hostedexecution.ErrUnavailable
			}
		}
		if err != nil || len(page.Value) > 500 || len(result)+len(page.Value) > 1000 {
			return nil, hostedexecution.ErrUnavailable
		}
		result = append(result, page.Value...)
		target = page.NextLink
	}
	if target != "" {
		return nil, hostedexecution.ErrUnavailable
	}
	return result, nil
}

func (c *Client) validContinuation(target, path string) bool {
	u, err := url.Parse(target)
	if err != nil || len(target) > 4096 || u.Scheme != "https" || "https://"+u.Host != c.config.endpoint() || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.EscapedPath() != c.config.path()+path || u.RawPath != "" && u.RawPath != u.EscapedPath() {
		return false
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query["api-version"]) != 1 || query.Get("api-version") != APIVersion {
		return false
	}
	for key, values := range query {
		if len(values) != 1 || values[0] == "" {
			return false
		}
		switch key {
		case "api-version", "$skiptoken", "skipToken", "continuationToken", "continuation", "$skip", "$top":
		default:
			return false
		}
	}
	return true
}
