package azureworker

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s EndpointSettings) rules() []*n.SecurityRule {
	allow := perimeterRule("worker-https", n.SecurityRuleDirectionInbound, 100, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, s.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerSubnetPrefix, "", "443")
	allow.Properties.DestinationAddressPrefix = nil
	for _, ip := range s.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs {
		allow.Properties.DestinationAddressPrefixes = append(allow.Properties.DestinationAddressPrefixes, to.Ptr(ip))
	}
	return []*n.SecurityRule{allow,
		perimeterRule("endpoint-deny-in", n.SecurityRuleDirectionInbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
		perimeterRule("endpoint-deny-out", n.SecurityRuleDirectionOutbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
	}
}

func (s EndpointSettings) SecurityGroupDefinition(c Config) (n.SecurityGroup, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(c.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: s.rules()}}, nil
}

func (s EndpointSettings) SubnetDefinition(c Config) (n.Subnet, error) {
	s, err := s.canonical(c)
	if err != nil {
		return n.Subnet{}, err
	}
	return n.Subnet{Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.SubnetPrefix), DefaultOutboundAccess: to.Ptr(false), NetworkSecurityGroup: &n.SecurityGroup{ID: to.Ptr(s.SecurityGroup)}, PrivateEndpointNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesEnabled), PrivateLinkServiceNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled)}}, nil
}

func (g *EndpointGuard) verifySecurityGroup(actual n.SecurityGroup) error {
	return verifyEndpointSecurityGroup(actual, g.settings.SecurityGroup, g.settings.Subnet, g.worker.Region, g.settings.rules())
}

func verifyEndpointSecurityGroup(actual n.SecurityGroup, id, subnet, location string, rules []*n.SecurityRule) error {
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), id, "Microsoft.Network/networkSecurityGroups", value(actual.Location), location) || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if !value(p.FlushConnection) || len(p.NetworkInterfaces) != 0 || !soleSubnet(p.Subnets, subnet) {
		return hostedexecution.ErrOwnership
	}
	return verifySecurityRules(id, p.SecurityRules, rules)
}

func (g *EndpointGuard) verifySubnet(actual n.Subnet, interfaces []n.Interface) error {
	s := g.settings
	return verifyEndpointSubnet(actual, interfaces, s.Subnet, s.SubnetPrefix, s.SecurityGroup, []string{s.RegistryEndpoint, s.ExchangeEndpoint})
}

func verifyEndpointSubnet(actual n.Subnet, interfaces []n.Interface, subnet, prefix, security string, endpointIDs []string) error {
	if err := subnetMatches(actual, subnet); err != nil {
		return err
	}
	p := actual.Properties
	prefixes := subnetPrefixes(actual)
	if len(prefixes) != 1 ||
		prefixes[0] != prefix ||
		p.DefaultOutboundAccess == nil ||
		value(p.DefaultOutboundAccess) ||
		p.NetworkSecurityGroup == nil ||
		!strings.EqualFold(value(p.NetworkSecurityGroup.ID), security) ||
		value(p.PrivateEndpointNetworkPolicies) != n.VirtualNetworkPrivateEndpointNetworkPoliciesEnabled ||
		value(p.PrivateLinkServiceNetworkPolicies) != n.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled ||
		p.NatGateway != nil ||
		p.RouteTable != nil ||
		value(p.SharingScope) != "" ||
		len(p.ApplicationGatewayIPConfigurations)+len(p.Delegations)+len(p.IPAllocations)+len(p.IpamPoolPrefixAllocations)+len(p.ServiceEndpoints)+len(p.ServiceEndpointPolicies)+len(p.IPConfigurationProfiles)+len(p.ResourceNavigationLinks)+len(p.ServiceAssociationLinks) != 0 {
		return hostedexecution.ErrOwnership
	}
	endpoints := map[string]bool{}
	for _, id := range endpointIDs {
		endpoints[strings.ToLower(id)] = true
	}
	if len(p.PrivateEndpoints) < len(endpointIDs) {
		return hostedexecution.ErrUnavailable
	}
	if len(p.PrivateEndpoints) != len(endpointIDs) {
		return hostedexecution.ErrOwnership
	}
	for _, endpoint := range p.PrivateEndpoints {
		if endpoint == nil || !endpoints[strings.ToLower(value(endpoint.ID))] {
			return hostedexecution.ErrOwnership
		}
		delete(endpoints, strings.ToLower(value(endpoint.ID)))
	}
	ips := map[string]bool{}
	for _, nic := range interfaces {
		for _, ip := range nic.Properties.IPConfigurations {
			ips[strings.ToLower(value(ip.ID))] = true
		}
	}
	if len(p.IPConfigurations) < len(ips) {
		return hostedexecution.ErrUnavailable
	}
	if len(p.IPConfigurations) != len(ips) {
		return hostedexecution.ErrOwnership
	}
	for _, ip := range p.IPConfigurations {
		if ip == nil || !ips[strings.ToLower(value(ip.ID))] {
			return hostedexecution.ErrOwnership
		}
		delete(ips, strings.ToLower(value(ip.ID)))
	}
	return nil
}
