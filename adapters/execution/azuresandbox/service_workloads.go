package azuresandbox

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKServiceWorkloads struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func serviceWorkloadPath(s Settings, kind string) string {
	return "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.App/" + kind
}

func serviceWorkloadVersions(s Settings) map[string]string {
	versions := map[string]string{serviceWorkloadPath(s, "containerApps"): "2025-07-01", serviceWorkloadPath(s, "jobs"): "2025-07-01"}
	for _, identity := range s.Placement.Services.identities() {
		versions[identity.Resource] = "2024-11-30"
		versions[identity.Resource+"/federatedIdentityCredentials"] = "2024-11-30"
	}
	return versions
}

func NewSDKServiceWorkloads(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKServiceWorkloads, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(serviceWorkloadVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	reader.http.pages = map[string]bool{serviceWorkloadPath(s, "containerApps"): true, serviceWorkloadPath(s, "jobs"): true}
	for _, identity := range s.Placement.Services.identities() {
		reader.http.pages[identity.Resource+"/federatedIdentityCredentials"] = true
	}
	return &SDKServiceWorkloads{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKServiceWorkloads) Close() { c.http.Close() }

func (c *SDKServiceWorkloads) Check(ctx context.Context, config Config) error {
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

func (c *SDKServiceWorkloads) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKServiceWorkloads) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var identities, apps, jobs []placementResource
	var errors [7]error
	pins := c.settings.Placement.Services.identities()
	ids := []string{}
	for _, pin := range pins {
		ids = append(ids, pin.Resource)
	}
	var group sync.WaitGroup
	group.Go(func() { identities, errors[0] = c.observe(ctx, ids) })
	group.Go(func() {
		apps, errors[1] = c.inventoryPages(ctx, serviceWorkloadPath(c.settings, "containerApps"), 512, 16)
	})
	group.Go(func() { jobs, errors[2] = c.inventoryPages(ctx, serviceWorkloadPath(c.settings, "jobs"), 512, 16) })
	for i, pin := range pins {
		group.Go(func() { _, errors[i+3] = c.inventoryPages(ctx, pin.Resource+"/federatedIdentityCredentials", 0, 1) })
	}
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	for i, pin := range pins {
		if err := c.settings.verifyAuthorityIdentity(identities[i], pin.Resource, pin.Client, pin.Principal); err != nil {
			return err
		}
	}
	applications, err := resourceModels[a.ContainerApp](apps)
	if err != nil {
		return err
	}
	tasks, err := resourceModels[a.Job](jobs)
	if err != nil {
		return err
	}
	return azureworker.VerifyServiceWorkloadInventory(c.settings.Worker.Subscription, []azureworker.ServiceWorkloadBinding{c.settings.serviceWorkloadBinding(false), c.settings.serviceWorkloadBinding(true)}, applications, tasks)
}
