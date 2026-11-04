package azureworker

import (
	"context"
	"net/url"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func armInventory[T any](ctx context.Context, h *armReadHTTP, path, version string, maxItems, maxPages int, more func() bool, fetch func(context.Context) ([]*T, *string, error)) ([]T, error) {
	seen := map[string]bool{}
	result := []T{}
	for count := 0; more() && count < maxPages; count++ {
		items, next, err := fetch(ctx)
		if err != nil {
			return nil, sdkError(err, false)
		}
		for _, item := range items {
			if item == nil {
				return nil, hostedexecution.ErrOwnership
			}
			result = append(result, *item)
			if len(result) > maxItems {
				return nil, hostedexecution.ErrOwnership
			}
		}
		if value(next) != "" {
			u, err := url.Parse(value(next))
			if err != nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.Path != path || !h.queryAllowed(u, version) {
				return nil, hostedexecution.ErrOwnership
			}
			token := u.Query().Get("$skipToken")
			if token == "" || seen[token] {
				return nil, hostedexecution.ErrOwnership
			}
			seen[token] = true
		}
	}
	if more() {
		return nil, hostedexecution.ErrUnavailable
	}
	return result, nil
}
