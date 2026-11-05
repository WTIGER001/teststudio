package azureworker

import (
	"strings"

	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func privateDNSInventoryID(c Config, id string) bool {
	parts := strings.Split(id, "/")
	if len(parts) != 9 || len(parts[8]) < 1 || len(parts[8]) > 253 || strings.ContainsAny(parts[8], "%?#\\") {
		return false
	}
	for _, r := range parts[8] {
		if r <= 32 || r == 127 {
			return false
		}
	}
	return parts[0] == "" && strings.EqualFold(parts[1], "subscriptions") && strings.EqualFold(parts[2], c.Subscription) && strings.EqualFold(parts[3], "resourceGroups") && inventoryGroup(parts[4]) && strings.EqualFold(parts[5], "providers") && strings.EqualFold(parts[6], "Microsoft.Network") && strings.EqualFold(parts[7], "privateDnsZones")
}

func privateDNSInventoryLinkID(zone, id string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 11 && endpointChild(id, parts[10], zone, "virtualNetworkLinks")
}

func privateDNSVNetReference(id string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 9 && parts[0] == "" && strings.EqualFold(parts[1], "subscriptions") && guid.MatchString(strings.ToLower(parts[2])) && strings.EqualFold(parts[3], "resourceGroups") && inventoryGroup(parts[4]) && strings.EqualFold(parts[5], "providers") && strings.EqualFold(parts[6], "Microsoft.Network") && strings.EqualFold(parts[7], "virtualNetworks") && endpointChildName.MatchString(parts[8])
}

func privateDNSInventoryZone(c Config, zone p.PrivateZone) error {
	id := value(zone.ID)
	if !privateDNSInventoryID(c, id) || !dnsIdentity(id, value(zone.Name), value(zone.Type), id, "Microsoft.Network/privateDnsZones", value(zone.Location), "global") {
		return hostedexecution.ErrOwnership
	}
	props := zone.Properties
	if props == nil || value(props.ProvisioningState) != p.ProvisioningStateSucceeded || props.NumberOfVirtualNetworkLinks == nil || props.NumberOfVirtualNetworkLinksWithRegistration == nil {
		return hostedexecution.ErrUnavailable
	}
	count, registered := value(props.NumberOfVirtualNetworkLinks), value(props.NumberOfVirtualNetworkLinksWithRegistration)
	if count < 0 || count > 1024 || registered < 0 || registered > count {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func privateDNSInventoryLink(zone string, link p.VirtualNetworkLink) (PrivateDNSZoneLink, error) {
	empty := PrivateDNSZoneLink{}
	if !privateDNSInventoryLinkID(zone, value(link.ID)) || !dnsIdentity(value(link.ID), value(link.Name), value(link.Type), zone+"/virtualNetworkLinks/"+value(link.Name), "Microsoft.Network/privateDnsZones/virtualNetworkLinks", value(link.Location), "global") {
		return empty, hostedexecution.ErrOwnership
	}
	props := link.Properties
	if props == nil || props.VirtualNetwork == nil || props.RegistrationEnabled == nil || value(props.ProvisioningState) != p.ProvisioningStateSucceeded || value(props.VirtualNetworkLinkState) != p.VirtualNetworkLinkStateCompleted {
		return empty, hostedexecution.ErrUnavailable
	}
	if !privateDNSVNetReference(value(props.VirtualNetwork.ID)) {
		return empty, hostedexecution.ErrOwnership
	}
	return PrivateDNSZoneLink{ZoneID: zone, ResourceID: value(link.ID), VirtualNetworkID: value(props.VirtualNetwork.ID), RegistrationEnabled: value(props.RegistrationEnabled)}, nil
}
