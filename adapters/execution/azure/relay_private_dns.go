package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayPrivateDNSGuard struct {
	worker   Config
	settings RelayPrivateDNSSettings
	endpoint *RelayRegistryEndpointGuard
	reader   PrivateDNSReader
}

func NewRelayPrivateDNSGuard(c Config, s RelayPrivateDNSSettings, endpoint *RelayRegistryEndpointGuard, reader PrivateDNSReader) (*RelayPrivateDNSGuard, error) {
	s, err := s.canonical(c)
	if err != nil || endpoint == nil || reader == nil || fingerprint(c) != fingerprint(endpoint.worker) || fingerprint(s.Endpoint) != fingerprint(endpoint.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RelayPrivateDNSGuard{worker: c, settings: s, endpoint: endpoint, reader: reader}, nil
}

func NewSDKRelayPrivateDNSReader(c Config, s RelayPrivateDNSSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivateDNSReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKPrivateDNSReader(c, []privateZoneBinding{s.binding(c)}, credential, transport)
}

func (g *RelayPrivateDNSGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 85*time.Second)
	defer cancel()
	if err := g.endpoint.Check(ctx, c); err != nil {
		return err
	}
	return checkPrivateZones(ctx, g.reader, []privateZoneBinding{g.settings.binding(c)})
}

func (s RelayPrivateDNSSettings) ZoneDefinition(c Config) (p.PrivateZone, error) {
	if _, err := s.canonical(c); err != nil {
		return p.PrivateZone{}, err
	}
	return p.PrivateZone{Location: to.Ptr("global")}, nil
}

func (s RelayPrivateDNSSettings) LinkDefinition(c Config) (p.VirtualNetworkLink, error) {
	if _, err := s.canonical(c); err != nil {
		return p.VirtualNetworkLink{}, err
	}
	return s.binding(c).linkDefinition(), nil
}

func (s RelayPrivateDNSSettings) RecordDefinitions(c Config) (map[string]p.RecordSet, error) {
	if _, err := s.canonical(c); err != nil {
		return nil, err
	}
	return s.binding(c).recordDefinitions(), nil
}
