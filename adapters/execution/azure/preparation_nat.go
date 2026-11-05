package azureworker

import (
	"net/netip"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s PreparationSettings) PreparationNATDefinition() (n.NatGateway, error) {
	if err := s.ValidateNative(); err != nil {
		return n.NatGateway{}, err
	}
	return n.NatGateway{Location: to.Ptr(s.Worker.Region), SKU: &n.NatGatewaySKU{Name: to.Ptr(n.NatGatewaySKUNameStandard)}, Properties: &n.NatGatewayPropertiesFormat{IdleTimeoutInMinutes: to.Ptr[int32](10), PublicIPAddresses: []*n.SubResource{{ID: to.Ptr(s.Network.NATPublicIP)}}}}, nil
}

func (s PreparationSettings) PreparationPublicIPDefinition() (n.PublicIPAddress, error) {
	if err := s.ValidateNative(); err != nil {
		return n.PublicIPAddress{}, err
	}
	return n.PublicIPAddress{Location: to.Ptr(s.Worker.Region), SKU: &n.PublicIPAddressSKU{Name: to.Ptr(n.PublicIPAddressSKUNameStandard), Tier: to.Ptr(n.PublicIPAddressSKUTierRegional)}, Properties: &n.PublicIPAddressPropertiesFormat{PublicIPAllocationMethod: to.Ptr(n.IPAllocationMethodStatic), PublicIPAddressVersion: to.Ptr(n.IPVersionIPv4)}}, nil
}

func verifyPreparationNAT(s PreparationSettings, nat n.NatGateway, ip n.PublicIPAddress) error {
	p := s.Network
	if !perimeterIdentity(value(nat.ID), value(nat.Name), value(nat.Type), p.NATGateway, "Microsoft.Network/natGateways", value(nat.Location), s.Worker.Region) || !perimeterIdentity(value(ip.ID), value(ip.Name), value(ip.Type), p.NATPublicIP, "Microsoft.Network/publicIPAddresses", value(ip.Location), s.Worker.Region) || nat.Properties == nil || nat.SKU == nil || ip.Properties == nil || ip.SKU == nil {
		return hostedexecution.ErrOwnership
	}
	np, pp := nat.Properties, ip.Properties
	if value(np.ProvisioningState) != n.ProvisioningStateSucceeded || value(pp.ProvisioningState) != n.ProvisioningStateSucceeded || pp.IPAddress == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(nat.SKU.Name) != n.NatGatewaySKUNameStandard || value(np.IdleTimeoutInMinutes) != 10 || len(np.PublicIPAddresses) != 1 || np.PublicIPAddresses[0] == nil || !strings.EqualFold(value(np.PublicIPAddresses[0].ID), p.NATPublicIP) || len(np.PublicIPAddressesV6)+len(np.PublicIPPrefixes)+len(np.PublicIPPrefixesV6) != 0 || np.SourceVirtualNetwork != nil || len(np.Subnets) != 1 || np.Subnets[0] == nil || !strings.EqualFold(value(np.Subnets[0].ID), s.Writer.Registry.SourceSubnet) {
		return hostedexecution.ErrOwnership
	}
	address, err := netip.ParseAddr(value(pp.IPAddress))
	if err != nil || !address.Is4() || !address.IsGlobalUnicast() || address.IsPrivate() || value(ip.SKU.Name) != n.PublicIPAddressSKUNameStandard || value(ip.SKU.Tier) != n.PublicIPAddressSKUTierRegional || value(pp.PublicIPAllocationMethod) != n.IPAllocationMethodStatic || value(pp.PublicIPAddressVersion) != n.IPVersionIPv4 || pp.NatGateway == nil || !strings.EqualFold(value(pp.NatGateway.ID), p.NATGateway) || pp.IPConfiguration != nil || pp.PublicIPPrefix != nil || pp.LinkedPublicIPAddress != nil || pp.ServicePublicIPAddress != nil || len(pp.IPTags) != 0 || pp.DNSSettings != nil || ip.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	return nil
}
