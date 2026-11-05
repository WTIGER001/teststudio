package azureworker

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKReadIdentityReader) FederationAbsent(ctx context.Context, id string) error {
	if id != c.settings.Identity {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.federation.NewListPager(group, name, nil)
	page, err := pager.NextPage(ctx)
	if err != nil {
		return sdkError(err, false)
	}
	if page.Value == nil {
		return hostedexecution.ErrUnavailable
	}
	for _, credential := range page.Value {
		if credential != nil {
			return hostedexecution.ErrOwnership
		}
	}
	if len(page.Value) != 0 || value(page.NextLink) != "" {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
