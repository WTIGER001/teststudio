package azureworker

import (
	"context"
	"net/netip"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayTopologyReader interface {
	VirtualNetwork(context.Context, string) (n.VirtualNetwork, error)
}

type RelayTopologyGuard struct {
	worker   Config
	settings RelayTopologySettings
	endpoint *BackendEndpointGuard
	reader   RelayTopologyReader
}

func NewRelayTopologyGuard(c Config, s RelayTopologySettings, endpoint *BackendEndpointGuard, reader RelayTopologyReader) (*RelayTopologyGuard, error) {
	s, err := s.canonical(c)
	if err != nil || endpoint == nil || reader == nil || fingerprint(c) != fingerprint(endpoint.worker) || fingerprint(s.BackendEndpoint) != fingerprint(endpoint.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RelayTopologyGuard{worker: c, settings: s, endpoint: endpoint, reader: reader}, nil
}

func (g *RelayTopologyGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	observed, err := g.endpoint.inspect(ctx, c)
	if err != nil {
		return err
	}
	id := g.settings.BackendEndpoint.Backend.Exchange.VirtualNetwork
	network, err := g.reader.VirtualNetwork(ctx, id)
	if err != nil {
		return err
	}
	if err := networkMatches(network, id, c.Region); err != nil {
		return err
	}
	p := network.Properties
	if len(p.AddressSpace.AddressPrefixes) != 1 || value(p.AddressSpace.AddressPrefixes[0]) != g.settings.VirtualNetworkPrefix || len(p.VirtualNetworkPeerings) != 0 || p.DefaultPublicNatGateway != nil || len(p.IPAllocations) != 0 || p.BgpCommunities != nil || p.DhcpOptions != nil && len(p.DhcpOptions.DNSServers) != 0 {
		return hostedexecution.ErrOwnership
	}
	return verifyRelaySubnets(c, network, observed, g.settings)
}

func verifyRelaySubnets(c Config, network n.VirtualNetwork, observed backendEndpointObservation, s RelayTopologySettings) error {
	p := network.Properties
	if p.Subnets == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(p.Subnets) < 2 || len(p.Subnets) > 256 {
		return hostedexecution.ErrOwnership
	}
	parent, _ := netip.ParsePrefix(s.VirtualNetworkPrefix)
	id := s.BackendEndpoint.Backend.Exchange.VirtualNetwork
	seen := map[string]bool{}
	var prefixes []netip.Prefix
	for _, subnet := range p.Subnets {
		if subnet == nil || !subnetResource(c, value(subnet.ID), id) || seen[strings.ToLower(value(subnet.ID))] {
			return hostedexecution.ErrOwnership
		}
		seen[strings.ToLower(value(subnet.ID))] = true
		if err := subnetMatches(*subnet, value(subnet.ID)); err != nil {
			return err
		}
		raw := subnetPrefixes(*subnet)
		if len(raw) != 1 {
			return hostedexecution.ErrOwnership
		}
		prefix, valid := privatePrefix(raw[0])
		if !valid || !containsPrefix(parent, prefix) {
			return hostedexecution.ErrOwnership
		}
		for _, previous := range prefixes {
			if prefix.Overlaps(previous) {
				return hostedexecution.ErrOwnership
			}
		}
		prefixes = append(prefixes, prefix)
	}
	if !embeddedSubnet(network, s.BackendEndpoint.Subnet, observed.subnet) || !embeddedSubnet(network, s.BackendEndpoint.Backend.Exchange.Subnet, observed.relay) {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
