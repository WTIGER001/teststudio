package azureworker

import (
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func (s PreparationSettings) sourceRules() []*n.SecurityRule {
	r, p := s.Writer.Registry, *s.Network
	addresses := []string{r.RegistryIP, r.DataIP}
	slices.Sort(addresses)
	rules := platformPerimeterRules("preparation", r.SourceSubnetPrefix, addresses)
	end := rules[len(rules)-1]
	rules = rules[:len(rules)-1]
	for i, tag := range []string{"AzureResourceManager", "AzureKeyVault", "AzureMonitor"} {
		rules = append(rules, perimeterRule("preparation-service-"+tag, n.SecurityRuleDirectionOutbound, int32(310+i*10), n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, r.SourceSubnetPrefix, tag, "443"))
	}
	rules = append(rules,
		perimeterRule("preparation-database", n.SecurityRuleDirectionOutbound, 350, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, r.SourceSubnetPrefix, p.DatabaseIP, "5432"),
		perimeterRule("preparation-journal", n.SecurityRuleDirectionOutbound, 360, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, r.SourceSubnetPrefix, p.FileIP, "445"), end)
	return rules
}

func (s PreparationSettings) fileRules() []*n.SecurityRule {
	return []*n.SecurityRule{
		perimeterRule("preparation-journal-smb", n.SecurityRuleDirectionInbound, 100, n.SecurityRuleProtocolTCP, n.SecurityRuleAccessAllow, s.Writer.Registry.SourceSubnetPrefix, s.Network.FileIP, "445"),
		perimeterRule("endpoint-deny-in", n.SecurityRuleDirectionInbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
		perimeterRule("endpoint-deny-out", n.SecurityRuleDirectionOutbound, 4096, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessDeny, "*", "*", "*"),
	}
}

func (s PreparationSettings) PreparationSourceSecurityGroupDefinition() (n.SecurityGroup, error) {
	if err := s.ValidateNative(); err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(s.Worker.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: s.sourceRules()}}, nil
}

func (s PreparationSettings) PreparationSourceSubnetDefinition() (n.Subnet, error) {
	if err := s.ValidateNative(); err != nil {
		return n.Subnet{}, err
	}
	return n.Subnet{Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.Writer.Registry.SourceSubnetPrefix), DefaultOutboundAccess: to.Ptr(false), NetworkSecurityGroup: &n.SecurityGroup{ID: to.Ptr(s.Network.SourceSecurityGroup)}, NatGateway: &n.SubResource{ID: to.Ptr(s.Network.NATGateway)}, Delegations: []*n.Delegation{{Name: to.Ptr("container-apps"), Properties: &n.ServiceDelegationPropertiesFormat{ServiceName: to.Ptr("Microsoft.App/environments")}}}}}, nil
}

func (s PreparationSettings) PreparationFileSecurityGroupDefinition() (n.SecurityGroup, error) {
	if err := s.ValidateNative(); err != nil {
		return n.SecurityGroup{}, err
	}
	return n.SecurityGroup{Location: to.Ptr(s.Worker.Region), Properties: &n.SecurityGroupPropertiesFormat{FlushConnection: to.Ptr(true), SecurityRules: s.fileRules()}}, nil
}

func (s PreparationSettings) PreparationFileSubnetDefinition() (n.Subnet, error) {
	if err := s.ValidateNative(); err != nil {
		return n.Subnet{}, err
	}
	return n.Subnet{Properties: &n.SubnetPropertiesFormat{AddressPrefix: to.Ptr(s.Network.FileSubnetPrefix), DefaultOutboundAccess: to.Ptr(false), NetworkSecurityGroup: &n.SecurityGroup{ID: to.Ptr(s.Network.FileSecurityGroup)}, PrivateEndpointNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesEnabled), PrivateLinkServiceNetworkPolicies: to.Ptr(n.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled)}}, nil
}
