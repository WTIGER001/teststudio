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
	var policy d.Policy
	var allow, block d.DomainList
	var rules []d.DNSSecurityRule
	var links []d.PolicyVirtualNetworkLink
	var attached []d.SubResource
	var errs [6]error
	var group sync.WaitGroup
	group.Go(func() { policy, errs[0] = g.reader.Policy(ctx, g.settings.Policy) })
	group.Go(func() { allow, errs[1] = g.reader.DomainList(ctx, g.settings.AllowList) })
	group.Go(func() { block, errs[2] = g.reader.DomainList(ctx, g.settings.BlockList) })
	group.Go(func() { rules, errs[3] = g.reader.Rules(ctx, g.settings.Policy) })
	group.Go(func() { links, errs[4] = g.reader.Links(ctx, g.settings.Policy) })
	group.Go(func() {
		attached, errs[5] = g.reader.VNetPolicies(ctx, g.settings.Firewall.Perimeter.Topology.WorkerVNet)
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
	if !dnsIdentity(value(policy.ID), value(policy.Name), value(policy.Type), g.settings.Policy, "Microsoft.Network/dnsResolverPolicies", value(policy.Location), c.Region) || policy.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	if policy.Properties.ProvisioningState == nil || value(policy.Properties.ProvisioningState) != d.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if err := g.verifyDomainList(allow, true); err != nil {
		return err
	}
	if err := g.verifyDomainList(block, false); err != nil {
		return err
	}
	if err := g.verifyRules(rules); err != nil {
		return err
	}
	return g.verifyLinks(links, attached)
}
