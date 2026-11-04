package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
)

func (s DNSSettings) PolicyDefinition(c Config) (d.Policy, error) {
	if _, err := s.canonical(c); err != nil {
		return d.Policy{}, err
	}
	return d.Policy{Location: to.Ptr(c.Region)}, nil
}

func (s DNSSettings) DomainListDefinition(c Config, allow bool) (d.DomainList, error) {
	if _, err := s.canonical(c); err != nil {
		return d.DomainList{}, err
	}
	domains := []string{"."}
	if allow {
		domains = s.ApprovedDomains(c)
	}
	list := d.DomainList{Location: to.Ptr(c.Region), Properties: &d.DomainListProperties{}}
	for _, domain := range domains {
		list.Properties.Domains = append(list.Properties.Domains, to.Ptr(domain))
	}
	return list, nil
}

func (s DNSSettings) RuleDefinition(c Config, allow bool) (d.DNSSecurityRule, error) {
	if _, err := s.canonical(c); err != nil {
		return d.DNSSecurityRule{}, err
	}
	action, list, priority := d.ActionTypeBlock, s.BlockList, int32(65000)
	if allow {
		action, list, priority = d.ActionTypeAllow, s.AllowList, 100
	}
	return d.DNSSecurityRule{Location: to.Ptr(c.Region), Properties: &d.DNSSecurityRuleProperties{Action: &d.DNSSecurityRuleAction{ActionType: to.Ptr(action)}, DNSResolverDomainLists: []*d.SubResource{{ID: to.Ptr(list)}}, Priority: to.Ptr(priority), DNSSecurityRuleState: to.Ptr(d.DNSSecurityRuleStateEnabled)}}, nil
}

func (s DNSSettings) LinkDefinition(c Config) (d.PolicyVirtualNetworkLink, error) {
	if _, err := s.canonical(c); err != nil {
		return d.PolicyVirtualNetworkLink{}, err
	}
	return d.PolicyVirtualNetworkLink{Location: to.Ptr(c.Region), Properties: &d.PolicyVirtualNetworkLinkProperties{VirtualNetwork: &d.SubResource{ID: to.Ptr(s.Firewall.Perimeter.Topology.WorkerVNet)}}}, nil
}
