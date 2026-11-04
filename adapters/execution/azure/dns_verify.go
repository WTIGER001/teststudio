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

func (g *DNSGuard) verifyDomainList(actual d.DomainList, allow bool) error {
	id := g.settings.BlockList
	expected := []string{"."}
	if allow {
		id = g.settings.AllowList
		expected = g.settings.ApprovedDomains(g.worker)
	}
	if !dnsIdentity(value(actual.ID), value(actual.Name), value(actual.Type), id, "Microsoft.Network/dnsResolverDomainLists", value(actual.Location), g.worker.Region) || actual.Properties == nil {
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

func (g *DNSGuard) verifyRules(rules []d.DNSSecurityRule) error {
	if len(rules) < 2 {
		return hostedexecution.ErrUnavailable
	}
	if len(rules) != 2 {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, rule := range rules {
		name := value(rule.Name)
		allow := name == "worker-allow-approved"
		if !allow && name != "worker-default-block" || seen[name] || !dnsIdentity(value(rule.ID), name, value(rule.Type), g.settings.rulePath()+"/"+name, "Microsoft.Network/dnsResolverPolicies/dnsSecurityRules", value(rule.Location), g.worker.Region) || rule.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		seen[name] = true
		if value(rule.Properties.ProvisioningState) != d.ProvisioningStateSucceeded {
			return hostedexecution.ErrUnavailable
		}
		fields := document(rule.Properties)
		delete(fields, "provisioningState")
		expected, _ := g.settings.RuleDefinition(g.worker, allow)
		if fingerprint(fields) != fingerprint(document(expected.Properties)) {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func (g *DNSGuard) verifyLinks(links []d.PolicyVirtualNetworkLink, attached []d.SubResource) error {
	if len(links) == 0 || len(attached) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(links) != 1 || len(attached) != 1 || !strings.EqualFold(value(attached[0].ID), g.settings.Policy) {
		return hostedexecution.ErrOwnership
	}
	link := links[0]
	if !dnsIdentity(value(link.ID), value(link.Name), value(link.Type), g.settings.linkPath()+"/worker-network", "Microsoft.Network/dnsResolverPolicies/virtualNetworkLinks", value(link.Location), g.worker.Region) || link.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	if value(link.Properties.ProvisioningState) != d.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	fields := document(link.Properties)
	delete(fields, "provisioningState")
	expected, _ := g.settings.LinkDefinition(g.worker)
	if fingerprint(fields) != fingerprint(document(expected.Properties)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
