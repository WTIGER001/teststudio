package azuresandbox

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKRelayNetwork struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func relayNetworkIDs(s Settings) []string {
	r, v := s.Placement.Services.RelayNetwork, s.Placement.Services
	return []string{r.virtualNetwork(s), v.Relay.Subnet, v.BackendEndpoint.Subnet, r.Registry.Subnet, r.virtualNetwork(s) + "/subnets/AzureFirewallSubnet", r.SecurityGroup, r.RouteTable, r.Registry.Endpoint, r.Registry.Interface, r.Registry.SecurityGroup, r.Topology.Firewall, r.Topology.PublicIP, r.Topology.Policy, s.Placement.Registry, r.Registry.Connection, v.Relay.Pull.Resource, r.Registry.PullAssignment, "/subscriptions/" + s.Worker.Subscription + pullRole}
}

func relayNetworkVersions(s Settings) map[string]string {
	r, v := s.Placement.Services.RelayNetwork, s.Placement.Services
	versions := map[string]string{}
	for _, id := range relayNetworkIDs(s) {
		versions[id] = "2025-03-01"
	}
	versions[r.Topology.Policy+"/ruleCollectionGroups"] = "2025-03-01"
	versions[s.Placement.Registry], versions[r.Registry.Connection] = "2025-11-01", "2025-11-01"
	versions[v.Relay.Pull.Resource], versions[r.Registry.PullAssignment], versions["/subscriptions/"+s.Worker.Subscription+pullRole] = "2024-11-30", "2022-04-01", "2022-05-01-preview"
	for _, id := range []string{r.Resolver.Policy, r.Resolver.AllowList, r.Resolver.BlockList, r.Resolver.Policy + "/dnsSecurityRules", r.Resolver.Policy + "/virtualNetworkLinks", r.virtualNetwork(s) + "/listDnsResolverPolicies"} {
		versions[id] = "2025-05-01"
	}
	for _, suffix := range []string{"", "/ALL", "/virtualNetworkLinks"} {
		versions[r.Registry.Zone+suffix] = "2024-06-01"
	}
	return versions
}

func NewSDKRelayNetwork(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKRelayNetwork, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(relayNetworkVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	r := s.Placement.Services.RelayNetwork
	reader.http.pages = map[string]bool{r.Topology.Policy + "/ruleCollectionGroups": true, r.Resolver.Policy + "/dnsSecurityRules": true, r.Resolver.Policy + "/virtualNetworkLinks": true, r.Registry.Zone + "/ALL": true, r.Registry.Zone + "/virtualNetworkLinks": true}
	reader.http.posts = map[string]bool{r.virtualNetwork(s) + "/listDnsResolverPolicies": true}
	return &SDKRelayNetwork{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKRelayNetwork) Close() { c.http.Close() }

func (c *SDKRelayNetwork) Check(ctx context.Context, config Config) error {
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

func (c *SDKRelayNetwork) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKRelayNetwork) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	r := c.settings.Placement.Services.RelayNetwork
	var values, groups []placementResource
	var errors [4]error
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, relayNetworkIDs(c.settings)) })
	group.Go(func() { groups, errors[1] = c.inventoryPages(ctx, r.Topology.Policy+"/ruleCollectionGroups", 4, 4) })
	group.Go(func() { errors[2] = c.readResolverPolicy(ctx, r.resolverBinding(c.settings), c.settings.Worker.Region) })
	group.Go(func() { errors[3] = c.readPrivateZone(ctx, r.zoneBinding(c.settings)) })
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return c.verify(values, groups)
}
