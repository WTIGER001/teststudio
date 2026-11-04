package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type EndpointReader interface {
	Endpoint(context.Context, string) (n.PrivateEndpoint, error)
	Interface(context.Context, string) (n.Interface, error)
	Subnet(context.Context, string) (n.Subnet, error)
	SecurityGroup(context.Context, string) (n.SecurityGroup, error)
}

type EndpointGuard struct {
	worker   Config
	settings EndpointSettings
	dns      *PrivateDNSGuard
	reader   EndpointReader
}

func NewEndpointGuard(c Config, s EndpointSettings, dns *PrivateDNSGuard, reader EndpointReader) (*EndpointGuard, error) {
	s, err := s.canonical(c)
	if err != nil || dns == nil || reader == nil || fingerprint(c) != fingerprint(dns.worker) || fingerprint(s.PrivateDNS) != fingerprint(dns.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &EndpointGuard{worker: c, settings: s, dns: dns, reader: reader}, nil
}

func (g *EndpointGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if err := g.dns.Check(ctx, c); err != nil {
		return err
	}
	roles := g.settings.roles()
	endpoints := make([]n.PrivateEndpoint, len(roles))
	interfaces := make([]n.Interface, len(roles))
	var subnet n.Subnet
	var security n.SecurityGroup
	errs := make([]error, 6)
	var group sync.WaitGroup
	for i, role := range roles {
		group.Go(func() { endpoints[i], errs[i*2] = g.reader.Endpoint(ctx, role.endpoint) })
		group.Go(func() { interfaces[i], errs[i*2+1] = g.reader.Interface(ctx, role.nic) })
	}
	group.Go(func() { subnet, errs[4] = g.reader.Subnet(ctx, g.settings.Subnet) })
	group.Go(func() { security, errs[5] = g.reader.SecurityGroup(ctx, g.settings.SecurityGroup) })
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
	for i, role := range roles {
		if err := g.verifyEndpoint(role, endpoints[i]); err != nil {
			return err
		}
		if err := g.verifyInterface(role, interfaces[i]); err != nil {
			return err
		}
	}
	if err := g.verifySubnet(subnet, interfaces); err != nil {
		return err
	}
	return g.verifySecurityGroup(security)
}
