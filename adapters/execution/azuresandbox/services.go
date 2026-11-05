package azuresandbox

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKServices struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func serviceIDs(s Settings) []string {
	p, v := s.Placement, s.Placement.Services
	return []string{p.Private.ExchangeEnvironment, v.Relay.Application, v.Relay.Certificate, v.Relay.Application + "/authConfigs/current", v.Backend.Environment, v.Backend.Application, v.Backend.Application + "/authConfigs/current"}
}

func serviceCollections(s Settings) map[string]int {
	p, v := s.Placement, s.Placement.Services
	return map[string]int{p.Private.ExchangeEnvironment + "/privateEndpointConnections": 1, p.Private.ExchangeEnvironment + "/privateLinkResources": 1, p.Private.ExchangeEnvironment + "/httpRouteConfigs": 0, v.Relay.Application + "/revisions": 4, v.Backend.Environment + "/httpRouteConfigs": 0, v.Backend.Application + "/revisions": 4}
}

func serviceVersions(s Settings) map[string]string {
	result := map[string]string{}
	for _, id := range serviceIDs(s) {
		result[id] = "2025-07-01"
	}
	for id := range serviceCollections(s) {
		result[id] = "2025-07-01"
	}
	return result
}

func NewSDKServices(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKServices, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(serviceVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	reader.http.pages = map[string]bool{}
	for path := range serviceCollections(s) {
		reader.http.pages[path] = true
	}
	return &SDKServices{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKServices) Close() { c.http.Close() }

func (c *SDKServices) Check(ctx context.Context, config Config) error {
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

func (c *SDKServices) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKServices) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var values []placementResource
	paths := []string{}
	limits := serviceCollections(c.settings)
	for path := range limits {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	observed := make([]serviceObservation, len(paths))
	var group sync.WaitGroup
	var directError error
	group.Go(func() { values, directError = c.observe(ctx, serviceIDs(c.settings)) })
	for i, path := range paths {
		group.Go(func() {
			limit := limits[path]
			pages := 2
			if limit == 4 {
				pages = 4
			}
			entries, err := c.inventoryPages(ctx, path, limit, pages)
			observed[i] = serviceObservation{entries, err}
		})
	}
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	if directError != nil {
		return directError
	}
	collections := map[string][]placementResource{}
	for i, path := range paths {
		observation := observed[i]
		if observation.err != nil {
			return observation.err
		}
		collections[path] = observation.values
	}
	return c.verify(values, collections)
}

type serviceObservation struct {
	values []placementResource
	err    error
}

func (c *SDKServices) verifyApplication(app a.ContainerApp, revisions []a.Revision, domain string, backend bool) error {
	s := c.settings
	r, b := s.Placement.Services.Relay, s.Placement.Services.Backend
	application, environment, certificate := r.Application, s.Placement.Private.ExchangeEnvironment, r.Certificate
	expected := s.relayDefinition()
	pins := map[string]azureworker.ServiceIdentityPin{r.Pull.Resource: {Client: r.Pull.Client, Principal: r.Pull.Principal, Lifecycle: "None"}}
	if backend {
		application, environment, certificate = b.Application, b.Environment, ""
		expected = s.backendDefinition()
		pins = map[string]azureworker.ServiceIdentityPin{b.Pull.Resource: {Client: b.Pull.Client, Principal: b.Pull.Principal, Lifecycle: "None"}, b.Startup.Resource: {Client: b.Startup.Client, Principal: b.Startup.Principal, Lifecycle: "None"}, b.Runtime.Resource: {Client: b.Runtime.Client, Principal: b.Runtime.Principal, Lifecycle: "Main"}}
		name := application[strings.LastIndex(application, "/")+1:]
		if b.Origin != "https://"+name+"."+domain {
			return hostedexecution.ErrOwnership
		}
	}
	if err := azureworker.VerifyServiceIdentityPins(app, pins); err != nil {
		return err
	}
	binding := azureworker.ServiceApplicationBinding{Application: application, Environment: environment, Region: s.Worker.Region, Domain: domain, Certificate: certificate, Backend: backend, Expected: expected}
	return binding.Verify(app, revisions)
}
