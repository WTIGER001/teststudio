package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type FirewallReader interface {
	Firewall(context.Context, string) (n.AzureFirewall, error)
	Policy(context.Context, string) (n.FirewallPolicy, error)
	Subnet(context.Context, string) (n.Subnet, error)
	PublicIP(context.Context, string) (n.PublicIPAddress, error)
	RuleGroups(context.Context, string) ([]n.FirewallPolicyRuleCollectionGroup, error)
}

type FirewallGuard struct {
	worker    Config
	settings  FirewallSettings
	perimeter *PerimeterGuard
	reader    FirewallReader
}

func NewFirewallGuard(c Config, s FirewallSettings, perimeter *PerimeterGuard, reader FirewallReader) (*FirewallGuard, error) {
	s, err := s.canonical(c)
	if err != nil || perimeter == nil || reader == nil || fingerprint(c) != fingerprint(perimeter.worker) || fingerprint(s.Perimeter) != fingerprint(perimeter.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &FirewallGuard{worker: c, settings: s, perimeter: perimeter, reader: reader}, nil
}

func (g *FirewallGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := g.perimeter.Check(ctx, c); err != nil {
		return err
	}
	_, err := g.settings.binding().inspect(ctx, c, g.reader)
	return err
}

func (s firewallBinding) inspect(ctx context.Context, c Config, reader FirewallReader) (n.Subnet, error) {
	var firewall n.AzureFirewall
	var policy n.FirewallPolicy
	var subnet n.Subnet
	var public n.PublicIPAddress
	var errs [4]error
	var group sync.WaitGroup
	group.Go(func() { firewall, errs[0] = reader.Firewall(ctx, s.Firewall) })
	group.Go(func() { policy, errs[1] = reader.Policy(ctx, s.Policy) })
	group.Go(func() { subnet, errs[2] = reader.Subnet(ctx, s.subnet()) })
	group.Go(func() { public, errs[3] = reader.PublicIP(ctx, s.PublicIP) })
	group.Wait()
	for _, err := range errs {
		if err != nil {
			return n.Subnet{}, hostedexecution.ErrUnavailable
		}
	}
	if err := s.verifyFirewall(firewall, c.Region); err != nil {
		return n.Subnet{}, err
	}
	if err := s.verifySubnet(subnet); err != nil {
		return n.Subnet{}, err
	}
	if err := s.verifyPublicIP(public, c.Region); err != nil {
		return n.Subnet{}, err
	}
	groups, err := reader.RuleGroups(ctx, s.Policy)
	if err != nil {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return n.Subnet{}, hostedexecution.ErrOwnership
		}
		return n.Subnet{}, hostedexecution.ErrUnavailable
	}
	if err := s.verifyPolicy(policy, groups, c.Region); err != nil {
		return n.Subnet{}, err
	}
	return subnet, nil
}
