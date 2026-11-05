package azureworker

import (
	"net/netip"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyPreparationRegistryNetwork(c Config, s PreparationRegistrySettings, network n.VirtualNetwork, endpoint, source n.Subnet) error {
	if err := networkMatches(network, s.network(), c.Region); err != nil {
		return err
	}
	if err := subnetMatches(source, s.SourceSubnet); err != nil {
		return err
	}
	prefixes := subnetPrefixes(source)
	if len(prefixes) != 1 || prefixes[0] != s.SourceSubnetPrefix || len(source.Properties.PrivateEndpoints) != 0 {
		return hostedexecution.ErrOwnership
	}
	if len(network.Properties.AddressSpace.AddressPrefixes) == 0 || len(network.Properties.AddressSpace.AddressPrefixes) > 8 || len(network.Properties.Subnets) > 256 {
		return hostedexecution.ErrUnavailable
	}
	spaces := []netip.Prefix{}
	seen := map[string]bool{}
	worker, _ := netip.ParsePrefix(s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNetPrefix)
	for _, entry := range network.Properties.AddressSpace.AddressPrefixes {
		space, valid := privatePrefix(value(entry))
		if !valid || space.Overlaps(worker) || seen[space.String()] {
			return hostedexecution.ErrOwnership
		}
		seen[space.String()] = true
		spaces = append(spaces, space)
	}
	if !registryStrings(network.Properties.AddressSpace.AddressPrefixes, s.ControllerPrefixes...) {
		return hostedexecution.ErrOwnership
	}
	desired := []string{s.SubnetPrefix, s.SourceSubnetPrefix}
	for _, raw := range desired {
		prefix, _ := netip.ParsePrefix(raw)
		found := false
		for _, space := range spaces {
			if containsPrefix(space, prefix) {
				found = true
			}
		}
		if !found {
			return hostedexecution.ErrOwnership
		}
	}
	core := s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerSubnet
	coreSeen := false
	for _, subnet := range network.Properties.Subnets {
		if subnet == nil {
			return hostedexecution.ErrUnavailable
		}
		if !strings.EqualFold(value(subnet.ID), core) {
			continue
		}
		if coreSeen || subnet.Properties == nil {
			return hostedexecution.ErrUnavailable
		}
		coreSeen = true
		prefixes := subnetPrefixes(*subnet)
		if len(prefixes) != 1 {
			return hostedexecution.ErrUnavailable
		}
		prefix, valid := privatePrefix(prefixes[0])
		if !valid {
			return hostedexecution.ErrUnavailable
		}
		for _, raw := range desired {
			desired, _ := netip.ParsePrefix(raw)
			if prefix.Overlaps(desired) {
				return hostedexecution.ErrOwnership
			}
		}
	}
	if !coreSeen || !embeddedSubnet(network, s.Subnet, endpoint) || !embeddedSubnet(network, s.SourceSubnet, source) {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
