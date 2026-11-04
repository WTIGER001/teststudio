package ecsworker

import (
	"context"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53profiles"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *NetworkGuard) checkDNS(ctx context.Context) error {
	s, c := g.settings, g.worker
	profiles, err := g.profiles.ListProfileAssociations(ctx, &route53profiles.ListProfileAssociationsInput{ResourceId: aws.String(s.VPC), MaxResults: aws.Int32(100)}, oneProfiles)
	if err != nil || profiles == nil {
		return hostedexecution.ErrUnavailable
	}
	if profiles.NextToken != nil || len(profiles.ProfileAssociations) != 0 {
		return hostedexecution.ErrOwnership
	}
	config, err := g.resolver.GetFirewallConfig(ctx, &route53resolver.GetFirewallConfigInput{ResourceId: aws.String(s.VPC)}, oneResolver)
	if err != nil || config == nil {
		return hostedexecution.ErrUnavailable
	}
	if config.FirewallConfig == nil || aws.ToString(config.FirewallConfig.ResourceId) != s.VPC || aws.ToString(config.FirewallConfig.OwnerId) != c.Account || config.FirewallConfig.FirewallFailOpen != types.FirewallFailOpenStatusDisabled {
		return hostedexecution.ErrOwnership
	}
	associations, err := g.resolver.ListFirewallRuleGroupAssociations(ctx, &route53resolver.ListFirewallRuleGroupAssociationsInput{VpcId: aws.String(s.VPC), MaxResults: aws.Int32(100)}, oneResolver)
	if err != nil || associations == nil {
		return hostedexecution.ErrUnavailable
	}
	if associations.NextToken != nil || len(associations.FirewallRuleGroupAssociations) != 1 {
		return hostedexecution.ErrOwnership
	}
	a := associations.FirewallRuleGroupAssociations[0]
	if aws.ToString(a.Id) != s.FirewallAssociation || aws.ToString(a.VpcId) != s.VPC || aws.ToString(a.FirewallRuleGroupId) != s.FirewallGroup || aws.ToString(a.Arn) != g.resolverARN("firewall-rule-group-association", s.FirewallAssociation) || aws.ToString(a.ManagedOwnerName) != "" || a.Status != types.FirewallRuleGroupAssociationStatusComplete || a.MutationProtection != types.MutationProtectionStatusEnabled {
		return hostedexecution.ErrOwnership
	}
	group, err := g.resolver.GetFirewallRuleGroup(ctx, &route53resolver.GetFirewallRuleGroupInput{FirewallRuleGroupId: aws.String(s.FirewallGroup)}, oneResolver)
	if err != nil || group == nil {
		return hostedexecution.ErrUnavailable
	}
	if group.FirewallRuleGroup == nil || aws.ToString(group.FirewallRuleGroup.Id) != s.FirewallGroup || aws.ToString(group.FirewallRuleGroup.OwnerId) != c.Account || aws.ToString(group.FirewallRuleGroup.Arn) != g.resolverARN("firewall-rule-group", s.FirewallGroup) || group.FirewallRuleGroup.Status != types.FirewallRuleGroupStatusComplete {
		return hostedexecution.ErrOwnership
	}
	rules, err := g.resolver.ListFirewallRules(ctx, &route53resolver.ListFirewallRulesInput{FirewallRuleGroupId: aws.String(s.FirewallGroup), MaxResults: aws.Int32(100)}, oneResolver)
	if err != nil || rules == nil {
		return hostedexecution.ErrUnavailable
	}
	if rules.NextToken != nil || len(rules.FirewallRules) != 2 {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, rule := range rules.FirewallRules {
		id := aws.ToString(rule.FirewallDomainListId)
		if seen[id] || aws.ToString(rule.FirewallRuleGroupId) != s.FirewallGroup || aws.ToString(rule.Qtype) != "" || rule.FirewallRuleType != nil || rule.DnsThreatProtection != "" || rule.FirewallDomainRedirectionAction != types.FirewallDomainRedirectionActionTrustRedirectionDomain {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
		switch id {
		case s.AllowList:
			if rule.Action != types.ActionAllow || aws.ToInt32(rule.Priority) != 100 || rule.BlockResponse != "" || aws.ToString(rule.BlockOverrideDomain) != "" {
				return hostedexecution.ErrOwnership
			}
		case s.BlockList:
			if rule.Action != types.ActionBlock || aws.ToInt32(rule.Priority) != 200 || rule.BlockResponse != types.BlockResponseNxdomain || aws.ToString(rule.BlockOverrideDomain) != "" {
				return hostedexecution.ErrOwnership
			}
		default:
			return hostedexecution.ErrOwnership
		}
	}
	if err := g.checkDomains(ctx, s.AllowList, s.Domains(c)); err != nil {
		return err
	}
	return g.checkDomains(ctx, s.BlockList, []string{"*"})
}

func (g *NetworkGuard) checkDomains(ctx context.Context, id string, expected []string) error {
	list, err := g.resolver.GetFirewallDomainList(ctx, &route53resolver.GetFirewallDomainListInput{FirewallDomainListId: aws.String(id)}, oneResolver)
	if err != nil || list == nil {
		return hostedexecution.ErrUnavailable
	}
	if list.FirewallDomainList == nil || aws.ToString(list.FirewallDomainList.Id) != id || aws.ToString(list.FirewallDomainList.Arn) != g.resolverARN("firewall-domain-list", id) || list.FirewallDomainList.Status != types.FirewallDomainListStatusComplete || aws.ToString(list.FirewallDomainList.ManagedOwnerName) != "" || aws.ToInt32(list.FirewallDomainList.DomainCount) != int32(len(expected)) {
		return hostedexecution.ErrOwnership
	}
	values, err := g.resolver.ListFirewallDomains(ctx, &route53resolver.ListFirewallDomainsInput{FirewallDomainListId: aws.String(id), MaxResults: aws.Int32(100)}, oneResolver)
	if err != nil || values == nil {
		return hostedexecution.ErrUnavailable
	}
	if values.NextToken != nil || len(values.Domains) != len(expected) {
		return hostedexecution.ErrOwnership
	}
	actual := slices.Clone(values.Domains)
	expected = slices.Clone(expected)
	for i := range actual {
		actual[i] = strings.TrimSuffix(strings.ToLower(actual[i]), ".")
	}
	for i := range expected {
		expected[i] = strings.TrimSuffix(strings.ToLower(expected[i]), ".")
	}
	if !equalSet(actual, expected) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *NetworkGuard) resolverARN(kind, id string) string {
	return "arn:aws:route53resolver:" + g.worker.Region + ":" + g.worker.Account + ":" + kind + "/" + id
}
