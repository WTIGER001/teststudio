package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationRegistryReader interface {
	EndpointReader
	VirtualNetwork(context.Context, string) (n.VirtualNetwork, error)
}
type SDKPreparationRegistryReader struct {
	*sdkEndpointReader
	network  string
	networks *n.VirtualNetworksClient
}

func NewSDKPreparationRegistryReader(c Config, s PreparationRegistrySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPreparationRegistryReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	core, err := newSDKEndpointReader(c, []string{s.Registry.PreparationEndpoint}, []string{s.Interface}, []string{s.Subnet, s.SourceSubnet}, []string{s.SecurityGroup}, credential, transport)
	if err != nil {
		return nil, err
	}
	core.http.resources[s.network()] = "2025-03-01"
	networks, err := n.NewVirtualNetworksClient(c.Subscription, credential, publicARMOptions(core.http))
	if err != nil {
		core.Close()
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPreparationRegistryReader{sdkEndpointReader: core, network: s.network(), networks: networks}, nil
}
func (c *SDKPreparationRegistryReader) VirtualNetwork(ctx context.Context, id string) (n.VirtualNetwork, error) {
	if id != c.network {
		return n.VirtualNetwork{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.networks.Get(ctx, group, name, nil)
	return result.VirtualNetwork, sdkError(err, false)
}
func NewSDKPreparationRegistryDNSReader(c Config, s PreparationRegistrySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivateDNSReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKPrivateDNSReader(c, []privateZoneBinding{s.binding(c)}, credential, transport)
}
