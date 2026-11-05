package azuresandbox

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKHostRegistry struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func hostRegistryIDs(s Settings) []string {
	h, r := s.Host, s.Host.Registry
	return []string{h.Subnet, r.Endpoint, r.Interface, r.Subnet, r.SecurityGroup, s.Placement.Registry, r.Connection, h.Pull.Resource, r.PullAssignment, "/subscriptions/" + s.Worker.Subscription + pullRole}
}

func hostRegistryVersions(s Settings) map[string]string {
	result := map[string]string{}
	for _, id := range hostRegistryIDs(s) {
		result[id] = "2025-03-01"
	}
	r := s.Host.Registry
	result[s.Placement.Registry], result[r.Connection] = "2025-11-01", "2025-11-01"
	result[s.Host.Pull.Resource], result[r.PullAssignment], result["/subscriptions/"+s.Worker.Subscription+pullRole] = "2024-11-30", "2022-04-01", "2022-05-01-preview"
	for _, suffix := range []string{"", "/ALL", "/virtualNetworkLinks"} {
		result[r.Zone+suffix] = "2024-06-01"
	}
	return result
}

func NewSDKHostRegistry(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKHostRegistry, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(hostRegistryVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	r := s.Host.Registry
	reader.http.pages = map[string]bool{r.Zone + "/ALL": true, r.Zone + "/virtualNetworkLinks": true}
	return &SDKHostRegistry{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKHostRegistry) Close() { c.http.Close() }

func (c *SDKHostRegistry) Check(ctx context.Context, config Config) error {
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

func (c *SDKHostRegistry) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKHostRegistry) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var values []placementResource
	var errors [2]error
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, hostRegistryIDs(c.settings)) })
	group.Go(func() { errors[1] = c.readPrivateZone(ctx, c.settings.Host.registryZone(c.settings)) })
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return c.verify(values)
}
