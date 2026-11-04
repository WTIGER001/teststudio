package azureworker

import (
	"context"
	"time"

	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKDNSReader) Rules(ctx context.Context, id string) ([]d.DNSSecurityRule, error) {
	if id != c.settings.Policy {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.rules.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.rulePath(), "2025-05-01", 2, 2, pager.More, func(ctx context.Context) ([]*d.DNSSecurityRule, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

func (c *SDKDNSReader) Links(ctx context.Context, id string) ([]d.PolicyVirtualNetworkLink, error) {
	if id != c.settings.Policy {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.links.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.linkPath(), "2025-05-01", 1, 2, pager.More, func(ctx context.Context) ([]*d.PolicyVirtualNetworkLink, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

func (c *SDKDNSReader) VNetPolicies(ctx context.Context, id string) ([]d.SubResource, error) {
	if id != c.settings.Firewall.Perimeter.Topology.WorkerVNet {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.policies.NewListByVirtualNetworkPager(group, name, nil)
	page, err := pager.NextPage(ctx)
	if err != nil {
		return nil, sdkError(err, false)
	}
	if len(page.Value) > 1 {
		return nil, hostedexecution.ErrOwnership
	}
	if pager.More() {
		return nil, hostedexecution.ErrUnavailable
	}
	result := []d.SubResource{}
	for _, item := range page.Value {
		if item == nil {
			return nil, hostedexecution.ErrOwnership
		}
		result = append(result, *item)
	}
	return result, nil
}

var _ DNSReader = (*SDKDNSReader)(nil)
