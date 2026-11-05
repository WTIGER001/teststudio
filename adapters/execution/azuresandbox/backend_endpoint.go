package azuresandbox

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKBackendEndpoint struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func backendEndpointIDs(s Settings) []string {
	v := s.Placement.Services
	e := v.BackendEndpoint
	return []string{v.Backend.Environment, e.Endpoint, e.Interface, e.Subnet, e.SecurityGroup, v.Relay.Subnet}
}

func backendEndpointVersions(s Settings) map[string]string {
	versions := map[string]string{}
	for _, id := range backendEndpointIDs(s) {
		versions[id] = "2025-03-01"
	}
	for _, suffix := range []string{"", "/privateEndpointConnections", "/privateLinkResources"} {
		versions[s.Placement.Services.Backend.Environment+suffix] = "2025-07-01"
	}
	return versions
}

func NewSDKBackendEndpoint(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKBackendEndpoint, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(backendEndpointVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	base := s.Placement.Services.Backend.Environment
	reader.http.pages = map[string]bool{base + "/privateEndpointConnections": true, base + "/privateLinkResources": true}
	return &SDKBackendEndpoint{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKBackendEndpoint) Close() { c.http.Close() }

func (c *SDKBackendEndpoint) Check(ctx context.Context, config Config) error {
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

func (c *SDKBackendEndpoint) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKBackendEndpoint) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var values, connections, links []placementResource
	var errors [3]error
	base := c.settings.Placement.Services.Backend.Environment
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, backendEndpointIDs(c.settings)) })
	group.Go(func() { connections, errors[1] = c.inventoryPages(ctx, base+"/privateEndpointConnections", 1, 2) })
	group.Go(func() { links, errors[2] = c.inventoryPages(ctx, base+"/privateLinkResources", 1, 2) })
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return c.verify(values, connections, links)
}

func (c *SDKBackendEndpoint) verify(values, connections, links []placementResource) error {
	if len(values) != 6 {
		return hostedexecution.ErrUnavailable
	}
	s := c.settings
	v, e := s.Placement.Services, s.Placement.Services.BackendEndpoint
	environment, err := resourceModel[a.ManagedEnvironment](values[0])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateServiceEnvironment(environment, v.Backend.Environment, v.Backend.Subnet, v.Backend.InfrastructureGroup, s.Worker.Region); err != nil {
		return err
	}
	approved, err := resourceModels[a.PrivateEndpointConnection](connections)
	if err != nil {
		return err
	}
	if err := azureworker.VerifyServiceEndpointConnections(environment, approved, e.Endpoint); err != nil {
		return err
	}
	resources, err := resourceModels[a.PrivateLinkResource](links)
	if err != nil {
		return err
	}
	member, err := azureworker.VerifyServicePrivateLinkMember(resources, v.Backend.Environment, s.Worker.Region)
	if err != nil {
		return err
	}
	endpoint, err := resourceModel[n.PrivateEndpoint](values[1])
	if err != nil {
		return err
	}
	nic, err := resourceModel[n.Interface](values[2])
	if err != nil {
		return err
	}
	binding := azureworker.PrivateEndpointBinding{Endpoint: e.Endpoint, Interface: e.Interface, Service: v.Backend.Environment, Group: "managedEnvironment", Subnet: e.Subnet, Region: s.Worker.Region, Members: map[string]string{"managedEnvironment": v.Backend.Address}}
	if err := binding.Verify(endpoint, nic); err != nil {
		return err
	}
	if nic.Properties.IPConfigurations[0].Properties.PrivateLinkConnectionProperties.RequiredMemberName == nil || *nic.Properties.IPConfigurations[0].Properties.PrivateLinkConnectionProperties.RequiredMemberName != member {
		return hostedexecution.ErrOwnership
	}
	for _, configuration := range endpoint.Properties.IPConfigurations {
		if configuration.Properties.MemberName == nil || *configuration.Properties.MemberName != member {
			return hostedexecution.ErrOwnership
		}
	}
	subnet, err := resourceModel[n.Subnet](values[3])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateEndpointSubnet(subnet, []n.Interface{nic}, e.Subnet, e.SubnetPrefix, e.SecurityGroup, []string{e.Endpoint}); err != nil {
		return err
	}
	if err := verifyNetworkGroup(values[4], e.SecurityGroup, e.Subnet, s.Worker.Region, e.rules(s)); err != nil {
		return err
	}
	return s.verifyRelaySubnet(values[5])
}
