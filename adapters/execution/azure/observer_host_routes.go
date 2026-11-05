package azureworker

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKObserverHostReader) RoutesAbsent(ctx context.Context, environment string) error {
	if environment != c.service.environment {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(environment)
	pager := c.service.routes.NewListPager(group, name, nil)
	page, err := pager.NextPage(ctx)
	if err != nil {
		if controlError(err) == hostedexecution.ErrOwnership {
			return hostedexecution.ErrOwnership
		}
		return hostedexecution.ErrUnavailable
	}
	if page.Value == nil {
		return hostedexecution.ErrUnavailable
	}
	for _, route := range page.Value {
		if route != nil {
			return hostedexecution.ErrOwnership
		}
	}
	if len(page.Value) != 0 || value(page.NextLink) != "" {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
