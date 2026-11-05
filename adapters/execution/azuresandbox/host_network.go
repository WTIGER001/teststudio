package azuresandbox

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKHostNetwork struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func hostNetworkIDs(s Settings) []string {
	h, r := s.Host, s.Host.Network
	return []string{h.virtualNetwork(), h.Subnet, h.Registry.Subnet, h.firewallBinding().Subnet, r.SecurityGroup, r.RouteTable, r.Topology.Firewall, r.Topology.PublicIP, r.Topology.Policy}
}

func hostNetworkVersions(s Settings) map[string]string {
	result := map[string]string{}
	for _, id := range hostNetworkIDs(s) {
		result[id] = "2025-03-01"
	}
	r := s.Host.Network
	result[r.Topology.Policy+"/ruleCollectionGroups"] = "2025-03-01"
	for _, id := range []string{r.Resolver.Policy, r.Resolver.AllowList, r.Resolver.BlockList, r.Resolver.Policy + "/dnsSecurityRules", r.Resolver.Policy + "/virtualNetworkLinks", s.Host.virtualNetwork() + "/listDnsResolverPolicies"} {
		result[id] = "2025-05-01"
	}
	return result
}

func NewSDKHostNetwork(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKHostNetwork, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(hostNetworkVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	r := s.Host.Network
	reader.http.pages = map[string]bool{r.Topology.Policy + "/ruleCollectionGroups": true, r.Resolver.Policy + "/dnsSecurityRules": true, r.Resolver.Policy + "/virtualNetworkLinks": true}
	reader.http.posts = map[string]bool{s.Host.virtualNetwork() + "/listDnsResolverPolicies": true}
	return &SDKHostNetwork{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKHostNetwork) Close() { c.http.Close() }

func (c *SDKHostNetwork) Check(ctx context.Context, config Config) error {
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

func (c *SDKHostNetwork) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKHostNetwork) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var values, groups []placementResource
	var errors [3]error
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, hostNetworkIDs(c.settings)) })
	group.Go(func() {
		groups, errors[1] = c.inventoryPages(ctx, c.settings.Host.Network.Topology.Policy+"/ruleCollectionGroups", 4, 4)
	})
	group.Go(func() {
		errors[2] = c.readResolverPolicy(ctx, c.settings.Host.resolverBinding(c.settings), c.settings.Worker.Region)
	})
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
