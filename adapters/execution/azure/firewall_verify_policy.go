package azureworker

import (
	"slices"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func canonicalFirewallContent(v any) any {
	switch value := v.(type) {
	case map[string]any:
		delete(value, "description")
		if value["terminateTLS"] == false {
			delete(value, "terminateTLS")
		}
		for key, child := range value {
			value[key] = canonicalFirewallContent(child)
		}
	case []any:
		for i, child := range value {
			value[i] = canonicalFirewallContent(child)
		}
		slices.SortFunc(value, func(a, b any) int { return strings.Compare(fingerprint(a), fingerprint(b)) })
	}
	return v
}

func (g *FirewallGuard) verifyPolicy(actual n.FirewallPolicy, groups []n.FirewallPolicyRuleCollectionGroup) error {
	s, c := g.settings, g.worker
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), s.Policy, "Microsoft.Network/firewallPolicies", value(actual.Location), c.Region) || actual.Properties == nil || actual.Identity != nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if len(groups) == 0 || len(p.RuleCollectionGroups) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(p.Firewalls) != 1 || p.Firewalls[0] == nil || !strings.EqualFold(value(p.Firewalls[0].ID), s.Firewall) || len(p.ChildPolicies) != 0 || len(p.RuleCollectionGroups) != len(groups) || len(groups) < 1 || len(groups) > 4 {
		return hostedexecution.ErrOwnership
	}
	refs := map[string]bool{}
	for _, ref := range p.RuleCollectionGroups {
		if ref == nil || refs[strings.ToLower(value(ref.ID))] {
			return hostedexecution.ErrOwnership
		}
		refs[strings.ToLower(value(ref.ID))] = true
	}
	if p.DNSSettings != nil && (value(p.DNSSettings.EnableProxy) || value(p.DNSSettings.RequireProxyForNetworkRules) || len(p.DNSSettings.Servers) != 0) {
		return hostedexecution.ErrOwnership
	}
	expected, _ := s.PolicyDefinition(c)
	fields := document(p)
	for _, key := range []string{"provisioningState", "size", "firewalls", "ruleCollectionGroups", "childPolicies"} {
		delete(fields, key)
	}
	fields["dnsSettings"] = document(expected.Properties.DNSSettings)
	if fingerprint(fields) != fingerprint(document(expected.Properties)) {
		return hostedexecution.ErrOwnership
	}
	defaults := map[string]int32{"DefaultDnatRuleCollectionGroup": 100, "DefaultNetworkRuleCollectionGroup": 200, "DefaultApplicationRuleCollectionGroup": 300}
	owned := false
	for _, group := range groups {
		id, name := value(group.ID), value(group.Name)
		key := strings.ToLower(id)
		if !refs[key] || !strings.EqualFold(id, s.Policy+"/ruleCollectionGroups/"+name) || value(group.Type) != "Microsoft.Network/firewallPolicies/ruleCollectionGroups" || group.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		delete(refs, key)
		if value(group.Properties.ProvisioningState) != n.ProvisioningStateSucceeded {
			return hostedexecution.ErrUnavailable
		}
		if name == "worker-platform" {
			if owned || !strings.EqualFold(id, s.group()) {
				return hostedexecution.ErrOwnership
			}
			owned = true
			expected, _ := s.RuleGroupDefinition(c)
			fields := document(group.Properties)
			delete(fields, "provisioningState")
			delete(fields, "size")
			if fingerprint(canonicalFirewallContent(fields)) != fingerprint(canonicalFirewallContent(document(expected.Properties))) {
				return hostedexecution.ErrOwnership
			}
		} else {
			priority, ok := defaults[name]
			if !ok || value(group.Properties.Priority) != priority || len(group.Properties.RuleCollections) != 0 {
				return hostedexecution.ErrOwnership
			}
			delete(defaults, name)
		}
	}
	if !owned {
		return hostedexecution.ErrUnavailable
	}
	if len(refs) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
