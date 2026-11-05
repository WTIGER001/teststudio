package azureworker

import n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"

func PlatformNetworkRules(role, source string, endpoints []string) []*n.SecurityRule {
	return platformPerimeterRules(role, source, endpoints)
}

func PlatformDomains() []string { return platformDomains() }

func PlatformFirewallRules(role, source string) n.FirewallPolicyRuleCollectionGroup {
	return (firewallBinding{Role: role, SourcePrefix: source}).ruleGroupDefinition()
}

func VerifyDefaultFirewallRoute(actual n.RouteTable, id, subnet, region, name, address string) error {
	return verifyPerimeterRoutes(actual, id, subnet, region, perimeterRouteDefinition(region, name, address))
}
