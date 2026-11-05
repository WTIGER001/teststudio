package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	s "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const preparationStorageVersion = "2024-01-01"

type SDKPreparationNetworkReader struct {
	*sdkEndpointReader
	settings PreparationSettings
	networks *n.VirtualNetworksClient
	nats     *n.NatGatewaysClient
	ips      *n.PublicIPAddressesClient
	accounts *s.AccountsClient
	shares   *s.FileSharesClient
}

func NewSDKPreparationNetworkReader(settings PreparationSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPreparationNetworkReader, error) {
	if settings.ValidateNative() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	settings.Writer, _ = settings.Writer.canonical(settings.Worker)
	network, _ := settings.Network.canonical(settings)
	host := *settings.Host
	settings.Network, settings.Host = &network, &host
	c, p, r := settings.Worker, *settings.Network, settings.Writer.Registry
	core, err := newSDKEndpointReader(c, []string{p.FileEndpoint}, []string{p.FileInterface}, []string{p.FileSubnet, r.SourceSubnet}, []string{p.FileSecurityGroup, p.SourceSecurityGroup}, credential, transport)
	if err != nil {
		return nil, err
	}
	for _, id := range []string{r.network(), p.NATGateway, p.NATPublicIP} {
		core.http.resources[id] = "2025-03-01"
	}
	for _, id := range []string{p.StorageAccount, p.share(settings)} {
		core.http.resources[id] = preparationStorageVersion
	}
	options := publicARMOptions(core.http)
	failed := func() (*SDKPreparationNetworkReader, error) {
		core.Close()
		return nil, hostedexecution.ErrConfiguration
	}
	networks, err := n.NewVirtualNetworksClient(c.Subscription, credential, options)
	if err != nil {
		return failed()
	}
	nats, err := n.NewNatGatewaysClient(c.Subscription, credential, options)
	if err != nil {
		return failed()
	}
	ips, err := n.NewPublicIPAddressesClient(c.Subscription, credential, options)
	if err != nil {
		return failed()
	}
	storage, err := s.NewClientFactory(c.Subscription, credential, options)
	if err != nil {
		return failed()
	}
	return &SDKPreparationNetworkReader{sdkEndpointReader: core, settings: settings, networks: networks, nats: nats, ips: ips, accounts: storage.NewAccountsClient(), shares: storage.NewFileSharesClient()}, nil
}

func (c *SDKPreparationNetworkReader) Network(ctx context.Context, id string) (n.VirtualNetwork, error) {
	if id != c.settings.Writer.Registry.network() {
		return n.VirtualNetwork{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.networks.Get(ctx, group, name, nil)
	return response.VirtualNetwork, sdkError(err, false)
}

func (c *SDKPreparationNetworkReader) NAT(ctx context.Context, id string) (n.NatGateway, error) {
	if id != c.settings.Network.NATGateway {
		return n.NatGateway{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.nats.Get(ctx, group, name, nil)
	return response.NatGateway, sdkError(err, false)
}

func (c *SDKPreparationNetworkReader) PublicIP(ctx context.Context, id string) (n.PublicIPAddress, error) {
	if id != c.settings.Network.NATPublicIP {
		return n.PublicIPAddress{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.ips.Get(ctx, group, name, nil)
	return response.PublicIPAddress, sdkError(err, false)
}

func (c *SDKPreparationNetworkReader) Account(ctx context.Context, id string) (s.Account, error) {
	if id != c.settings.Network.StorageAccount {
		return s.Account{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.accounts.GetProperties(ctx, group, name, nil)
	return response.Account, sdkError(err, false)
}

func (c *SDKPreparationNetworkReader) Share(ctx context.Context, id string) (s.FileShare, error) {
	if id != c.settings.Network.share(c.settings) {
		return s.FileShare{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(c.settings.Network.StorageAccount)
	response, err := c.shares.Get(ctx, group, name, strings.TrimPrefix(id, c.settings.Network.StorageAccount+"/fileServices/default/shares/"), nil)
	return response.FileShare, sdkError(err, false)
}

func NewSDKPreparationFileDNSReader(settings PreparationSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivateDNSReader, error) {
	if settings.ValidateNative() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKPrivateDNSReader(settings.Worker, []privateZoneBinding{settings.Network.binding(settings)}, credential, transport)
}
