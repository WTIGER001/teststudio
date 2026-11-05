package azuresandbox

import (
	"context"
	"net/url"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (h *placementHTTP) queryAllowed(u *url.URL) bool {
	version := h.allowed[u.Path]
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
	if token == "" || len(token) > 2048 {
		return false
	}
	for _, r := range token {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func (c *metadataReader) inventory(ctx context.Context, path string, limit int) ([]placementResource, error) {
	if !c.http.pages[path] || c.http.allowed[path] == "" {
		return nil, hostedexecution.ErrOwnership
	}
	escaped := path
	parts := strings.Split(path, "/")
	if len(parts) > 4 && parts[3] == "resourceGroups" {
		escaped = strings.Replace(path, "/resourceGroups/"+parts[4]+"/", "/resourceGroups/"+url.PathEscape(parts[4])+"/", 1)
	}
	target := "https://management.azure.com" + escaped + "?api-version=" + c.http.allowed[path]
	result := []placementResource{}
	seen, tokens := map[string]bool{}, map[string]bool{}
	for count := 0; target != "" && count < 3; count++ {
		u, err := url.Parse(target)
		if err != nil || len(target) > 8192 || u.Path != path {
			return nil, hostedexecution.ErrOwnership
		}
		if count > 0 {
			token := u.Query().Get("$skipToken")
			if token == "" || tokens[token] {
				return nil, hostedexecution.ErrOwnership
			}
			tokens[token] = true
		}
		var page struct {
			Value    *[]placementResource `json:"value"`
			NextLink string               `json:"nextLink"`
		}
		if err := c.document(ctx, target, &page); err != nil {
			return nil, err
		}
		if page.Value == nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, entry := range *page.Value {
			id := strings.ToLower(entry.ID)
			if id == "" || seen[id] || len(result) >= limit {
				return nil, hostedexecution.ErrOwnership
			}
			seen[id] = true
			result = append(result, entry)
		}
		target = page.NextLink
	}
	if target != "" {
		return nil, hostedexecution.ErrUnavailable
	}
	return result, nil
}
