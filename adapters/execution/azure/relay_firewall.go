package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayFirewallGuard struct {
	worker    Config
	settings  RelayFirewallSettings
	perimeter *RelayPerimeterGuard
	reader    FirewallReader
}

func NewRelayFirewallGuard(c Config, s RelayFirewallSettings, perimeter *RelayPerimeterGuard, reader FirewallReader) (*RelayFirewallGuard, error) {
	s, err := s.canonical(c)
	if err != nil || perimeter == nil || reader == nil || fingerprint(c) != fingerprint(perimeter.worker) || fingerprint(s.Perimeter) != fingerprint(perimeter.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RelayFirewallGuard{worker: c, settings: s, perimeter: perimeter, reader: reader}, nil
}

func NewSDKRelayFirewallReader(c Config, s RelayFirewallSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKFirewallReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKFirewallReader(c, s.binding(), credential, transport)
}

func (g *RelayFirewallGuard) Check(ctx context.Context, c Config) error {
	_, err := g.inspect(ctx, c)
	return err
}

func (g *RelayFirewallGuard) inspect(ctx context.Context, c Config) (observation relayTopologyObservation, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return relayTopologyObservation{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 70*time.Second)
	defer cancel()
	observed, err := g.perimeter.inspect(ctx, c)
	if err != nil {
		return relayTopologyObservation{}, err
	}
	binding := g.settings.binding()
	subnet, err := binding.inspect(ctx, c, g.reader)
	if err != nil {
		return relayTopologyObservation{}, err
	}
	if !embeddedSubnet(observed.network, binding.subnet(), subnet) {
		return relayTopologyObservation{}, hostedexecution.ErrUnavailable
	}
	return observed, nil
}

func (s RelayFirewallSettings) FirewallDefinition(c Config) (n.AzureFirewall, error) {
	if _, err := s.canonical(c); err != nil {
		return n.AzureFirewall{}, err
	}
	return s.binding().firewallDefinition(c.Region), nil
}

func (s RelayFirewallSettings) PolicyDefinition(c Config) (n.FirewallPolicy, error) {
	if _, err := s.canonical(c); err != nil {
		return n.FirewallPolicy{}, err
	}
	return s.binding().policyDefinition(c.Region), nil
}

func (s RelayFirewallSettings) RuleGroupDefinition(c Config) (n.FirewallPolicyRuleCollectionGroup, error) {
	if _, err := s.canonical(c); err != nil {
		return n.FirewallPolicyRuleCollectionGroup{}, err
	}
	return s.binding().ruleGroupDefinition(), nil
}
