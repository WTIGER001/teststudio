package azureworker

import (
	"context"
	"net/url"
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
	seen := map[string]bool{}
	result := []n.FirewallPolicyRuleCollectionGroup{}
	for pageCount := 0; pager.More() && pageCount < 4; pageCount++ {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, sdkError(err, false)
		}
		for _, item := range page.Value {
			if item == nil {
				return nil, hostedexecution.ErrOwnership
			}
			result = append(result, *item)
			if len(result) > 4 {
				return nil, hostedexecution.ErrOwnership
			}
		}
		if next := value(page.NextLink); next != "" {
			u, err := url.Parse(next)
			if err != nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.Path != id+"/ruleCollectionGroups" || !c.http.queryAllowed(u, "2025-03-01") {
				return nil, hostedexecution.ErrOwnership
			}
			token := u.Query().Get("$skipToken")
			if token == "" || seen[token] {
				return nil, hostedexecution.ErrOwnership
			}
			seen[token] = true
		}
	}
	if pager.More() {
		return nil, hostedexecution.ErrUnavailable
	}
	return result, nil
}
