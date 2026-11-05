package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationRegistryGuard struct {
	worker   Config
	settings PreparationRegistrySettings
	registry *RegistryGuard
	reader   PreparationRegistryReader
	dns      PrivateDNSReader
}

func NewPreparationRegistryGuard(c Config, s PreparationRegistrySettings, registry *RegistryGuard, reader PreparationRegistryReader, dns PrivateDNSReader) (*PreparationRegistryGuard, error) {
	s, err := s.canonical(c)
	if err != nil || registry == nil || reader == nil || dns == nil || fingerprint(c) != fingerprint(registry.worker) || fingerprint(s.Registry) != fingerprint(registry.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &PreparationRegistryGuard{worker: c, settings: s, registry: registry, reader: reader, dns: dns}, nil
}
func (g *PreparationRegistryGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	if err := g.registry.Check(ctx, c); err != nil {
		return err
	}
	s := g.settings
	var endpoint n.PrivateEndpoint
	var nic n.Interface
	var subnet, source n.Subnet
	var security n.SecurityGroup
	var network n.VirtualNetwork
	var errs [6]error
	var group sync.WaitGroup
	group.Go(func() { endpoint, errs[0] = g.reader.Endpoint(ctx, s.Registry.PreparationEndpoint) })
	group.Go(func() { nic, errs[1] = g.reader.Interface(ctx, s.Interface) })
	group.Go(func() { subnet, errs[2] = g.reader.Subnet(ctx, s.Subnet) })
	group.Go(func() { security, errs[3] = g.reader.SecurityGroup(ctx, s.SecurityGroup) })
	group.Go(func() { source, errs[4] = g.reader.Subnet(ctx, s.SourceSubnet) })
	group.Go(func() { network, errs[5] = g.reader.VirtualNetwork(ctx, s.network()) })
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
	if err := verifyPrivateEndpoint(s.role(), endpoint, s.Subnet, c.Region); err != nil {
		return err
	}
	if err := verifyPrivateInterface(s.role(), nic, s.Subnet, c.Region); err != nil {
		return err
	}
	if err := verifyEndpointSubnet(subnet, []n.Interface{nic}, s.Subnet, s.SubnetPrefix, s.SecurityGroup, []string{s.Registry.PreparationEndpoint}); err != nil {
		return err
	}
	if err := verifyEndpointSecurityGroup(security, s.SecurityGroup, s.Subnet, c.Region, s.rules()); err != nil {
		return err
	}
	if err := verifyPreparationRegistryNetwork(c, s, network, subnet, source); err != nil {
		return err
	}
	return checkPrivateZones(ctx, g.dns, []privateZoneBinding{s.binding(c)})
}
