package ecsworker

import (
	"context"
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53profiles"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
)

type networkFixture struct {
	profiles                                      *route53profiles.ListProfileAssociationsOutput
	mu                                            sync.Mutex
	calls                                         map[string]int
	fail                                          string
	DescribeVpcsValue                             *ec2.DescribeVpcsOutput
	DescribeDhcpOptionsValue                      *ec2.DescribeDhcpOptionsOutput
	DescribeSubnetsValue                          *ec2.DescribeSubnetsOutput
	DescribeSecurityGroupsValue                   *ec2.DescribeSecurityGroupsOutput
	DescribeRouteTablesValue                      *ec2.DescribeRouteTablesOutput
	DescribeVpcEndpointsValue                     *ec2.DescribeVpcEndpointsOutput
	DescribePrefixListsValue                      *ec2.DescribePrefixListsOutput
	DescribeVpcEndpointServiceConfigurationsValue *ec2.DescribeVpcEndpointServiceConfigurationsOutput
	DescribeVpcEndpointServicePermissionsValue    *ec2.DescribeVpcEndpointServicePermissionsOutput
	DescribeLoadBalancersValue                    *elasticloadbalancingv2.DescribeLoadBalancersOutput
	DescribeListenersValue                        *elasticloadbalancingv2.DescribeListenersOutput
	DescribeTargetGroupsValue                     *elasticloadbalancingv2.DescribeTargetGroupsOutput
	DescribeTargetGroupAttributesValue            *elasticloadbalancingv2.DescribeTargetGroupAttributesOutput
	backendInput                                  *ec2.DescribeNetworkInterfacesInput
	DescribeTargetHealthValue                     *elasticloadbalancingv2.DescribeTargetHealthOutput
	ListFirewallRuleGroupAssociationsValue        *route53resolver.ListFirewallRuleGroupAssociationsOutput
	GetFirewallRuleGroupValue                     *route53resolver.GetFirewallRuleGroupOutput
	ListFirewallRulesValue                        *route53resolver.ListFirewallRulesOutput
	GetFirewallConfigValue                        *route53resolver.GetFirewallConfigOutput
	workerENI                                     *ec2.DescribeNetworkInterfacesOutput
	backendENI                                    *ec2.DescribeNetworkInterfacesOutput
	allowList                                     *route53resolver.GetFirewallDomainListOutput
	blockList                                     *route53resolver.GetFirewallDomainListOutput
	allowDomains                                  *route53resolver.ListFirewallDomainsOutput
	blockDomains                                  *route53resolver.ListFirewallDomainsOutput
	settings                                      NetworkSettings
}

func (f *networkFixture) before(ctx context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	f.calls[name]++
	if f.fail == name {
		return errors.New("private-network-canary")
	}
	return nil
}

func (f *networkFixture) DescribeVpcs(ctx context.Context, input *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	if err := f.before(ctx, "DescribeVpcs"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeVpcsValue), nil
}

func (f *networkFixture) DescribeVpcAttribute(ctx context.Context, input *ec2.DescribeVpcAttributeInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error) {
	if err := f.before(ctx, "DescribeVpcAttribute"); err != nil {
		return nil, err
	}
	value := &ec2.DescribeVpcAttributeOutput{VpcId: input.VpcId}
	if input.Attribute == "enableDnsSupport" {
		value.EnableDnsSupport = &ec2types.AttributeBooleanValue{Value: aws.Bool(true)}
	} else {
		value.EnableDnsHostnames = &ec2types.AttributeBooleanValue{Value: aws.Bool(true)}
	}
	return value, nil
}

func (f *networkFixture) DescribeDhcpOptions(ctx context.Context, input *ec2.DescribeDhcpOptionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error) {
	if err := f.before(ctx, "DescribeDhcpOptions"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeDhcpOptionsValue), nil
}

func (f *networkFixture) DescribeSubnets(ctx context.Context, input *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	if err := f.before(ctx, "DescribeSubnets"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeSubnetsValue), nil
}

func (f *networkFixture) DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	if err := f.before(ctx, "DescribeSecurityGroups"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeSecurityGroupsValue), nil
}

func (f *networkFixture) DescribeRouteTables(ctx context.Context, input *ec2.DescribeRouteTablesInput, _ ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	if err := f.before(ctx, "DescribeRouteTables"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeRouteTablesValue), nil
}

func (f *networkFixture) DescribeVpcEndpoints(ctx context.Context, input *ec2.DescribeVpcEndpointsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	if err := f.before(ctx, "DescribeVpcEndpoints"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeVpcEndpointsValue), nil
}

func (f *networkFixture) DescribePrefixLists(ctx context.Context, input *ec2.DescribePrefixListsInput, _ ...func(*ec2.Options)) (*ec2.DescribePrefixListsOutput, error) {
	if err := f.before(ctx, "DescribePrefixLists"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribePrefixListsValue), nil
}

func (f *networkFixture) DescribeNetworkInterfaces(ctx context.Context, input *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	if err := f.before(ctx, "DescribeNetworkInterfaces"); err != nil {
		return nil, err
	}
	if len(input.NetworkInterfaceIds) > 0 {
		return copyValue(f.workerENI), nil
	}
	f.backendInput = copyValue(input)
	return copyValue(f.backendENI), nil
}

func (f *networkFixture) DescribeVpcEndpointServiceConfigurations(ctx context.Context, input *ec2.DescribeVpcEndpointServiceConfigurationsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointServiceConfigurationsOutput, error) {
	if err := f.before(ctx, "DescribeVpcEndpointServiceConfigurations"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeVpcEndpointServiceConfigurationsValue), nil
}

