package azuresandbox

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKHostRegistry) verify(values []placementResource) error {
	if len(values) != 10 {
		return hostedexecution.ErrUnavailable
	}
	s, h, r := c.settings, c.settings.Host, c.settings.Host.Registry
	if err := verifyServiceSubnet(values[0], h.Subnet, h.SubnetPrefix); err != nil {
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
	binding := azureworker.PrivateEndpointBinding{Endpoint: r.Endpoint, Interface: r.Interface, Service: s.Placement.Registry, Group: "registry", Subnet: r.Subnet, Region: s.Worker.Region, Members: map[string]string{"registry": r.LoginIP, "registry_data_" + s.Worker.Region: r.DataIP}}
	if err := binding.Verify(endpoint, nic); err != nil {
		return err
	}
	subnet, err := resourceModel[n.Subnet](values[3])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateEndpointSubnet(subnet, []n.Interface{nic}, r.Subnet, r.SubnetPrefix, r.SecurityGroup, []string{r.Endpoint}); err != nil {
		return err
	}
	if err := verifyNetworkGroup(values[4], r.SecurityGroup, r.Subnet, s.Worker.Region, h.registryRules()); err != nil {
		return err
	}
	if err := s.verifyRegistry(values[5]); err != nil {
		return err
	}
	if !namedResource(values[6], r.Connection, "Microsoft.ContainerRegistry/registries/privateEndpointConnections") {
		return hostedexecution.ErrOwnership
	}
	connection, err := resourceModel[a.PrivateEndpointConnection](values[6])
	if err != nil {
		return err
	}
	if _, err := azureworker.VerifyRegistryEndpointConnections(s.Placement.Registry, []string{r.Endpoint}, []a.PrivateEndpointConnection{connection}); err != nil {
		return err
	}
	registry, err := resourceModel[a.Registry](values[5])
	if err != nil {
		return err
	}
	selected := []a.PrivateEndpointConnection{}
	for _, item := range registry.Properties.PrivateEndpointConnections {
		if item != nil && item.ID != nil && strings.EqualFold(*item.ID, r.Connection) {
			selected = append(selected, *item)
		}
	}
	if _, err := azureworker.VerifyRegistryEndpointConnections(s.Placement.Registry, []string{r.Endpoint}, selected); err != nil {
		return err
	}
	if err := s.verifyAuthorityIdentity(values[7], h.Pull.Resource, h.Pull.Client, h.Pull.Principal); err != nil {
		return err
	}
	if err := verifyAuthorityAssignment(values[8], r.PullAssignment, s.Placement.Registry, h.Pull.Principal, "/subscriptions/"+s.Worker.Subscription+pullRole); err != nil {
		return err
	}
	return s.verifyPullRole(values[9])
}
