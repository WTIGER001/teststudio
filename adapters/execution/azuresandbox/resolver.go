package azuresandbox

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKResolver struct {
	*metadataReader
	inventory *azureworker.SDKPrivateDNSInventoryReader
	settings  Settings
	cache     metadataCache
	now       func() time.Time
}

type resolverInventoryTransport struct{ owner *placementHTTP }

func (t resolverInventoryTransport) Do(r *http.Request) (*http.Response, error) {
	if t.owner.closed.Load() || r.Context().Err() != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	return t.owner.readBounded(r)
}

func resolverVersions(s Settings) map[string]string {
	r := s.Placement.Network.Resolver
	versions := map[string]string{}
	for _, id := range []string{r.Policy, r.AllowList, r.BlockList, r.Policy + "/dnsSecurityRules", r.Policy + "/virtualNetworkLinks", s.Placement.virtualNetwork() + "/listDnsResolverPolicies"} {
		versions[id] = "2025-05-01"
	}
	return versions
}

func NewSDKResolver(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKResolver, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(resolverVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	r := s.Placement.Network.Resolver
	reader.http.pages = map[string]bool{r.Policy + "/dnsSecurityRules": true, r.Policy + "/virtualNetworkLinks": true}
	reader.http.posts = map[string]bool{s.Placement.virtualNetwork() + "/listDnsResolverPolicies": true}
	inventory, err := azureworker.NewBoundPrivateDNSInventoryReader(s.Worker.Subscription, credential, resolverInventoryTransport{reader.http})
	if err != nil {
		reader.http.Close()
		return nil, err
	}
	return &SDKResolver{metadataReader: reader, inventory: inventory, settings: s, now: time.Now}, nil
}

func (c *SDKResolver) Close() {
	c.http.Close()
	c.inventory.Close()
}

func (c *SDKResolver) Check(ctx context.Context, config Config) error {
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

func (c *SDKResolver) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKResolver) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var errors [2]error
	var group sync.WaitGroup
	group.Go(func() { errors[0] = c.readPolicy(ctx) })
	group.Go(func() {
		actual, err := c.inventory.Observe(ctx)
		if err != nil {
			errors[1] = err
			return
		}
		p := c.settings.Placement
		wanted := map[string]string{p.Private.RegistryZone + "/virtualNetworkLinks/sandbox-network": p.virtualNetwork(), p.Private.ExchangeZone + "/virtualNetworkLinks/sandbox-network": p.virtualNetwork()}
		errors[1] = azureworker.VerifyPrivateDNSInventoryLinks(c.settings.Worker.Subscription, actual, wanted, []string{p.virtualNetwork()})
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
	return nil
}

func (c *SDKResolver) readPolicy(ctx context.Context) error {
	r := c.settings.Placement.Network.Resolver
	var values, rules, links []placementResource
	var attached struct {
		Value    *[]d.SubResource `json:"value"`
		NextLink string           `json:"nextLink"`
	}
	var errors [4]error
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, []string{r.Policy, r.AllowList, r.BlockList}) })
	group.Go(func() { rules, errors[1] = c.inventoryPages(ctx, r.Policy+"/dnsSecurityRules", 2, 2) })
	group.Go(func() { links, errors[2] = c.inventoryPages(ctx, r.Policy+"/virtualNetworkLinks", 1, 2) })
	group.Go(func() {
		errors[3] = c.documentID(ctx, c.settings.Placement.virtualNetwork()+"/listDnsResolverPolicies", &attached)
	})
	group.Wait()
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	if attached.Value == nil || attached.NextLink != "" {
		return hostedexecution.ErrUnavailable
	}
	policy, err := resourceModel[d.Policy](values[0])
	if err != nil {
		return err
	}
	lists, err := resourceModels[d.DomainList](values[1:])
	if err != nil {
		return err
	}
	ruleModels, err := resourceModels[d.DNSSecurityRule](rules)
	if err != nil {
		return err
	}
	linkModels, err := resourceModels[d.PolicyVirtualNetworkLink](links)
	if err != nil {
		return err
	}
	return r.binding(c.settings).Verify(c.settings.Worker.Region, azureworker.ResolverObservation{Policy: policy, Allow: lists[0], Block: lists[1], Rules: ruleModels, Links: linkModels, Attached: *attached.Value})
}
