package azureworker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayPerimeterGuard struct {
	worker   Config
	settings RelayPerimeterSettings
	topology *RelayTopologyGuard
	reader   PerimeterReader
}

func NewRelayPerimeterGuard(c Config, s RelayPerimeterSettings, topology *RelayTopologyGuard, reader PerimeterReader) (*RelayPerimeterGuard, error) {
	s, err := s.canonical(c)
	if err != nil || topology == nil || reader == nil || fingerprint(c) != fingerprint(topology.worker) || fingerprint(s.Topology) != fingerprint(topology.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RelayPerimeterGuard{worker: c, settings: s, topology: topology, reader: reader}, nil
}

func NewSDKRelayPerimeterReader(c Config, s RelayPerimeterSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPerimeterReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKPerimeterReader(c, s.SecurityGroup, s.RouteTable, credential, transport)
}

func (g *RelayPerimeterGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	observed, err := g.topology.inspect(ctx, c)
	if err != nil {
		return err
	}
	s := g.settings
	if err := verifyRelaySourceSubnet(observed.relay, s); err != nil {
		return err
	}
	var security n.SecurityGroup
	var routes n.RouteTable
	var errs [2]error
	var group sync.WaitGroup
	group.Go(func() { security, errs[0] = g.reader.SecurityGroup(ctx, s.SecurityGroup) })
	group.Go(func() { routes, errs[1] = g.reader.RouteTable(ctx, s.RouteTable) })
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
	subnet := s.Topology.BackendEndpoint.Backend.Exchange.Subnet
	if err := verifyEndpointSecurityGroup(security, s.SecurityGroup, subnet, c.Region, s.rules()); err != nil {
		return err
	}
	expected, _ := s.RouteTableDefinition(c)
	return verifyPerimeterRoutes(routes, s.RouteTable, subnet, c.Region, expected)
}

func verifyRelaySourceSubnet(subnet n.Subnet, s RelayPerimeterSettings) error {
	p := subnet.Properties
	if p == nil || p.DefaultOutboundAccess == nil || value(p.DefaultOutboundAccess) || p.NetworkSecurityGroup == nil || !strings.EqualFold(value(p.NetworkSecurityGroup.ID), s.SecurityGroup) || p.RouteTable == nil || !strings.EqualFold(value(p.RouteTable.ID), s.RouteTable) || p.NatGateway != nil || value(p.SharingScope) != "" || len(p.ServiceEndpoints)+len(p.ServiceEndpointPolicies)+len(p.PrivateEndpoints)+len(p.ApplicationGatewayIPConfigurations)+len(p.IPAllocations)+len(p.IpamPoolPrefixAllocations) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
