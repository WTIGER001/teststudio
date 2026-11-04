package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const registryVersion = "2025-11-01"

type SDKRegistryReader struct {
	settings     RegistrySettings
	registries   *a.RegistriesClient
	connections  *a.PrivateEndpointConnectionsClient
	replications *a.ReplicationsClient
	http         *armReadHTTP
}

func NewSDKRegistryReader(c Config, s RegistrySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKRegistryReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	allowed := map[string]string{}
	for _, id := range []string{s.registry(), s.connections(), s.links(), s.replicas()} {
		allowed[id] = registryVersion
	}
	h := newARMReadHTTP(c, allowed, transport)
	h.pages = map[string]bool{s.connections(): true, s.links(): true, s.replicas(): true}
	factory, err := a.NewClientFactory(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKRegistryReader{settings: s, registries: factory.NewRegistriesClient(), connections: factory.NewPrivateEndpointConnectionsClient(), replications: factory.NewReplicationsClient(), http: h}, nil
}

func (c *SDKRegistryReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKRegistryReader) Registry(ctx context.Context, id string) (a.Registry, error) {
	if id != c.settings.registry() {
		return a.Registry{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.registries.Get(ctx, group, name, nil)
	return response.Registry, sdkError(err, false)
}

func (c *SDKRegistryReader) Connections(ctx context.Context, id string) ([]a.PrivateEndpointConnection, error) {
	if id != c.settings.registry() {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.connections.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.connections(), registryVersion, 2, 3, pager.More, func(ctx context.Context) ([]*a.PrivateEndpointConnection, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

func (c *SDKRegistryReader) Links(ctx context.Context, id string) ([]a.PrivateLinkResource, error) {
	if id != c.settings.registry() {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.registries.NewListPrivateLinkResourcesPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.links(), registryVersion, 1, 2, pager.More, func(ctx context.Context) ([]*a.PrivateLinkResource, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

func (c *SDKRegistryReader) Replicas(ctx context.Context, id string) ([]a.Replication, error) {
	if id != c.settings.registry() {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.replications.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.settings.replicas(), registryVersion, 1, 2, pager.More, func(ctx context.Context) ([]*a.Replication, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

var _ RegistryReader = (*SDKRegistryReader)(nil)
