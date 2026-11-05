package azuresandbox

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKRelayNetwork) verify(values, rules []placementResource) error {
	if len(values) != 18 {
		return hostedexecution.ErrUnavailable
	}
	s := c.settings
	r, v := s.Placement.Services.RelayNetwork, s.Placement.Services
	vnet, err := resourceModel[n.VirtualNetwork](values[0])
	if err != nil {
		return err
	}
	subnets, err := resourceModels[n.Subnet](values[1:5])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyIsolatedVirtualNetwork(vnet, r.virtualNetwork(s), s.Worker.Region, r.Topology.Prefix, subnets); err != nil {
		return err
	}
	if err := s.verifyRelaySubnet(values[1]); err != nil {
		return err
	}
	if err := azureworker.VerifyRelaySubnetAttachments(subnets[0], r.SecurityGroup, r.RouteTable); err != nil {
		return err
	}
	if err := verifyNetworkGroup(values[5], r.SecurityGroup, v.Relay.Subnet, s.Worker.Region, r.rules(s)); err != nil {
		return err
	}
	routes, err := resourceModel[n.RouteTable](values[6])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyDefaultFirewallRoute(routes, r.RouteTable, v.Relay.Subnet, s.Worker.Region, "relay-egress", r.FirewallIP); err != nil {
		return err
	}
	endpoint, err := resourceModel[n.PrivateEndpoint](values[7])
	if err != nil {
		return err
	}
	nic, err := resourceModel[n.Interface](values[8])
	if err != nil {
		return err
	}
	binding := azureworker.PrivateEndpointBinding{Endpoint: r.Registry.Endpoint, Interface: r.Registry.Interface, Service: s.Placement.Registry, Group: "registry", Subnet: r.Registry.Subnet, Region: s.Worker.Region, Members: map[string]string{"registry": r.Registry.LoginIP, "registry_data_" + s.Worker.Region: r.Registry.DataIP}}
	if err := binding.Verify(endpoint, nic); err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateEndpointSubnet(subnets[2], []n.Interface{nic}, r.Registry.Subnet, r.Registry.SubnetPrefix, r.Registry.SecurityGroup, []string{r.Registry.Endpoint}); err != nil {
		return err
	}
	if err := verifyNetworkGroup(values[9], r.Registry.SecurityGroup, r.Registry.Subnet, s.Worker.Region, r.registryRules(s)); err != nil {
		return err
	}
	if err := c.verifyFirewall(values[10:13], subnets[3], rules); err != nil {
		return err
	}
	if err := s.verifyRegistry(values[13]); err != nil {
		return err
	}
	if !namedResource(values[14], r.Registry.Connection, "Microsoft.ContainerRegistry/registries/privateEndpointConnections") {
		return hostedexecution.ErrOwnership
	}
	connection, err := resourceModel[a.PrivateEndpointConnection](values[14])
	if err != nil {
		return err
	}
	if _, err := azureworker.VerifyRegistryEndpointConnections(s.Placement.Registry, []string{r.Registry.Endpoint}, []a.PrivateEndpointConnection{connection}); err != nil {
		return err
	}
	registry, err := resourceModel[a.Registry](values[13])
	if err != nil {
		return err
	}
	selected := []a.PrivateEndpointConnection{}
	for _, item := range registry.Properties.PrivateEndpointConnections {
		if item != nil && item.ID != nil && strings.EqualFold(*item.ID, r.Registry.Connection) {
			selected = append(selected, *item)
		}
	}
	if _, err := azureworker.VerifyRegistryEndpointConnections(s.Placement.Registry, []string{r.Registry.Endpoint}, selected); err != nil {
		return err
	}
	if err := s.verifyAuthorityIdentity(values[15], v.Relay.Pull.Resource, v.Relay.Pull.Client, v.Relay.Pull.Principal); err != nil {
		return err
	}
	if err := verifyAuthorityAssignment(values[16], r.Registry.PullAssignment, s.Placement.Registry, v.Relay.Pull.Principal, "/subscriptions/"+s.Worker.Subscription+pullRole); err != nil {
		return err
	}
	return s.verifyPullRole(values[17])
}

func (c *SDKRelayNetwork) verifyFirewall(values []placementResource, subnet n.Subnet, rules []placementResource) error {
	r, s := c.settings.Placement.Services.RelayNetwork, c.settings
	firewall, err := resourceModel[n.AzureFirewall](values[0])
	if err != nil {
		return err
	}
	public, err := resourceModel[n.PublicIPAddress](values[1])
	if err != nil {
		return err
	}
	policy, err := resourceModel[n.FirewallPolicy](values[2])
	if err != nil {
		return err
	}
	groups, err := resourceModels[n.FirewallPolicyRuleCollectionGroup](rules)
	if err != nil {
		return err
	}
	expected := azureworker.PlatformFirewallRules("relay", s.Placement.Services.BackendEndpoint.RelaySubnetPrefix)
	if err := r.firewallBinding(s).Verify(azureworker.FirewallObservation{Firewall: firewall, PublicIP: public, Policy: policy, Subnet: subnet, Groups: groups}, s.Worker.Region, "relay-platform", expected); err != nil {
		return err
	}
	props := subnet.Properties
	if len(props.IPConfigurations) != 1 || props.IPConfigurations[0] == nil || props.IPConfigurations[0].ID == nil || !strings.EqualFold(*props.IPConfigurations[0].ID, r.Topology.Firewall+"/azureFirewallIpConfigurations/egress") || len(props.IPConfigurationProfiles)+len(props.ResourceNavigationLinks)+len(props.ServiceAssociationLinks) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
