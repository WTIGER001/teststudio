package azureworker

import (
	"net/netip"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *TopologyGuard) workerAddress(raw string) error {
	if raw == "" {
		return hostedexecution.ErrUnavailable
	}
	ip, err := netip.ParseAddr(raw)
	prefix, _ := netip.ParsePrefix(g.settings.WorkerSubnetPrefix)
	if err != nil || !ip.Is4() || !prefix.Contains(ip) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func networkMatches(v n.VirtualNetwork, id, location string) error {
	_, name := resourceParts(id)
	if !strings.EqualFold(value(v.ID), id) || value(v.Name) != name || !strings.EqualFold(value(v.Type), "Microsoft.Network/virtualNetworks") || value(v.Location) != location || v.Properties == nil || v.Properties.AddressSpace == nil || v.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	if value(v.Properties.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (g *TopologyGuard) networks(worker, controller n.VirtualNetwork) error {
	s := g.settings
	if err := networkMatches(worker, s.WorkerVNet, g.worker.Region); err != nil {
		return err
	}
	if err := networkMatches(controller, s.ControllerVNet, g.worker.Region); err != nil {
		return err
	}
	p := worker.Properties
	if len(p.AddressSpace.AddressPrefixes) != 1 || value(p.AddressSpace.AddressPrefixes[0]) != s.WorkerVNetPrefix || len(p.VirtualNetworkPeerings) != 0 || p.DefaultPublicNatGateway != nil || len(p.IPAllocations) != 0 || p.BgpCommunities != nil || p.DhcpOptions != nil && len(p.DhcpOptions.DNSServers) != 0 {
		return hostedexecution.ErrOwnership
	}
	prefixes := controller.Properties.AddressSpace.AddressPrefixes
	if len(prefixes) == 0 || len(prefixes) > 8 {
		return hostedexecution.ErrOwnership
	}
	workerPrefix, _ := netip.ParsePrefix(s.WorkerVNetPrefix)
	seen := map[string]bool{}
	for _, raw := range prefixes {
		prefix, valid := privatePrefix(value(raw))
		if !valid || prefix.Overlaps(workerPrefix) || seen[prefix.String()] {
			return hostedexecution.ErrOwnership
		}
		seen[prefix.String()] = true
	}
	return nil
}

func subnetPrefixes(s n.Subnet) []string {
	if s.Properties == nil {
		return nil
	}
	p := s.Properties
	if p.AddressPrefix != nil && len(p.AddressPrefixes) != 0 {
		return nil
	}
	if p.AddressPrefix != nil {
		return []string{value(p.AddressPrefix)}
	}
	var result []string
	for _, raw := range p.AddressPrefixes {
		result = append(result, value(raw))
	}
	return result
}

func subnetMatches(s n.Subnet, id string) error {
	parts := strings.Split(id, "/")
	if !strings.EqualFold(value(s.ID), id) || value(s.Name) != parts[10] || !strings.EqualFold(value(s.Type), "Microsoft.Network/virtualNetworks/subnets") || s.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	if value(s.Properties.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (g *TopologyGuard) subnets(worker, controller n.VirtualNetwork, wsub, csub n.Subnet) error {
	s := g.settings
	if err := subnetMatches(wsub, s.WorkerSubnet); err != nil {
		return err
	}
	if err := subnetMatches(csub, s.ControllerSubnet); err != nil {
		return err
	}
	p := wsub.Properties
	prefixes := subnetPrefixes(wsub)
	if len(prefixes) != 1 || prefixes[0] != s.WorkerSubnetPrefix || p.DefaultOutboundAccess == nil || value(p.DefaultOutboundAccess) || p.NatGateway != nil || p.NetworkSecurityGroup == nil || !strings.EqualFold(value(p.NetworkSecurityGroup.ID), s.SecurityGroup) || p.RouteTable == nil || !strings.EqualFold(value(p.RouteTable.ID), s.RouteTable) || len(p.ServiceEndpoints)+len(p.ServiceEndpointPolicies)+len(p.PrivateEndpoints)+len(p.ApplicationGatewayIPConfigurations)+len(p.IPAllocations)+len(p.IpamPoolPrefixAllocations) != 0 || value(p.SharingScope) != "" || len(p.Delegations) != 1 || p.Delegations[0] == nil || p.Delegations[0].Properties == nil || value(p.Delegations[0].Properties.ServiceName) != "Microsoft.App/environments" {
		return hostedexecution.ErrOwnership
	}
	if value(p.Delegations[0].Properties.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if !embeddedSubnet(worker, s.WorkerSubnet, wsub) || !embeddedSubnet(controller, s.ControllerSubnet, csub) {
		return hostedexecution.ErrOwnership
	}
	cp := subnetPrefixes(csub)
	if len(cp) != 1 {
		return hostedexecution.ErrOwnership
	}
	prefix, ok := privatePrefix(cp[0])
	if !ok {
		return hostedexecution.ErrOwnership
	}
	contained := false
	for _, raw := range controller.Properties.AddressSpace.AddressPrefixes {
		parent, _ := netip.ParsePrefix(value(raw))
		contained = contained || parent.Bits() <= prefix.Bits() && parent.Contains(prefix.Addr())
	}
	if !contained {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func embeddedSubnet(v n.VirtualNetwork, id string, expected n.Subnet) bool {
	count := 0
	for _, s := range v.Properties.Subnets {
		if s == nil {
			return false
		}
		if strings.EqualFold(value(s.ID), id) {
			count++
			if fingerprint(document(s)) != fingerprint(document(expected)) {
				return false
			}
		}
	}
	return count == 1
}
