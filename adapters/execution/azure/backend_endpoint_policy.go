package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func (s BackendEndpointSettings) rules() []*n.SecurityRule {
	return []*n.SecurityRule{
		perimeterRule("relay-backend-https", n.SecurityRuleDirectionInbound, 100, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, s.RelaySubnetPrefix, s.Address, "443"),
		perimeterRule("endpoint-deny-in", n.SecurityRuleDirectionInbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
		perimeterRule("endpoint-deny-out", n.SecurityRuleDirectionOutbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
	}
}

func (s BackendEndpointSettings) SecurityGroupDefinition(c Config) (n.SecurityGroup, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(c.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: s.rules()}}, nil
}

func (s BackendEndpointSettings) SubnetDefinition(c Config) (n.Subnet, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.Subnet{}, err
	}
	return n.Subnet{Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.SubnetPrefix), DefaultOutboundAccess: to.Ptr(false), NetworkSecurityGroup: &n.SecurityGroup{ID: to.Ptr(s.SecurityGroup)}, PrivateEndpointNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesEnabled), PrivateLinkServiceNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled)}}, nil
}
