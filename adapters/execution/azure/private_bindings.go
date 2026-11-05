package azureworker

import (
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

type PrivateEndpointBinding struct {
	Endpoint, Interface, Service, Group, Subnet, Region string
	Members                                             map[string]string
}

func (b PrivateEndpointBinding) Verify(endpoint n.PrivateEndpoint, nic n.Interface) error {
	role := endpointRole{b.Endpoint, b.Interface, b.Service, b.Group, b.Members}
	if err := verifyPrivateEndpoint(role, endpoint, b.Subnet, b.Region); err != nil {
		return err
	}
	return verifyPrivateInterface(role, nic, b.Subnet, b.Region)
}

func VerifyPrivateEndpointSubnet(actual n.Subnet, interfaces []n.Interface, subnet, prefix, securityGroup string, endpoints []string) error {
	return verifyEndpointSubnet(actual, interfaces, subnet, prefix, securityGroup, endpoints)
}

type PrivateZoneBinding struct {
	ID, VirtualNetwork, LinkName string
	Records                      map[string]string
	ResolutionPolicy             bool
}

func (b PrivateZoneBinding) binding() privateZoneBinding {
	return privateZoneBinding{ID: b.ID, VirtualNetwork: b.VirtualNetwork, LinkName: b.LinkName, Records: b.Records, ResolutionPolicy: b.ResolutionPolicy}
}

func (b PrivateZoneBinding) Verify(zone p.PrivateZone, records []p.RecordSet, links []p.VirtualNetworkLink) error {
	return b.binding().verify(privateZoneObservation{zone: zone, records: records, links: links})
}

func (b PrivateZoneBinding) LinkDefinition() p.VirtualNetworkLink {
	return b.binding().linkDefinition()
}
func (b PrivateZoneBinding) RecordDefinitions() map[string]p.RecordSet {
	return b.binding().recordDefinitions()
}
