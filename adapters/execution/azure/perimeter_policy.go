package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func perimeterRule(name string, direction n.SecurityRuleDirection, priority int32, protocol n.SecurityRuleProtocol, access n.SecurityRuleAccess, source, destination, port string) *n.SecurityRule {
	return &n.SecurityRule{Name: to.Ptr(name), Properties: &n.SecurityRulePropertiesFormat{Direction: to.Ptr(direction), Priority: to.Ptr(priority), Protocol: to.Ptr(protocol), Access: to.Ptr(access), SourceAddressPrefix: to.Ptr(source), SourcePortRange: to.Ptr("*"), DestinationAddressPrefix: to.Ptr(destination), DestinationPortRange: to.Ptr(port)}}
}

func (s PerimeterSettings) rules() []*n.SecurityRule {
	worker := s.Topology.WorkerSubnetPrefix
	rules := []*n.SecurityRule{
		perimeterRule("worker-platform-internal-in", n.SecurityRuleDirectionInbound, 100, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessAllow, worker, worker, "*"),
		perimeterRule("worker-load-balancer", n.SecurityRuleDirectionInbound, 110, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, "AzureLoadBalancer", worker, "30000-32767"),
		perimeterRule("worker-deny-in", n.SecurityRuleDirectionInbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
		perimeterRule("worker-deny-metadata", n.SecurityRuleDirectionOutbound, 100, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "AzurePlatformIMDS", "*"),
		perimeterRule("worker-platform-internal-out", n.SecurityRuleDirectionOutbound, 110, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessAllow, worker, worker, "*"),
		perimeterRule("worker-platform-dns-udp", n.SecurityRuleDirectionOutbound, 120, n.SecurityRuleProtocolUDP, n.SecurityRuleAccessAllow, worker, "AzurePlatformDNS", "53"),
		perimeterRule("worker-platform-dns-tcp", n.SecurityRuleDirectionOutbound, 121, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "AzurePlatformDNS", "53"),
		perimeterRule("worker-deny-other-platform-dns", n.SecurityRuleDirectionOutbound, 130, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "AzurePlatformDNS", "*"),
		perimeterRule("worker-platform-registry", n.SecurityRuleDirectionOutbound, 200, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "MicrosoftContainerRegistry", "443"),
		perimeterRule("worker-platform-registry-cdn", n.SecurityRuleDirectionOutbound, 210, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "AzureFrontDoor.FirstParty", "443"),
		perimeterRule("worker-platform-identity", n.SecurityRuleDirectionOutbound, 220, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "AzureActiveDirectory", "443"),
		perimeterRule("worker-private-endpoints", n.SecurityRuleDirectionOutbound, 300, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, worker, "", "443"),
		perimeterRule("worker-deny-out", n.SecurityRuleDirectionOutbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
	}
	endpoints := rules[len(rules)-2].Properties
	endpoints.DestinationAddressPrefix = nil
	for _, ip := range s.PrivateEndpointIPs {
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
	return n.RouteTable{Location: to.Ptr(c.Region), Properties: &n.RouteTablePropertiesFormat{DisableBgpRoutePropagation: to.Ptr(true), Routes: []*n.Route{{Name: to.Ptr("worker-egress"), Properties: &n.RoutePropertiesFormat{AddressPrefix: to.Ptr("0.0.0.0/0"), NextHopType: to.Ptr(n.RouteNextHopTypeVirtualAppliance), NextHopIPAddress: to.Ptr(canonical.FirewallIP)}}}}}, nil
}
