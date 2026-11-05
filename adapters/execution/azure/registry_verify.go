package azureworker

import (
	"slices"
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func registryStrings(actual []*string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	values := make([]string, 0, len(actual))
	for _, entry := range actual {
		if entry == nil {
			return false
		}
		values = append(values, *entry)
	}
	slices.Sort(values)
	expected = slices.Clone(expected)
	slices.Sort(expected)
	return slices.Equal(values, expected)
}

func (s RegistryServiceBinding) verifyRegistry(actual a.Registry) error {
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), s.Registry, "Microsoft.ContainerRegistry/registries", value(actual.Location), s.Region) ||
		actual.Properties == nil ||
		actual.SKU == nil ||
		value(actual.SKU.Name) != a.SKUNamePremium ||
		actual.SKU.Tier != nil && value(actual.SKU.Tier) != a.SKUTierPremium {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != a.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if p.AdminUserEnabled == nil ||
		p.AnonymousPullEnabled == nil ||
		p.DataEndpointEnabled == nil ||
		p.NetworkRuleBypassAllowedForTasks == nil ||
		p.PublicNetworkAccess == nil ||
		p.NetworkRuleBypassOptions == nil ||
		p.RoleAssignmentMode == nil ||
		p.NetworkRuleSet == nil ||
		p.NetworkRuleSet.DefaultAction == nil ||
		p.Policies == nil ||
		p.Policies.ExportPolicy == nil ||
		p.Policies.ExportPolicy.Status == nil ||
		p.Policies.AzureADAuthenticationAsArmPolicy == nil ||
		p.Policies.AzureADAuthenticationAsArmPolicy.Status == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(p.AdminUserEnabled) ||
		value(p.AnonymousPullEnabled) ||
		!value(p.DataEndpointEnabled) ||
		value(p.NetworkRuleBypassAllowedForTasks) ||
		value(p.PublicNetworkAccess) != a.PublicNetworkAccessDisabled ||
		value(p.NetworkRuleBypassOptions) != a.NetworkRuleBypassOptionsNone ||
		value(p.RoleAssignmentMode) != a.RoleAssignmentModeLegacyRegistryPermissions ||
		value(p.NetworkRuleSet.DefaultAction) != a.DefaultActionDeny ||
		len(p.NetworkRuleSet.IPRules) != 0 ||
		value(p.Policies.ExportPolicy.Status) != a.ExportPolicyStatusDisabled ||
		value(p.Policies.AzureADAuthenticationAsArmPolicy.Status) != a.AzureADAuthenticationAsArmPolicyStatusEnabled ||
		value(p.LoginServer) != s.Login {
		return hostedexecution.ErrOwnership
	}
	if len(p.DataEndpointHostNames) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if !registryStrings(p.DataEndpointHostNames, strings.TrimSuffix(s.Login, ".azurecr.io")+"."+s.Region+".data.azurecr.io") {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func VerifyRegistryEndpointConnections(registry string, endpoints []string, connections []a.PrivateEndpointConnection) (map[string]string, error) {
	if len(connections) < len(endpoints) {
		return nil, hostedexecution.ErrUnavailable
	}
	if len(connections) != len(endpoints) {
		return nil, hostedexecution.ErrOwnership
	}
	wanted := map[string]bool{}
	for _, id := range endpoints {
		wanted[strings.ToLower(id)] = true
	}
	observed := map[string]string{}
	names := map[string]bool{}
	for _, connection := range connections {
		name := value(connection.Name)
		if !endpointChild(value(connection.ID), name, registry, "privateEndpointConnections") || !strings.EqualFold(value(connection.Type), "Microsoft.ContainerRegistry/registries/privateEndpointConnections") || names[strings.ToLower(name)] || connection.Properties == nil || connection.Properties.PrivateEndpoint == nil {
			return nil, hostedexecution.ErrOwnership
		}
		names[strings.ToLower(name)] = true
		p := connection.Properties
		endpoint := strings.ToLower(value(p.PrivateEndpoint.ID))
		if !wanted[endpoint] || observed[endpoint] != "" {
			return nil, hostedexecution.ErrOwnership
		}
		if value(p.ProvisioningState) != a.ProvisioningStateSucceeded || p.PrivateLinkServiceConnectionState == nil {
			return nil, hostedexecution.ErrUnavailable
		}
		state := p.PrivateLinkServiceConnectionState
		switch value(state.Status) {
		case a.ConnectionStatusApproved:
		case "", a.ConnectionStatusPending:
			return nil, hostedexecution.ErrUnavailable
		default:
			return nil, hostedexecution.ErrOwnership
		}
		if action := value(state.ActionsRequired); action != "" && action != a.ActionsRequiredNone {
			return nil, hostedexecution.ErrUnavailable
		}
		observed[endpoint] = strings.ToLower(value(connection.ID))
	}
	return observed, nil
}

func (s RegistryServiceBinding) verifyLinks(links []a.PrivateLinkResource) error {
	if len(links) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(links) != 1 {
		return hostedexecution.ErrOwnership
	}
	link := links[0]
	if !strings.EqualFold(value(link.ID), s.Registry+"/privateLinkResources/registry") || value(link.Name) != "registry" || !strings.EqualFold(value(link.Type), "Microsoft.ContainerRegistry/registries/privateLinkResources") || link.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := link.Properties
	if p.GroupID == nil || len(p.RequiredMembers) == 0 || len(p.RequiredZoneNames) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if value(p.GroupID) != "registry" || !registryStrings(p.RequiredMembers, "registry", "registry_data_"+s.Region) || !registryStrings(p.RequiredZoneNames, "privatelink.azurecr.io") {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s RegistryServiceBinding) verifyReplicas(replicas []a.Replication) error {
	if len(replicas) > 1 {
		return hostedexecution.ErrOwnership
	}
	for _, replica := range replicas {
		if !endpointChild(value(replica.ID), value(replica.Name), s.Registry, "replications") || !strings.EqualFold(value(replica.Type), "Microsoft.ContainerRegistry/registries/replications") || value(replica.Location) != s.Region || replica.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		if value(replica.Properties.ProvisioningState) != a.ProvisioningStateSucceeded || replica.Properties.RegionEndpointEnabled == nil {
			return hostedexecution.ErrUnavailable
		}
		if !value(replica.Properties.RegionEndpointEnabled) {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}
