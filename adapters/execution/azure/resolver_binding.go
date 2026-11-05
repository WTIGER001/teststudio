package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
)

type ResolverBinding struct {
	Policy, AllowList, BlockList, VirtualNetwork, Role string
	Domains                                            []string
}

type ResolverObservation struct {
	Policy       d.Policy
	Allow, Block d.DomainList
	Rules        []d.DNSSecurityRule
	Links        []d.PolicyVirtualNetworkLink
	Attached     []d.SubResource
}

func (b ResolverBinding) binding() dnsBinding {
	return dnsBinding{Policy: b.Policy, AllowList: b.AllowList, BlockList: b.BlockList, VirtualNetwork: b.VirtualNetwork, Role: b.Role, Domains: b.Domains}
}

func (b ResolverBinding) Verify(region string, actual ResolverObservation) error {
	return b.binding().verify(region, actual.Policy, actual.Allow, actual.Block, actual.Rules, actual.Links, actual.Attached)
}

func (b ResolverBinding) Definitions(region string) map[string]any {
	s := b.binding()
	return map[string]any{
		b.Policy:    d.Policy{Location: to.Ptr(region)},
		b.AllowList: s.domainListDefinition(region, true), b.BlockList: s.domainListDefinition(region, false),
		s.rulePath() + "/" + b.Role + "-allow-approved": s.ruleDefinition(region, true),
		s.rulePath() + "/" + b.Role + "-default-block":  s.ruleDefinition(region, false),
		s.linkPath() + "/" + b.Role + "-network":        s.linkDefinition(region),
	}
}
