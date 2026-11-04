package azureworker

import (
	"context"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKFirewallReader) RuleGroups(ctx context.Context, id string) ([]n.FirewallPolicyRuleCollectionGroup, error) {
	if id != c.settings.Policy {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.groups.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, id+"/ruleCollectionGroups", "2025-03-01", 4, 4, pager.More, func(ctx context.Context) ([]*n.FirewallPolicyRuleCollectionGroup, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}
