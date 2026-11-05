package azureworker

import (
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func (s RelayPerimeterSettings) rules() []*n.SecurityRule {
	addresses := append([]string{s.Topology.BackendEndpoint.Address}, s.RegistryEndpointIPs...)
	slices.Sort(addresses)
	return platformPerimeterRules("relay", s.Topology.BackendEndpoint.RelaySubnetPrefix, addresses)
}

func (s RelayPerimeterSettings) SecurityGroupDefinition(c Config) (n.SecurityGroup, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(c.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: s.rules()}}, nil
}

func (s RelayPerimeterSettings) RouteTableDefinition(c Config) (n.RouteTable, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.RouteTable{}, err
	}
	return perimeterRouteDefinition(c.Region, "relay-egress", s.FirewallIP), nil
}
