package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type DNSReader interface {
	Policy(context.Context, string) (d.Policy, error)
	DomainList(context.Context, string) (d.DomainList, error)
	Rules(context.Context, string) ([]d.DNSSecurityRule, error)
	Links(context.Context, string) ([]d.PolicyVirtualNetworkLink, error)
	VNetPolicies(context.Context, string) ([]d.SubResource, error)
}

type DNSGuard struct {
	worker   Config
	settings DNSSettings
	firewall *FirewallGuard
	reader   DNSReader
}

func NewDNSGuard(c Config, s DNSSettings, firewall *FirewallGuard, reader DNSReader) (*DNSGuard, error) {
	s, err := s.canonical(c)
	if err != nil || firewall == nil || reader == nil || fingerprint(c) != fingerprint(firewall.worker) || fingerprint(s.Firewall) != fingerprint(firewall.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &DNSGuard{worker: c, settings: s, firewall: firewall, reader: reader}, nil
}

func (g *DNSGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := g.firewall.Check(ctx, c); err != nil {
		return err
	}
	return g.settings.binding(c).check(ctx, c, g.reader)
}

func (s dnsBinding) check(ctx context.Context, c Config, reader DNSReader) error {
	var policy d.Policy
	var allow, block d.DomainList
	var rules []d.DNSSecurityRule
	var links []d.PolicyVirtualNetworkLink
	var attached []d.SubResource
	var errs [6]error
	var group sync.WaitGroup
	group.Go(func() { policy, errs[0] = reader.Policy(ctx, s.Policy) })
	group.Go(func() { allow, errs[1] = reader.DomainList(ctx, s.AllowList) })
	group.Go(func() { block, errs[2] = reader.DomainList(ctx, s.BlockList) })
	group.Go(func() { rules, errs[3] = reader.Rules(ctx, s.Policy) })
	group.Go(func() { links, errs[4] = reader.Links(ctx, s.Policy) })
	group.Go(func() {
		attached, errs[5] = reader.VNetPolicies(ctx, s.VirtualNetwork)
	})
	group.Wait()
	for _, err := range errs {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return hostedexecution.ErrOwnership
		}
	}
	for _, err := range errs {
		if err != nil {
			return hostedexecution.ErrUnavailable
		}
	}
	return s.verify(c.Region, policy, allow, block, rules, links, attached)
}

func (s dnsBinding) verify(region string, policy d.Policy, allow, block d.DomainList, rules []d.DNSSecurityRule, links []d.PolicyVirtualNetworkLink, attached []d.SubResource) error {
	if !dnsIdentity(value(policy.ID), value(policy.Name), value(policy.Type), s.Policy, "Microsoft.Network/dnsResolverPolicies", value(policy.Location), region) || policy.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	if policy.Properties.ProvisioningState == nil || value(policy.Properties.ProvisioningState) != d.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if err := s.verifyDomainList(allow, true, region); err != nil {
		return err
	}
	if err := s.verifyDomainList(block, false, region); err != nil {
		return err
	}
	if err := s.verifyRules(rules, region); err != nil {
		return err
	}
	return s.verifyLinks(links, attached, region)
}
