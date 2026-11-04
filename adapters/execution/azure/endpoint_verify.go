package azureworker

import (
	"regexp"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

var endpointChildName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func endpointChild(id, name, parent, collection string) bool {
	return endpointChildName.MatchString(name) && strings.EqualFold(id, parent+"/"+collection+"/"+name)
}

func (g *EndpointGuard) verifyEndpoint(role endpointRole, actual n.PrivateEndpoint) error {
	return verifyPrivateEndpoint(role, actual, g.settings.Subnet, g.worker.Region)
}

func verifyPrivateEndpoint(role endpointRole, actual n.PrivateEndpoint, subnet, location string) error {
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), role.endpoint, "Microsoft.Network/privateEndpoints", value(actual.Location), location) || actual.Properties == nil || actual.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if len(p.NetworkInterfaces) == 0 || len(p.PrivateLinkServiceConnections) == 0 {
		return hostedexecution.ErrUnavailable
	}
	_, nicName := resourceParts(role.nic)
	if p.Subnet == nil ||
		!strings.EqualFold(value(p.Subnet.ID), subnet) ||
		len(p.NetworkInterfaces) != 1 ||
		p.NetworkInterfaces[0] == nil ||
		!strings.EqualFold(value(p.NetworkInterfaces[0].ID), role.nic) ||
		len(p.ApplicationSecurityGroups) != 0 ||
		len(p.ManualPrivateLinkServiceConnections) != 0 ||
		len(p.PrivateLinkServiceConnections) != 1 ||
		value(p.CustomNetworkInterfaceName) != "" && value(p.CustomNetworkInterfaceName) != nicName ||
		p.IPVersionType != nil && value(p.IPVersionType) != n.PrivateEndpointIPVersionTypeIPv4 {
		return hostedexecution.ErrOwnership
	}
	connection := p.PrivateLinkServiceConnections[0]
	if connection == nil ||
		connection.Properties == nil ||
		!endpointChildName.MatchString(value(connection.Name)) ||
		connection.ID != nil && !endpointChild(value(connection.ID), value(connection.Name), role.endpoint, "privateLinkServiceConnections") ||
		connection.Type != nil && !strings.EqualFold(value(connection.Type), "Microsoft.Network/privateEndpoints/privateLinkServiceConnections") {
		return hostedexecution.ErrOwnership
	}
	cp := connection.Properties
	if !strings.EqualFold(value(cp.PrivateLinkServiceID), role.service) || len(cp.GroupIDs) != 1 || value(cp.GroupIDs[0]) != role.group {
		return hostedexecution.ErrOwnership
	}
	if value(cp.ProvisioningState) != n.ProvisioningStateSucceeded || cp.PrivateLinkServiceConnectionState == nil {
		return hostedexecution.ErrUnavailable
	}
	state := cp.PrivateLinkServiceConnectionState
	switch value(state.Status) {
	case "Approved":
	case "", "Pending":
		return hostedexecution.ErrUnavailable
	default:
		return hostedexecution.ErrOwnership
	}
	if action := value(state.ActionsRequired); action != "" && action != "None" {
		return hostedexecution.ErrUnavailable
	}
	if len(p.IPConfigurations) > 0 {
		if len(p.IPConfigurations) != len(role.members) {
			return hostedexecution.ErrOwnership
		}
		seen := map[string]bool{}
		for _, ip := range p.IPConfigurations {
			if ip == nil || ip.Properties == nil {
				return hostedexecution.ErrOwnership
			}
			member, address := role.address(value(ip.Properties.MemberName), value(ip.Properties.GroupID))
			if address == "" || seen[member] || value(ip.Properties.PrivateIPAddress) != address {
				return hostedexecution.ErrOwnership
			}
			seen[member] = true
		}
	}
	return nil
}

func (r endpointRole) address(member, group string) (string, string) {
	if group != r.group || member == "" || len(member) > 128 {
		return "", ""
	}
	if r.group == "managedEnvironment" {
		return r.group, r.members[r.group]
	}
	return member, r.members[member]
}

