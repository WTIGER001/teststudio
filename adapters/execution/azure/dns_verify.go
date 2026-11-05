package azureworker

import (
	"slices"
	"strings"

	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func dnsIdentity(id, name, kind, expected, expectedKind, location, region string) bool {
	parts := strings.Split(expected, "/")
	return strings.EqualFold(id, expected) && name == parts[len(parts)-1] && strings.EqualFold(kind, expectedKind) && location == region
}

func (s dnsBinding) verifyDomainList(actual d.DomainList, allow bool, region string) error {
	id := s.BlockList
	expected := []string{"."}
	if allow {
		id = s.AllowList
		expected = s.Domains
	}
	if !dnsIdentity(value(actual.ID), value(actual.Name), value(actual.Type), id, "Microsoft.Network/dnsResolverDomainLists", value(actual.Location), region) || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != d.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if value(p.DomainsURL) != "" {
		return hostedexecution.ErrOwnership
	}
	if len(p.Domains) == 0 {
		return hostedexecution.ErrUnavailable
	}
	domains := make([]string, 0, len(p.Domains))
	for _, domain := range p.Domains {
		if domain == nil {
			return hostedexecution.ErrOwnership
		}
		domains = append(domains, value(domain))
	}
	slices.Sort(domains)
	if !slices.Equal(domains, expected) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s dnsBinding) verifyRules(rules []d.DNSSecurityRule, region string) error {
	if len(rules) < 2 {
		return hostedexecution.ErrUnavailable
	}
	if len(rules) != 2 {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, rule := range rules {
		name := value(rule.Name)
		allow := name == s.Role+"-allow-approved"
		if !allow && name != s.Role+"-default-block" || seen[name] || !dnsIdentity(value(rule.ID), name, value(rule.Type), s.rulePath()+"/"+name, "Microsoft.Network/dnsResolverPolicies/dnsSecurityRules", value(rule.Location), region) || rule.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		seen[name] = true
		if value(rule.Properties.ProvisioningState) != d.ProvisioningStateSucceeded {
			return hostedexecution.ErrUnavailable
		}
		fields := document(rule.Properties)
		delete(fields, "provisioningState")
		expected := s.ruleDefinition(region, allow)
		if fingerprint(fields) != fingerprint(document(expected.Properties)) {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func (s dnsBinding) verifyLinks(links []d.PolicyVirtualNetworkLink, attached []d.SubResource, region string) error {
	if len(links) == 0 || len(attached) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(links) != 1 || len(attached) != 1 || !strings.EqualFold(value(attached[0].ID), s.Policy) {
		return hostedexecution.ErrOwnership
	}
	link := links[0]
	if !dnsIdentity(value(link.ID), value(link.Name), value(link.Type), s.linkPath()+"/"+s.Role+"-network", "Microsoft.Network/dnsResolverPolicies/virtualNetworkLinks", value(link.Location), region) || link.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	if value(link.Properties.ProvisioningState) != d.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	fields := document(link.Properties)
	delete(fields, "provisioningState")
	expected := s.linkDefinition(region)
	if fingerprint(fields) != fingerprint(document(expected.Properties)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
