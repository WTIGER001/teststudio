package azuresandbox

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKRegistryService struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func registryServiceVersions(s Settings) map[string]string {
	versions := map[string]string{}
	for _, suffix := range []string{"", "/privateEndpointConnections", "/privateLinkResources", "/replications"} {
		versions[s.Placement.Registry+suffix] = "2025-11-01"
	}
	return versions
}

func NewSDKRegistryService(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKRegistryService, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	s.Placement.RegistryConsumers = slices.Clone(s.Placement.RegistryConsumers)
	reader, err := newMetadataReader(registryServiceVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	reader.http.pages = map[string]bool{}
	for _, suffix := range []string{"/privateEndpointConnections", "/privateLinkResources", "/replications"} {
		reader.http.pages[s.Placement.Registry+suffix] = true
	}
	return &SDKRegistryService{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKRegistryService) Close() { c.http.Close() }

func (c *SDKRegistryService) Check(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || c.http.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	epoch, started := c.cache.begin(c.now)
	err := c.read(ctx)
	c.cache.remember(epoch, started, err, c.http.closed.Load)
	return err
}

func (c *SDKRegistryService) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKRegistryService) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	binding := c.settings.registryBinding()
	var registry placementResource
	var connections, links, replicas []placementResource
	var errors [4]error
	var group sync.WaitGroup
	group.Go(func() { registry, errors[0] = c.get(ctx, binding.Registry) })
	group.Go(func() {
		connections, errors[1] = c.inventoryPages(ctx, binding.Registry+"/privateEndpointConnections", len(binding.Endpoints), len(binding.Endpoints)+1)
	})
	group.Go(func() { links, errors[2] = c.inventoryPages(ctx, binding.Registry+"/privateLinkResources", 1, 2) })
	group.Go(func() { replicas, errors[3] = c.inventoryPages(ctx, binding.Registry+"/replications", 1, 2) })
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	model, err := resourceModel[a.Registry](registry)
	if err != nil {
		return err
	}
	approved, err := resourceModels[a.PrivateEndpointConnection](connections)
	if err != nil {
		return err
	}
	members, err := resourceModels[a.PrivateLinkResource](links)
	if err != nil {
		return err
	}
	regions, err := resourceModels[a.Replication](replicas)
	if err != nil {
		return err
	}
	return binding.Verify(azureworker.RegistryServiceObservation{Registry: model, Connections: approved, Links: members, Replicas: regions})
}
