package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayDNSGuard struct {
	worker   Config
	settings RelayDNSSettings
	firewall *RelayFirewallGuard
	reader   DNSReader
}

func NewRelayDNSGuard(c Config, s RelayDNSSettings, firewall *RelayFirewallGuard, reader DNSReader) (*RelayDNSGuard, error) {
	s, err := s.canonical(c)
	if err != nil || firewall == nil || reader == nil || fingerprint(c) != fingerprint(firewall.worker) || fingerprint(s.Firewall) != fingerprint(firewall.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RelayDNSGuard{worker: c, settings: s, firewall: firewall, reader: reader}, nil
}

func NewSDKRelayDNSReader(c Config, s RelayDNSSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKDNSReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKDNSReader(c, s.binding(c), credential, transport)
}

func (g *RelayDNSGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	if err := g.firewall.Check(ctx, c); err != nil {
		return err
	}
	return g.settings.binding(c).check(ctx, c, g.reader)
}

func (s RelayDNSSettings) PolicyDefinition(c Config) (d.Policy, error) {
	if _, err := s.canonical(c); err != nil {
		return d.Policy{}, err
	}
	return d.Policy{Location: to.Ptr(c.Region)}, nil
}

func (s RelayDNSSettings) DomainListDefinition(c Config, allow bool) (d.DomainList, error) {
	if _, err := s.canonical(c); err != nil {
		return d.DomainList{}, err
	}
	return s.binding(c).domainListDefinition(c.Region, allow), nil
}

func (s RelayDNSSettings) RuleDefinition(c Config, allow bool) (d.DNSSecurityRule, error) {
	if _, err := s.canonical(c); err != nil {
		return d.DNSSecurityRule{}, err
	}
	return s.binding(c).ruleDefinition(c.Region, allow), nil
}

func (s RelayDNSSettings) LinkDefinition(c Config) (d.PolicyVirtualNetworkLink, error) {
	if _, err := s.canonical(c); err != nil {
		return d.PolicyVirtualNetworkLink{}, err
	}
	return s.binding(c).linkDefinition(c.Region), nil
}
