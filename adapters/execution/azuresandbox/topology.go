package azuresandbox

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKTopology struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func topologyIDs(s Settings) []string {
	p, t := s.Placement, s.Placement.Network.Topology
	return []string{p.virtualNetwork(), p.Subnet, p.Private.Subnet, p.firewallSubnet(), t.Firewall, t.PublicIP, t.Policy}
}

func topologyVersions(s Settings) map[string]string {
	versions := map[string]string{}
	for _, id := range append(topologyIDs(s), s.Placement.Network.Topology.Policy+"/ruleCollectionGroups") {
		versions[id] = "2025-03-01"
	}
	return versions
}

func NewSDKTopology(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKTopology, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(topologyVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	reader.http.pages = map[string]bool{s.Placement.Network.Topology.Policy + "/ruleCollectionGroups": true}
	return &SDKTopology{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKTopology) Close() { c.http.Close() }

func (c *SDKTopology) Check(ctx context.Context, config Config) error {
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

func (c *SDKTopology) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKTopology) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var values, rules []placementResource
	var errors [2]error
	var group sync.WaitGroup
	group.Go(func() { values, errors[0] = c.observe(ctx, topologyIDs(c.settings)) })
	group.Go(func() {
		rules, errors[1] = c.inventoryPages(ctx, c.settings.Placement.Network.Topology.Policy+"/ruleCollectionGroups", 4, 4)
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
	return c.verify(values, rules)
}

func (c *SDKTopology) verify(values, rules []placementResource) error {
	p := c.settings.Placement
	if len(values) != 7 {
		return hostedexecution.ErrUnavailable
	}
	vnet, err := resourceModel[n.VirtualNetwork](values[0])
	if err != nil {
		return err
	}
	subnets, err := resourceModels[n.Subnet](values[1:4])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyIsolatedVirtualNetwork(vnet, p.virtualNetwork(), c.settings.Worker.Region, p.Network.Topology.Prefix, subnets); err != nil {
		return err
	}
	firewall, err := resourceModel[n.AzureFirewall](values[4])
	if err != nil {
		return err
	}
	public, err := resourceModel[n.PublicIPAddress](values[5])
	if err != nil {
		return err
	}
	policy, err := resourceModel[n.FirewallPolicy](values[6])
	if err != nil {
		return err
	}
	groups, err := resourceModels[n.FirewallPolicyRuleCollectionGroup](rules)
	if err != nil {
		return err
	}
	if err := p.firewallBinding().Verify(azureworker.FirewallObservation{Firewall: firewall, PublicIP: public, Policy: policy, Subnet: subnets[2], Groups: groups}, c.settings.Worker.Region, sandboxFirewallGroup, sandboxFirewallRules()); err != nil {
		return err
	}
	props := subnets[2].Properties
	if len(props.IPConfigurations) != 1 || props.IPConfigurations[0] == nil || props.IPConfigurations[0].ID == nil || !strings.EqualFold(*props.IPConfigurations[0].ID, p.Network.Topology.Firewall+"/azureFirewallIpConfigurations/egress") || len(props.IPConfigurationProfiles)+len(props.ResourceNavigationLinks)+len(props.ServiceAssociationLinks) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
