package azureworker

import (
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func VerifyIsolatedVirtualNetwork(actual n.VirtualNetwork, id, region, prefix string, subnets []n.Subnet) error {
	if err := networkMatches(actual, id, region); err != nil {
		return err
	}
	p := actual.Properties
	if len(p.AddressSpace.AddressPrefixes) != 1 || value(p.AddressSpace.AddressPrefixes[0]) != prefix || len(p.VirtualNetworkPeerings) != 0 || p.DefaultPublicNatGateway != nil || len(p.IPAllocations) != 0 || p.BgpCommunities != nil || p.DhcpOptions != nil && len(p.DhcpOptions.DNSServers) != 0 || len(p.Subnets) != len(subnets) {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, subnet := range subnets {
		id := value(subnet.ID)
		if id == "" || seen[id] || !embeddedSubnet(actual, id, subnet) {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
	}
	return nil
}