func (f *networkFixture) DescribeVpcEndpointServicePermissions(ctx context.Context, input *ec2.DescribeVpcEndpointServicePermissionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointServicePermissionsOutput, error) {
	if err := f.before(ctx, "DescribeVpcEndpointServicePermissions"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeVpcEndpointServicePermissionsValue), nil
}

func (f *networkFixture) DescribeLoadBalancers(ctx context.Context, input *elasticloadbalancingv2.DescribeLoadBalancersInput, _ ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	if err := f.before(ctx, "DescribeLoadBalancers"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeLoadBalancersValue), nil
}

func (f *networkFixture) DescribeListeners(ctx context.Context, input *elasticloadbalancingv2.DescribeListenersInput, _ ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeListenersOutput, error) {
	if err := f.before(ctx, "DescribeListeners"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeListenersValue), nil
}

func (f *networkFixture) DescribeTargetGroups(ctx context.Context, input *elasticloadbalancingv2.DescribeTargetGroupsInput, _ ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetGroupsOutput, error) {
	if err := f.before(ctx, "DescribeTargetGroups"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeTargetGroupsValue), nil
}

func (f *networkFixture) DescribeTargetHealth(ctx context.Context, input *elasticloadbalancingv2.DescribeTargetHealthInput, _ ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetHealthOutput, error) {
	if err := f.before(ctx, "DescribeTargetHealth"); err != nil {
		return nil, err
	}
	return copyValue(f.DescribeTargetHealthValue), nil
}

func (f *networkFixture) ListFirewallRuleGroupAssociations(ctx context.Context, input *route53resolver.ListFirewallRuleGroupAssociationsInput, _ ...func(*route53resolver.Options)) (*route53resolver.ListFirewallRuleGroupAssociationsOutput, error) {
	if err := f.before(ctx, "ListFirewallRuleGroupAssociations"); err != nil {
		return nil, err
	}
	return copyValue(f.ListFirewallRuleGroupAssociationsValue), nil
}

func (f *networkFixture) GetFirewallRuleGroup(ctx context.Context, input *route53resolver.GetFirewallRuleGroupInput, _ ...func(*route53resolver.Options)) (*route53resolver.GetFirewallRuleGroupOutput, error) {
	if err := f.before(ctx, "GetFirewallRuleGroup"); err != nil {
		return nil, err
	}
	return copyValue(f.GetFirewallRuleGroupValue), nil
}

func (f *networkFixture) ListFirewallRules(ctx context.Context, input *route53resolver.ListFirewallRulesInput, _ ...func(*route53resolver.Options)) (*route53resolver.ListFirewallRulesOutput, error) {
	if err := f.before(ctx, "ListFirewallRules"); err != nil {
		return nil, err
	}
	return copyValue(f.ListFirewallRulesValue), nil
}

func (f *networkFixture) ListFirewallDomains(ctx context.Context, input *route53resolver.ListFirewallDomainsInput, _ ...func(*route53resolver.Options)) (*route53resolver.ListFirewallDomainsOutput, error) {
	if err := f.before(ctx, "ListFirewallDomains"); err != nil {
		return nil, err
	}
	if *input.FirewallDomainListId == f.settings.AllowList {
		return copyValue(f.allowDomains), nil
	}
	return copyValue(f.blockDomains), nil
}

func (f *networkFixture) GetFirewallDomainList(ctx context.Context, input *route53resolver.GetFirewallDomainListInput, _ ...func(*route53resolver.Options)) (*route53resolver.GetFirewallDomainListOutput, error) {
	if err := f.before(ctx, "GetFirewallDomainList"); err != nil {
		return nil, err
	}
	if *input.FirewallDomainListId == f.settings.AllowList {
		return copyValue(f.allowList), nil
	}
	return copyValue(f.blockList), nil
}

func (f *networkFixture) GetFirewallConfig(ctx context.Context, input *route53resolver.GetFirewallConfigInput, _ ...func(*route53resolver.Options)) (*route53resolver.GetFirewallConfigOutput, error) {
	if err := f.before(ctx, "GetFirewallConfig"); err != nil {
		return nil, err
	}
	return copyValue(f.GetFirewallConfigValue), nil
}

func (f *networkFixture) ListProfileAssociations(ctx context.Context, input *route53profiles.ListProfileAssociationsInput, options ...func(*route53profiles.Options)) (*route53profiles.ListProfileAssociationsOutput, error) {
	if err := f.before(ctx, "ListProfileAssociations"); err != nil {
		return nil, err
	}
	config := route53profiles.Options{}
	for _, option := range options {
		option(&config)
	}
	if aws.ToString(input.ResourceId) != f.settings.VPC || config.RetryMaxAttempts != 1 {
		return nil, errors.New("invalid profile request")
	}
	if f.profiles == nil {
		return &route53profiles.ListProfileAssociationsOutput{}, nil
	}
	return copyValue(f.profiles), nil
}

func (f *networkFixture) DescribeTargetGroupAttributes(ctx context.Context, input *elasticloadbalancingv2.DescribeTargetGroupAttributesInput, options ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetGroupAttributesOutput, error) {
	if err := f.before(ctx, "DescribeTargetGroupAttributes"); err != nil {
		return nil, err
	}
	var config elasticloadbalancingv2.Options
	for _, option := range options {
		option(&config)
	}
	if aws.ToString(input.TargetGroupArn) != f.settings.TargetGroup || config.RetryMaxAttempts != 1 {
		return nil, errors.New("unexpected attribute scope")
	}
	return copyValue(f.DescribeTargetGroupAttributesValue), nil
}
