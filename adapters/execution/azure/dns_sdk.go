package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKDNSReader struct {
	settings dnsBinding
	policies *d.PoliciesClient
	lists    *d.DomainListsClient
	rules    *d.DNSSecurityRulesClient
	links    *d.PolicyVirtualNetworkLinksClient
	http     *armReadHTTP
}

func NewSDKDNSReader(c Config, s DNSSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKDNSReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKDNSReader(c, s.binding(c), credential, transport)
}

func newSDKDNSReader(c Config, s dnsBinding, credential azcore.TokenCredential, transport policy.Transporter) (*SDKDNSReader, error) {
	resources := map[string]string{}
	for _, id := range []string{s.Policy, s.AllowList, s.BlockList, s.rulePath(), s.linkPath(), s.vnetPolicyPath()} {
		resources[id] = "2025-05-01"
	}
	h := newARMReadHTTP(c, resources, transport)
	h.pages = map[string]bool{s.rulePath(): true, s.linkPath(): true}
	h.posts = map[string]bool{s.vnetPolicyPath(): true}
	options := publicARMOptions(h)
	policies, err := d.NewPoliciesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	lists, err := d.NewDomainListsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	rules, err := d.NewDNSSecurityRulesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	links, err := d.NewPolicyVirtualNetworkLinksClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKDNSReader{settings: s, policies: policies, lists: lists, rules: rules, links: links, http: h}, nil
}

func (c *SDKDNSReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKDNSReader) Policy(ctx context.Context, id string) (d.Policy, error) {
	if id != c.settings.Policy {
		return d.Policy{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.policies.Get(ctx, group, name, nil)
	return result.Policy, sdkError(err, false)
}

func (c *SDKDNSReader) DomainList(ctx context.Context, id string) (d.DomainList, error) {
	if id != c.settings.AllowList && id != c.settings.BlockList {
		return d.DomainList{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.lists.Get(ctx, group, name, nil)
	return result.DomainList, sdkError(err, false)
}
