package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func perimeterRule(name string, direction n.SecurityRuleDirection, priority int32, protocol n.SecurityRuleProtocol, access n.SecurityRuleAccess, source, destination, port string) *n.SecurityRule {
	return &n.SecurityRule{Name: to.Ptr(name), Properties: &n.SecurityRulePropertiesFormat{Direction: to.Ptr(direction), Priority: to.Ptr(priority), Protocol: to.Ptr(protocol), Access: to.Ptr(access), SourceAddressPrefix: to.Ptr(source), SourcePortRange: to.Ptr("*"), DestinationAddressPrefix: to.Ptr(destination), DestinationPortRange: to.Ptr(port)}}
}

func (s PerimeterSettings) rules() []*n.SecurityRule {
	return platformPerimeterRules("worker", s.Topology.WorkerSubnetPrefix, s.PrivateEndpointIPs)
}

func platformPerimeterRules(role, worker string, addresses []string) []*n.SecurityRule {
	rules := []*n.SecurityRule{
		perimeterRule(role+"-platform-internal-in", n.SecurityRuleDirectionInbound, 100, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessAllow, worker, worker, "*"),
		perimeterRule(role+"-load-balancer", n.SecurityRuleDirectionInbound, 110, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, "AzureLoadBalancer", worker, "30000-32767"),
		perimeterRule(role+"-deny-in", n.SecurityRuleDirectionInbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
		perimeterRule(role+"-deny-metadata", n.SecurityRuleDirectionOutbound, 100, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "AzurePlatformIMDS", "*"),
		perimeterRule(role+"-platform-internal-out", n.SecurityRuleDirectionOutbound, 110, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessAllow, worker, worker, "*"),
		perimeterRule(role+"-platform-dns-udp", n.SecurityRuleDirectionOutbound, 120, n.SecurityRuleProtocolUDP, n.SecurityRuleAccessAllow, worker, "AzurePlatformDNS", "53"),
		perimeterRule(role+"-platform-dns-tcp", n.SecurityRuleDirectionOutbound, 121, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "AzurePlatformDNS", "53"),
		perimeterRule(role+"-deny-other-platform-dns", n.SecurityRuleDirectionOutbound, 130, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "AzurePlatformDNS", "*"),
		perimeterRule(role+"-platform-registry", n.SecurityRuleDirectionOutbound, 200, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "MicrosoftContainerRegistry", "443"),
		perimeterRule(role+"-platform-registry-cdn", n.SecurityRuleDirectionOutbound, 210, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "AzureFrontDoor.FirstParty", "443"),
		perimeterRule(role+"-platform-identity", n.SecurityRuleDirectionOutbound, 220, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "AzureActiveDirectory", "443"),
		perimeterRule(role+"-private-endpoints", n.SecurityRuleDirectionOutbound, 300, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "", "443"),
		perimeterRule(role+"-deny-out", n.SecurityRuleDirectionOutbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
	}
	endpoints := rules[len(rules)-2].Properties
	endpoints.DestinationAddressPrefix = nil
	for _, ip := range addresses {
		endpoints.DestinationAddressPrefixes = append(endpoints.DestinationAddressPrefixes, to.Ptr(ip))
	}
	return rules
}

func (s PerimeterSettings) SecurityGroupDefinition(c Config) (n.SecurityGroup, error) {
	canonical, err := s.canonical(c)
	if err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(c.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: canonical.rules()}}, nil
}

func (s PerimeterSettings) RouteTableDefinition(c Config) (n.RouteTable, error) {
	canonical, err := s.canonical(c)
	if err != nil {
		return n.RouteTable{}, err
	}
	return perimeterRouteDefinition(c.Region, "worker-egress", canonical.FirewallIP), nil
}

func perimeterRouteDefinition(region, name, firewall string) n.RouteTable {
	return n.RouteTable{Location: to.Ptr(region), Properties: &n.RouteTablePropertiesFormat{DisableBgpRoutePropagation: to.Ptr(true), Routes: []*n.Route{{Name: to.Ptr(name), Properties: &n.RoutePropertiesFormat{AddressPrefix: to.Ptr("0.0.0.0/0"), NextHopType: to.Ptr(n.RouteNextHopTypeVirtualAppliance), NextHopIPAddress: to.Ptr(firewall)}}}}}
}
