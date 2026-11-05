package azuresandbox

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKHostDatabase struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func hostDatabaseIDs(s Settings) []string {
	d := s.Host.Database
	return []string{d.Server, d.Endpoint, d.Interface, d.Subnet, d.SecurityGroup}
}

func hostDatabaseVersions(s Settings) map[string]string {
	d := s.Host.Database
	result := map[string]string{}
	for _, id := range hostDatabaseIDs(s) {
		result[id] = "2025-03-01"
	}
	for _, suffix := range []string{"", "/privateEndpointConnections", "/privateLinkResources"} {
		result[d.Server+suffix] = "2024-08-01"
	}
	for _, suffix := range []string{"", "/ALL", "/virtualNetworkLinks"} {
		result[d.Zone+suffix] = "2024-06-01"
	}
	return result
}

func NewSDKHostDatabase(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKHostDatabase, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	s.Host.Database.Consumers = slices.Clone(s.Host.Database.Consumers)
	reader, err := newMetadataReader(hostDatabaseVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	d := s.Host.Database
	reader.http.pages = map[string]bool{d.Server + "/privateEndpointConnections": true, d.Server + "/privateLinkResources": true, d.Zone + "/ALL": true, d.Zone + "/virtualNetworkLinks": true}
	return &SDKHostDatabase{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKHostDatabase) Close() { c.http.Close() }

func (c *SDKHostDatabase) Check(ctx context.Context, config Config) error {
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

func (c *SDKHostDatabase) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKHostDatabase) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	d := c.settings.Host.Database
	var values, connections, links []placementResource
	var errors [4]error
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, hostDatabaseIDs(c.settings)) })
	group.Go(func() {
		connections, errors[1] = c.inventoryPages(ctx, d.Server+"/privateEndpointConnections", len(d.consumers()), len(d.consumers())+1)
	})
	group.Go(func() { links, errors[2] = c.inventoryPages(ctx, d.Server+"/privateLinkResources", 1, 2) })
	group.Go(func() { errors[3] = c.readPrivateZone(ctx, c.settings.Host.databaseZone()) })
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	if err := c.verifyNetwork(values); err != nil {
		return err
	}
	return c.settings.verifyDatabaseService(values[0], connections, links)
}
