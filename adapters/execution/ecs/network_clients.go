package ecsworker

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53profiles"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
)

type EC2 interface {
	DescribeVpcs(context.Context, *ec2.DescribeVpcsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeVpcAttribute(context.Context, *ec2.DescribeVpcAttributeInput, ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error)
	DescribeDhcpOptions(context.Context, *ec2.DescribeDhcpOptionsInput, ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error)
	DescribeSubnets(context.Context, *ec2.DescribeSubnetsInput, ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeSecurityGroups(context.Context, *ec2.DescribeSecurityGroupsInput, ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeRouteTables(context.Context, *ec2.DescribeRouteTablesInput, ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DescribeVpcEndpoints(context.Context, *ec2.DescribeVpcEndpointsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DescribePrefixLists(context.Context, *ec2.DescribePrefixListsInput, ...func(*ec2.Options)) (*ec2.DescribePrefixListsOutput, error)
	DescribeNetworkInterfaces(context.Context, *ec2.DescribeNetworkInterfacesInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
	DescribeVpcEndpointServiceConfigurations(context.Context, *ec2.DescribeVpcEndpointServiceConfigurationsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointServiceConfigurationsOutput, error)
	DescribeVpcEndpointServicePermissions(context.Context, *ec2.DescribeVpcEndpointServicePermissionsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointServicePermissionsOutput, error)
}

type ELB interface {
	DescribeLoadBalancers(context.Context, *elasticloadbalancingv2.DescribeLoadBalancersInput, ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	DescribeListeners(context.Context, *elasticloadbalancingv2.DescribeListenersInput, ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeListenersOutput, error)
	DescribeTargetGroups(context.Context, *elasticloadbalancingv2.DescribeTargetGroupsInput, ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetGroupsOutput, error)
	DescribeTargetHealth(context.Context, *elasticloadbalancingv2.DescribeTargetHealthInput, ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetHealthOutput, error)
}

type Resolver interface {
	ListFirewallRuleGroupAssociations(context.Context, *route53resolver.ListFirewallRuleGroupAssociationsInput, ...func(*route53resolver.Options)) (*route53resolver.ListFirewallRuleGroupAssociationsOutput, error)
	GetFirewallRuleGroup(context.Context, *route53resolver.GetFirewallRuleGroupInput, ...func(*route53resolver.Options)) (*route53resolver.GetFirewallRuleGroupOutput, error)
	ListFirewallRules(context.Context, *route53resolver.ListFirewallRulesInput, ...func(*route53resolver.Options)) (*route53resolver.ListFirewallRulesOutput, error)
	ListFirewallDomains(context.Context, *route53resolver.ListFirewallDomainsInput, ...func(*route53resolver.Options)) (*route53resolver.ListFirewallDomainsOutput, error)
	GetFirewallDomainList(context.Context, *route53resolver.GetFirewallDomainListInput, ...func(*route53resolver.Options)) (*route53resolver.GetFirewallDomainListOutput, error)
	GetFirewallConfig(context.Context, *route53resolver.GetFirewallConfigInput, ...func(*route53resolver.Options)) (*route53resolver.GetFirewallConfigOutput, error)
}

type DNSProfiles interface {
	ListProfileAssociations(context.Context, *route53profiles.ListProfileAssociationsInput, ...func(*route53profiles.Options)) (*route53profiles.ListProfileAssociationsOutput, error)
}
