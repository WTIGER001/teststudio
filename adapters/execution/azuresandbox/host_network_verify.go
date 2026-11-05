package azuresandbox

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKHostNetwork) verify(values, groups []placementResource) error {
	if len(values) != 9 {
		return hostedexecution.ErrUnavailable
	}
	s, h, r := c.settings, c.settings.Host, c.settings.Host.Network
	vnet, err := resourceModel[n.VirtualNetwork](values[0])
	if err != nil {
		return err
	}
	subnets, err := resourceModels[n.Subnet](values[1:4])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyIsolatedVirtualNetwork(vnet, h.virtualNetwork(), s.Worker.Region, r.Topology.Prefix, subnets); err != nil {
		return err
	}
	if err := verifyServiceSubnet(values[1], h.Subnet, h.SubnetPrefix); err != nil {
		return err
	}
	if err := azureworker.VerifyRelaySubnetAttachments(subnets[0], r.SecurityGroup, r.RouteTable); err != nil {
		return err
	}
	if err := verifyNetworkGroup(values[4], r.SecurityGroup, h.Subnet, s.Worker.Region, h.networkRules()); err != nil {
		return err
	}
	routes, err := resourceModel[n.RouteTable](values[5])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyDefaultFirewallRoute(routes, r.RouteTable, h.Subnet, s.Worker.Region, "controller-egress", r.FirewallIP); err != nil {
		return err
	}
	firewall, err := resourceModel[n.AzureFirewall](values[6])
	if err != nil {
		return err
	}
	public, err := resourceModel[n.PublicIPAddress](values[7])
	if err != nil {
		return err
	}
	policy, err := resourceModel[n.FirewallPolicy](values[8])
	if err != nil {
		return err
	}
	rules, err := resourceModels[n.FirewallPolicyRuleCollectionGroup](groups)
	if err != nil {
		return err
	}
	if err := h.firewallBinding().Verify(azureworker.FirewallObservation{Firewall: firewall, PublicIP: public, Policy: policy, Subnet: subnets[2], Groups: rules}, s.Worker.Region, "controller-platform", h.firewallRules(s)); err != nil {
		return err
	}
	p := subnets[2].Properties
	if len(p.IPConfigurations) != 1 || p.IPConfigurations[0] == nil || p.IPConfigurations[0].ID == nil || !strings.EqualFold(*p.IPConfigurations[0].ID, r.Topology.Firewall+"/azureFirewallIpConfigurations/egress") || len(p.IPConfigurationProfiles)+len(p.ResourceNavigationLinks)+len(p.ServiceAssociationLinks) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
