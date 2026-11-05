package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayRegistryEndpointGuard struct {
	worker   Config
	settings RelayRegistryEndpointSettings
	dns      *RelayDNSGuard
	reader   EndpointReader
}

func NewRelayRegistryEndpointGuard(c Config, s RelayRegistryEndpointSettings, dns *RelayDNSGuard, reader EndpointReader) (*RelayRegistryEndpointGuard, error) {
	s, err := s.canonical(c)
	if err != nil || dns == nil || reader == nil || fingerprint(c) != fingerprint(dns.worker) || fingerprint(s.DNS) != fingerprint(dns.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RelayRegistryEndpointGuard{worker: c, settings: s, dns: dns, reader: reader}, nil
}

type SDKRelayRegistryEndpointReader struct{ *sdkEndpointReader }

func NewSDKRelayRegistryEndpointReader(c Config, s RelayRegistryEndpointSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKRelayRegistryEndpointReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newSDKEndpointReader(c, []string{s.Endpoint}, []string{s.Interface}, []string{s.Subnet}, []string{s.SecurityGroup}, credential, transport)
	if err != nil {
		return nil, err
	}
	return &SDKRelayRegistryEndpointReader{sdkEndpointReader: reader}, nil
}

func (g *RelayRegistryEndpointGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 80*time.Second)
	defer cancel()
	observed, err := g.dns.inspect(ctx, c)
	if err != nil {
		return err
	}
	s := g.settings
	var endpoint n.PrivateEndpoint
	var nic n.Interface
	var subnet n.Subnet
	var security n.SecurityGroup
	var errs [4]error
	var group sync.WaitGroup
	group.Go(func() { endpoint, errs[0] = g.reader.Endpoint(ctx, s.Endpoint) })
	group.Go(func() { nic, errs[1] = g.reader.Interface(ctx, s.Interface) })
	group.Go(func() { subnet, errs[2] = g.reader.Subnet(ctx, s.Subnet) })
	group.Go(func() { security, errs[3] = g.reader.SecurityGroup(ctx, s.SecurityGroup) })
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
	if err := verifyEndpointSubnet(subnet, []n.Interface{nic}, s.Subnet, s.SubnetPrefix, s.SecurityGroup, []string{s.Endpoint}); err != nil {
		return err
	}
	if err := verifyEndpointSecurityGroup(security, s.SecurityGroup, s.Subnet, c.Region, s.rules()); err != nil {
		return err
	}
	if !embeddedSubnet(observed.network, s.Subnet, subnet) {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