func (g *EndpointGuard) verifyInterface(role endpointRole, actual n.Interface) error {
	return verifyPrivateInterface(role, actual, g.settings.Subnet, g.worker.Region)
}

func verifyPrivateInterface(role endpointRole, actual n.Interface, subnet, location string) error {
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), role.nic, "Microsoft.Network/networkInterfaces", value(actual.Location), location) || actual.Properties == nil || actual.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if p.PrivateEndpoint == nil ||
		!strings.EqualFold(value(p.PrivateEndpoint.ID), role.endpoint) ||
		value(p.EnableIPForwarding) ||
		value(p.EnableAcceleratedNetworking) ||
		value(p.DisableTCPStateTracking) ||
		value(p.DefaultOutboundConnectivityEnabled) ||
		p.NetworkSecurityGroup != nil ||
		p.VirtualMachine != nil ||
		p.PrivateLinkService != nil ||
		p.DscpConfiguration != nil ||
		len(p.HostedWorkloads)+len(p.TapConfigurations) != 0 ||
		p.NicType != nil && value(p.NicType) != n.NetworkInterfaceNicTypeStandard ||
		value(p.WorkloadType) != "" ||
		p.AuxiliaryMode != nil && value(p.AuxiliaryMode) != n.NetworkInterfaceAuxiliaryModeNone ||
		p.AuxiliarySKU != nil && value(p.AuxiliarySKU) != n.NetworkInterfaceAuxiliarySKUNone ||
		p.MigrationPhase != nil && value(p.MigrationPhase) != n.NetworkInterfaceMigrationPhaseNone ||
		p.DNSSettings != nil && (len(p.DNSSettings.DNSServers) > 0 ||
			value(p.DNSSettings.InternalDNSNameLabel) != "") {
		return hostedexecution.ErrOwnership
	}
	if len(p.IPConfigurations) < len(role.members) {
		return hostedexecution.ErrUnavailable
	}
	if len(p.IPConfigurations) != len(role.members) {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	names := map[string]bool{}
	for _, ip := range p.IPConfigurations {
		if ip == nil || ip.Properties == nil || !endpointChild(value(ip.ID), value(ip.Name), role.nic, "ipConfigurations") || names[value(ip.Name)] {
			return hostedexecution.ErrOwnership
		}
		names[value(ip.Name)] = true
		cfg := ip.Properties
		if value(cfg.ProvisioningState) != n.ProvisioningStateSucceeded {
			return hostedexecution.ErrUnavailable
		}
		if cfg.PrivateLinkConnectionProperties == nil {
			return hostedexecution.ErrUnavailable
		}
		link := cfg.PrivateLinkConnectionProperties
		member, address := role.address(value(link.RequiredMemberName), value(link.GroupID))
		if address == "" ||
			seen[member] ||
			value(cfg.PrivateIPAddress) != address ||
			value(cfg.PrivateIPAddressVersion) != n.IPVersionIPv4 ||
			cfg.PrivateIPAddressPrefixLength != nil ||
			cfg.Subnet == nil ||
			!strings.EqualFold(value(cfg.Subnet.ID), subnet) ||
			cfg.PublicIPAddress != nil ||
			cfg.GatewayLoadBalancer != nil ||
			len(cfg.ApplicationGatewayBackendAddressPools)+len(cfg.ApplicationSecurityGroups)+len(cfg.LoadBalancerBackendAddressPools)+len(cfg.LoadBalancerInboundNatRules)+len(cfg.VirtualNetworkTaps) != 0 ||
			value(cfg.PrivateIPAllocationMethod) != n.IPAllocationMethodDynamic && value(cfg.PrivateIPAllocationMethod) != n.IPAllocationMethodStatic {
			return hostedexecution.ErrOwnership
		}
		seen[member] = true
	}
	return nil
}
