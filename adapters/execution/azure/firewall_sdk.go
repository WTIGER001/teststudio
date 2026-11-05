package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKFirewallReader struct {
	settings  firewallBinding
	firewalls *n.AzureFirewallsClient
	policies  *n.FirewallPoliciesClient
	groups    *n.FirewallPolicyRuleCollectionGroupsClient
	subnets   *n.SubnetsClient
	public    *n.PublicIPAddressesClient
	http      *armReadHTTP
}

func NewSDKFirewallReader(c Config, s FirewallSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKFirewallReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKFirewallReader(c, s.binding(), credential, transport)
}

func newSDKFirewallReader(c Config, s firewallBinding, credential azcore.TokenCredential, transport policy.Transporter) (*SDKFirewallReader, error) {
	path := s.Policy + "/ruleCollectionGroups"
	h := newARMReadHTTP(c, map[string]string{s.Firewall: "2025-03-01", s.Policy: "2025-03-01", s.PublicIP: "2025-03-01", s.subnet(): "2025-03-01", path: "2025-03-01"}, transport)
	h.pages = map[string]bool{path: true}
	options := publicARMOptions(h)
	firewalls, err := n.NewAzureFirewallsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	policies, err := n.NewFirewallPoliciesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	groups, err := n.NewFirewallPolicyRuleCollectionGroupsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	subnets, err := n.NewSubnetsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	public, err := n.NewPublicIPAddressesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKFirewallReader{settings: s, firewalls: firewalls, policies: policies, groups: groups, subnets: subnets, public: public, http: h}, nil
}

func (c *SDKFirewallReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKFirewallReader) Firewall(ctx context.Context, id string) (n.AzureFirewall, error) {
	if id != c.settings.Firewall {
		return n.AzureFirewall{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.firewalls.Get(ctx, group, name, nil)
	return result.AzureFirewall, sdkError(err, false)
}
func (c *SDKFirewallReader) Policy(ctx context.Context, id string) (n.FirewallPolicy, error) {
	if id != c.settings.Policy {
		return n.FirewallPolicy{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.policies.Get(ctx, group, name, nil)
	return result.FirewallPolicy, sdkError(err, false)
}
func (c *SDKFirewallReader) Subnet(ctx context.Context, id string) (n.Subnet, error) {
	if id != c.settings.subnet() {
		return n.Subnet{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.subnets.Get(ctx, group, name, "AzureFirewallSubnet", nil)
	return result.Subnet, sdkError(err, false)
}
func (c *SDKFirewallReader) PublicIP(ctx context.Context, id string) (n.PublicIPAddress, error) {
	if id != c.settings.PublicIP {
		return n.PublicIPAddress{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.public.Get(ctx, group, name, nil)
	return result.PublicIPAddress, sdkError(err, false)
}

var _ FirewallReader = (*SDKFirewallReader)(nil)
