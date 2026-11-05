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

type SDKHost struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func hostVersions(s Settings) map[string]string {
	h := s.Host
	result := map[string]string{}
	for _, path := range []string{h.Application, h.Environment, h.Application + "/authConfigs/current", h.Application + "/revisions", h.Environment + "/httpRouteConfigs", h.Environment + "/privateEndpointConnections"} {
		result[path] = "2025-07-01"
	}
	return result
}

func NewSDKHost(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKHost, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	if s.Host.Runtime != nil {
		copy := *s.Host.Runtime
		s.Host.Runtime = &copy
	}
	reader, err := newMetadataReader(hostVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	h := s.Host
	reader.http.pages = map[string]bool{h.Application + "/revisions": true, h.Environment + "/httpRouteConfigs": true, h.Environment + "/privateEndpointConnections": true}
	return &SDKHost{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKHost) Close() { c.http.Close() }

func (c *SDKHost) Check(ctx context.Context, config Config) error {
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

func (c *SDKHost) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKHost) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	h := c.settings.Host
	var values, revisions []placementResource
	var errors [4]error
	var group sync.WaitGroup
	group.Go(func() {
		values, errors[0] = c.observe(ctx, []string{h.Application, h.Environment, h.Application + "/authConfigs/current"})
	})
	group.Go(func() { revisions, errors[1] = c.inventoryPages(ctx, h.Application+"/revisions", 4, 4) })
	group.Go(func() { _, errors[2] = c.inventoryPages(ctx, h.Environment+"/httpRouteConfigs", 0, 1) })
	group.Go(func() { _, errors[3] = c.inventoryPages(ctx, h.Environment+"/privateEndpointConnections", 0, 1) })
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	environment, err := resourceModel[a.ManagedEnvironment](values[1])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateServiceEnvironment(environment, h.Environment, h.Subnet, h.InfrastructureGroup, c.settings.Worker.Region); err != nil {
		return err
	}
	if len(environment.Properties.PrivateEndpointConnections) != 0 {
		return hostedexecution.ErrOwnership
	}
	app, err := resourceModel[a.ContainerApp](values[0])
	if err != nil {
		return err
	}
	auth, err := resourceModel[a.AuthConfig](values[2])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyServiceAuth(auth, h.Application); err != nil {
		return err
	}
	if err := azureworker.VerifyServiceIdentityPins(app, c.settings.hostWorkloadBinding().Identities); err != nil {
		return err
	}
	models, err := resourceModels[a.Revision](revisions)
	if err != nil {
		return err
	}
	return (azureworker.ServiceApplicationBinding{Application: h.Application, Environment: h.Environment, Region: c.settings.Worker.Region, Backend: true, Expected: c.settings.hostDefinition()}).Verify(app, models)
}
