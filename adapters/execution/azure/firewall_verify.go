package azureworker

import (
	"net/netip"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *FirewallGuard) verifyFirewall(actual n.AzureFirewall) error {
	s, c := g.settings, g.worker
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), s.Firewall, "Microsoft.Network/azureFirewalls", value(actual.Location), c.Region) || actual.Properties == nil || actual.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if len(p.IPConfigurations) != 1 || p.IPConfigurations[0] == nil || p.IPConfigurations[0].Properties == nil {
		return hostedexecution.ErrOwnership
	}
	ip := p.IPConfigurations[0]
	if ip.ID != nil && !strings.EqualFold(value(ip.ID), s.configuration()) || ip.Type != nil && !strings.EqualFold(value(ip.Type), "Microsoft.Network/azureFirewalls/azureFirewallIpConfigurations") {
		return hostedexecution.ErrOwnership
	}
	if value(ip.Properties.ProvisioningState) != n.ProvisioningStateSucceeded || value(ip.Properties.PrivateIPAddress) == "" {
		return hostedexecution.ErrUnavailable
	}
	if value(ip.Properties.PrivateIPAddress) != s.Perimeter.FirewallIP {
		return hostedexecution.ErrOwnership
	}
	doc := document(p)
	delete(doc, "provisioningState")
	delete(doc, "ipGroups")
	delete(doc, "autoscaleConfiguration")
	configurations, _ := doc["ipConfigurations"].([]any)
	configuration, _ := configurations[0].(map[string]any)
	delete(configuration, "id")
	delete(configuration, "etag")
	delete(configuration, "type")
	fields, _ := configuration["properties"].(map[string]any)
	delete(fields, "privateIPAddress")
	delete(fields, "provisioningState")
	expected, _ := s.FirewallDefinition(c)
	if fingerprint(doc) != fingerprint(document(expected.Properties)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *FirewallGuard) verifySubnet(actual n.Subnet) error {
	s := g.settings
	if err := subnetMatches(actual, s.subnet()); err != nil {
		return err
	}
	prefixes := subnetPrefixes(actual)
	p := actual.Properties
	if len(prefixes) != 1 || prefixes[0] != s.SubnetPrefix || p.DefaultOutboundAccess == nil || value(p.DefaultOutboundAccess) || p.RouteTable != nil || p.NetworkSecurityGroup != nil || p.NatGateway != nil || value(p.SharingScope) != "" || len(p.Delegations)+len(p.ServiceEndpoints)+len(p.ServiceEndpointPolicies)+len(p.PrivateEndpoints)+len(p.ApplicationGatewayIPConfigurations)+len(p.IPAllocations)+len(p.IpamPoolPrefixAllocations) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *FirewallGuard) verifyPublicIP(actual n.PublicIPAddress) error {
	s, c := g.settings, g.worker
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), s.PublicIP, "Microsoft.Network/publicIPAddresses", value(actual.Location), c.Region) || actual.Properties == nil || actual.SKU == nil || actual.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded || value(p.IPAddress) == "" {
		return hostedexecution.ErrUnavailable
	}
	ip, err := netip.ParseAddr(value(p.IPAddress))
	if err != nil || !ip.Is4() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || value(actual.SKU.Name) != n.PublicIPAddressSKUNameStandard || value(actual.SKU.Tier) != n.PublicIPAddressSKUTierRegional || value(p.PublicIPAddressVersion) != n.IPVersionIPv4 || value(p.PublicIPAllocationMethod) != n.IPAllocationMethodStatic || p.IPConfiguration == nil || !strings.EqualFold(value(p.IPConfiguration.ID), s.configuration()) || p.NatGateway != nil || p.LinkedPublicIPAddress != nil || p.ServicePublicIPAddress != nil || p.PublicIPPrefix != nil || len(p.IPTags) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
