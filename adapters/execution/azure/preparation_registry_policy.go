package azureworker

import (
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

func (s PreparationRegistrySettings) rules() []*n.SecurityRule {
	addresses := []string{s.RegistryIP, s.DataIP}
	slices.Sort(addresses)
	allow := perimeterRule("trusted-registry-https", n.SecurityRuleDirectionInbound, 100, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, "", "", "443")
	allow.Properties.SourceAddressPrefix = nil
	for _, prefix := range s.ControllerPrefixes {
		allow.Properties.SourceAddressPrefixes = append(allow.Properties.SourceAddressPrefixes, to.Ptr(prefix))
	}
	allow.Properties.DestinationAddressPrefix = nil
	for _, ip := range addresses {
		allow.Properties.DestinationAddressPrefixes = append(allow.Properties.DestinationAddressPrefixes, to.Ptr(ip))
	}
	return []*n.SecurityRule{allow, perimeterRule("endpoint-deny-in", n.SecurityRuleDirectionInbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"), perimeterRule("endpoint-deny-out", n.SecurityRuleDirectionOutbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*")}
}
func (s PreparationRegistrySettings) SecurityGroupDefinition(c Config) (n.SecurityGroup, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(c.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: s.rules()}}, nil
}
func (s PreparationRegistrySettings) SubnetDefinition(c Config) (n.Subnet, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.Subnet{}, err
	}
	return n.Subnet{Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.SubnetPrefix), DefaultOutboundAccess: to.Ptr(false), NetworkSecurityGroup: &n.SecurityGroup{ID: to.Ptr(s.SecurityGroup)}, PrivateEndpointNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesEnabled), PrivateLinkServiceNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled)}}, nil
}
func (s PreparationRegistrySettings) ZoneDefinition(c Config) (p.PrivateZone, error) {
	if _, err := s.canonical(c); err != nil {
		return p.PrivateZone{}, err
	}
	return p.PrivateZone{Location: to.Ptr("global")}, nil
}
func (s PreparationRegistrySettings) LinkDefinition(c Config) (p.VirtualNetworkLink, error) {
	if _, err := s.canonical(c); err != nil {
		return p.VirtualNetworkLink{}, err
	}
	return s.binding(c).linkDefinition(), nil
}
func (s PreparationRegistrySettings) RecordDefinitions(c Config) (map[string]p.RecordSet, error) {
	if _, err := s.canonical(c); err != nil {
		return nil, err
	}
	return s.binding(c).recordDefinitions(), nil
}
