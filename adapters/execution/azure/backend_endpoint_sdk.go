package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKBackendEndpointReader struct {
	*sdkEndpointReader
	environment string
	connections *a.ManagedEnvironmentPrivateEndpointConnectionsClient
	links       *a.ManagedEnvironmentPrivateLinkResourcesClient
}

func NewSDKBackendEndpointReader(c Config, s BackendEndpointSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKBackendEndpointReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	core, err := newSDKEndpointReader(c, []string{s.Endpoint}, []string{s.Interface}, []string{s.Subnet, s.Backend.Exchange.Subnet}, []string{s.SecurityGroup}, credential, transport)
	if err != nil {
		return nil, err
	}
	core.http.pages = map[string]bool{}
	for _, kind := range []string{"privateEndpointConnections", "privateLinkResources"} {
		path := s.Backend.Environment + "/" + kind
		core.http.resources[path] = exchangeVersion
		core.http.pages[path] = true
	}
	factory, err := a.NewClientFactory(c.Subscription, credential, publicARMOptions(core.http))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKBackendEndpointReader{sdkEndpointReader: core, environment: s.Backend.Environment, connections: factory.NewManagedEnvironmentPrivateEndpointConnectionsClient(), links: factory.NewManagedEnvironmentPrivateLinkResourcesClient()}, nil
}

func (c *SDKBackendEndpointReader) Connections(ctx context.Context, id string) ([]a.PrivateEndpointConnection, error) {
	if id != c.environment {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.connections.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, id+"/privateEndpointConnections", exchangeVersion, 1, 2, pager.More, func(ctx context.Context) ([]*a.PrivateEndpointConnection, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

func (c *SDKBackendEndpointReader) Links(ctx context.Context, id string) ([]a.PrivateLinkResource, error) {
	if id != c.environment {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.links.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, id+"/privateLinkResources", exchangeVersion, 1, 2, pager.More, func(ctx context.Context) ([]*a.PrivateLinkResource, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

var _ BackendEndpointReader = (*SDKBackendEndpointReader)(nil)
