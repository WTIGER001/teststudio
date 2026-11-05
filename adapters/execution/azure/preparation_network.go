package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	s "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationNetworkReader interface {
	EndpointReader
	Network(context.Context, string) (n.VirtualNetwork, error)
	NAT(context.Context, string) (n.NatGateway, error)
	PublicIP(context.Context, string) (n.PublicIPAddress, error)
	Account(context.Context, string) (s.Account, error)
	Share(context.Context, string) (s.FileShare, error)
}

type PreparationNetworkGuard struct {
	settings PreparationSettings
	host     *PreparationHostGuard
	reader   PreparationNetworkReader
	dns      PrivateDNSReader
}

func NewPreparationNetworkGuard(settings PreparationSettings, host *PreparationHostGuard, reader PreparationNetworkReader, dns PrivateDNSReader) (*PreparationNetworkGuard, error) {
	if settings.ValidateNative() != nil || host == nil || reader == nil || dns == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	settings.Writer, _ = settings.Writer.canonical(settings.Worker)
	selected, expected := settings, host.settings
	selected.Network, expected.Network = nil, nil
	if fingerprint(selected) != fingerprint(expected) {
		return nil, hostedexecution.ErrConfiguration
	}
	network, _ := settings.Network.canonical(settings)
	settings = host.settings
	settings.Network = &network
	return &PreparationNetworkGuard{settings: settings, host: host, reader: reader, dns: dns}, nil
}

type preparationNetworkObservation struct {
	endpoint           n.PrivateEndpoint
	nic                n.Interface
	file, source       n.Subnet
	fileNSG, sourceNSG n.SecurityGroup
	nat                n.NatGateway
	ip                 n.PublicIPAddress
	network            n.VirtualNetwork
	account            s.Account
	share              s.FileShare
}

func (g *PreparationNetworkGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	defer cancel()
	if err := g.host.Check(ctx, c); err != nil {
		return err
	}
	s, r := g.settings, g.settings.Writer.Registry
	p := s.Network
	var o preparationNetworkObservation
	var errs [11]error
	var group sync.WaitGroup
	group.Go(func() { o.endpoint, errs[0] = g.reader.Endpoint(ctx, p.FileEndpoint) })
	group.Go(func() { o.nic, errs[1] = g.reader.Interface(ctx, p.FileInterface) })
	group.Go(func() { o.file, errs[2] = g.reader.Subnet(ctx, p.FileSubnet) })
	group.Go(func() { o.fileNSG, errs[3] = g.reader.SecurityGroup(ctx, p.FileSecurityGroup) })
	group.Go(func() { o.source, errs[4] = g.reader.Subnet(ctx, r.SourceSubnet) })
	group.Go(func() { o.sourceNSG, errs[5] = g.reader.SecurityGroup(ctx, p.SourceSecurityGroup) })
	group.Go(func() { o.nat, errs[6] = g.reader.NAT(ctx, p.NATGateway) })
	group.Go(func() { o.ip, errs[7] = g.reader.PublicIP(ctx, p.NATPublicIP) })
	group.Go(func() { o.network, errs[8] = g.reader.Network(ctx, r.network()) })
	group.Go(func() { o.account, errs[9] = g.reader.Account(ctx, p.StorageAccount) })
	group.Go(func() { o.share, errs[10] = g.reader.Share(ctx, p.share(s)) })
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
	for _, verify := range []func() error{
		func() error { return verifyPreparationSource(s, o) },
		func() error { return verifyPreparationNAT(s, o.nat, o.ip) },
		func() error { return verifyPreparationStorage(s, o.account, o.share) },
		func() error { return verifyPrivateEndpoint(p.role(), o.endpoint, p.FileSubnet, c.Region) },
		func() error { return verifyPrivateInterface(p.role(), o.nic, p.FileSubnet, c.Region) },
		func() error {
			return verifyEndpointSubnet(o.file, []n.Interface{o.nic}, p.FileSubnet, p.FileSubnetPrefix, p.FileSecurityGroup, []string{p.FileEndpoint})
		},
		func() error {
			return verifyEndpointSecurityGroup(o.fileNSG, p.FileSecurityGroup, p.FileSubnet, c.Region, s.fileRules())
		},
	} {
		if err := verify(); err != nil {
			return err
		}
	}
	return checkPrivateZones(ctx, g.dns, []privateZoneBinding{p.binding(s)})
}
