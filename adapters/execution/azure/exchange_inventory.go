package azureworker

import (
	"context"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKExchangeReader) Connections(ctx context.Context, id string) ([]a.PrivateEndpointConnection, error) {
	if id != c.settings.environment() {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.connections.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.connections(), exchangeVersion, 1, 2, pager.More, func(ctx context.Context) ([]*a.PrivateEndpointConnection, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}
func (c *SDKExchangeReader) Links(ctx context.Context, id string) ([]a.PrivateLinkResource, error) {
	if id != c.settings.environment() {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.links.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.links(), exchangeVersion, 1, 2, pager.More, func(ctx context.Context) ([]*a.PrivateLinkResource, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

var _ ExchangeReader = (*SDKExchangeReader)(nil)
