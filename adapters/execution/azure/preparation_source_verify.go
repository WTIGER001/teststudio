package azureworker

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyPreparationSource(s PreparationSettings, o preparationNetworkObservation) error {
	p, r := s.Network, s.Writer.Registry
	if err := subnetMatches(o.source, r.SourceSubnet); err != nil {
		return err
	}
	sp := o.source.Properties
	prefixes := subnetPrefixes(o.source)
	if sp.DefaultOutboundAccess == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(prefixes) != 1 || prefixes[0] != r.SourceSubnetPrefix || value(sp.DefaultOutboundAccess) || sp.NetworkSecurityGroup == nil || !strings.EqualFold(value(sp.NetworkSecurityGroup.ID), p.SourceSecurityGroup) || sp.NatGateway == nil || !strings.EqualFold(value(sp.NatGateway.ID), p.NATGateway) || sp.RouteTable != nil || value(sp.SharingScope) != "" || len(sp.ServiceEndpoints)+len(sp.ServiceEndpointPolicies)+len(sp.PrivateEndpoints)+len(sp.ApplicationGatewayIPConfigurations)+len(sp.IPAllocations)+len(sp.IpamPoolPrefixAllocations) != 0 || len(sp.Delegations) != 1 || sp.Delegations[0] == nil || sp.Delegations[0].Properties == nil || value(sp.Delegations[0].Properties.ServiceName) != "Microsoft.App/environments" {
		return hostedexecution.ErrOwnership
	}
	if value(sp.Delegations[0].Properties.ProvisioningState) != "Succeeded" {
		return hostedexecution.ErrUnavailable
	}
	if err := verifyEndpointSecurityGroup(o.sourceNSG, p.SourceSecurityGroup, r.SourceSubnet, s.Worker.Region, s.sourceRules()); err != nil {
		return err
	}
	if err := networkMatches(o.network, r.network(), s.Worker.Region); err != nil {
		return err
	}
	v := o.network.Properties
	if !registryStrings(v.AddressSpace.AddressPrefixes, r.ControllerPrefixes...) || v.DhcpOptions != nil && len(v.DhcpOptions.DNSServers) != 0 || v.DefaultPublicNatGateway != nil || len(v.Subnets) > 256 {
		return hostedexecution.ErrOwnership
	}
	if !embeddedSubnet(o.network, r.SourceSubnet, o.source) || !embeddedSubnet(o.network, p.FileSubnet, o.file) {
		return hostedexecution.ErrUnavailable
	}
	file, _ := netip.ParsePrefix(p.FileSubnetPrefix)
	seen := map[string]bool{}
	for _, subnet := range v.Subnets {
		if subnet == nil || subnet.ID == nil || seen[strings.ToLower(value(subnet.ID))] {
			return hostedexecution.ErrUnavailable
		}
		seen[strings.ToLower(value(subnet.ID))] = true
		if strings.EqualFold(value(subnet.ID), p.FileSubnet) {
			continue
		}
		prefixes := subnetPrefixes(*subnet)
		if len(prefixes) == 0 {
			return hostedexecution.ErrUnavailable
		}
		for _, raw := range prefixes {
			prefix, valid := privatePrefix(raw)
			if !valid || prefix.Overlaps(file) {
				return hostedexecution.ErrOwnership
			}
		}
	}
	return nil
}
