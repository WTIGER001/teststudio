package ecsworker

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type packageObservations struct {
	VPCs      []e.Vpc                  `json:"vpcs"`
	DHCP      []e.DhcpOptions          `json:"dhcp"`
	Subnets   []e.Subnet               `json:"subnets"`
	Groups    []e.SecurityGroup        `json:"groups"`
	Routes    []e.RouteTable           `json:"routes"`
	Endpoints []e.VpcEndpoint          `json:"endpoints"`
	Peers     []e.VpcPeeringConnection `json:"peers"`
	NATs      []e.NatGateway           `json:"nats"`
	Internet  []e.InternetGateway      `json:"internet"`
}

type packageEC2 struct {
	*networkFixture
	values     packageObservations
	gatewayENI e.NetworkInterface
}

func selectPackage[T any](values []T, ids []string, identify func(T) string) []T {
	result := []T{}
	for _, value := range values {
		for _, id := range ids {
			if identify(value) == id {
				result = append(result, copyValue(value))
				break
			}
		}
	}
	return result
}

func (f *packageEC2) DescribeVpcs(_ context.Context, input *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &ec2.DescribeVpcsOutput{Vpcs: selectPackage(f.values.VPCs, input.VpcIds, func(v e.Vpc) string { return aws.ToString(v.VpcId) })}, nil
}
func (f *packageEC2) DescribeDhcpOptions(_ context.Context, input *ec2.DescribeDhcpOptionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error) {
	return &ec2.DescribeDhcpOptionsOutput{DhcpOptions: selectPackage(f.values.DHCP, input.DhcpOptionsIds, func(v e.DhcpOptions) string { return aws.ToString(v.DhcpOptionsId) })}, nil
}
func (f *packageEC2) DescribeSubnets(_ context.Context, input *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &ec2.DescribeSubnetsOutput{Subnets: selectPackage(f.values.Subnets, input.SubnetIds, func(v e.Subnet) string { return aws.ToString(v.SubnetId) })}, nil
}
func (f *packageEC2) DescribeSecurityGroups(_ context.Context, input *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: selectPackage(f.values.Groups, input.GroupIds, func(v e.SecurityGroup) string { return aws.ToString(v.GroupId) })}, nil
}
func (f *packageEC2) DescribeRouteTables(_ context.Context, input *ec2.DescribeRouteTablesInput, _ ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	return &ec2.DescribeRouteTablesOutput{RouteTables: selectPackage(f.values.Routes, input.Filters[0].Values, func(v e.RouteTable) string { return aws.ToString(v.VpcId) })}, nil
}
func (f *packageEC2) DescribeVpcEndpoints(_ context.Context, input *ec2.DescribeVpcEndpointsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	return &ec2.DescribeVpcEndpointsOutput{VpcEndpoints: selectPackage(f.values.Endpoints, input.VpcEndpointIds, func(v e.VpcEndpoint) string { return aws.ToString(v.VpcEndpointId) })}, nil
}
func (f *packageEC2) DescribeVpcPeeringConnections(_ context.Context, input *ec2.DescribeVpcPeeringConnectionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	return &ec2.DescribeVpcPeeringConnectionsOutput{VpcPeeringConnections: selectPackage(f.values.Peers, input.VpcPeeringConnectionIds, func(v e.VpcPeeringConnection) string { return aws.ToString(v.VpcPeeringConnectionId) })}, nil
}
func (f *packageEC2) DescribeNatGateways(_ context.Context, input *ec2.DescribeNatGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	return &ec2.DescribeNatGatewaysOutput{NatGateways: selectPackage(f.values.NATs, input.NatGatewayIds, func(v e.NatGateway) string { return aws.ToString(v.NatGatewayId) })}, nil
}
func (f *packageEC2) DescribeInternetGateways(_ context.Context, input *ec2.DescribeInternetGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
	return &ec2.DescribeInternetGatewaysOutput{InternetGateways: selectPackage(f.values.Internet, input.InternetGatewayIds, func(v e.InternetGateway) string { return aws.ToString(v.InternetGatewayId) })}, nil
}
func (f *packageEC2) DescribeNetworkInterfaces(ctx context.Context, input *ec2.DescribeNetworkInterfacesInput, options ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	if len(input.NetworkInterfaceIds) == 1 && input.NetworkInterfaceIds[0] == aws.ToString(f.gatewayENI.NetworkInterfaceId) {
		return &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []e.NetworkInterface{copyValue(f.gatewayENI)}}, nil
	}
	return f.networkFixture.DescribeNetworkInterfaces(ctx, input, options...)
}

func packageDNS(f *networkFixture, config Config, network NetworkSettings) {
	f.settings = network
	base := "arn:aws:route53resolver:" + config.Region + ":" + config.Account + ":"
	f.GetFirewallConfigValue.FirewallConfig.ResourceId = aws.String(network.VPC)
	association := &f.ListFirewallRuleGroupAssociationsValue.FirewallRuleGroupAssociations[0]
	association.Id, association.Arn, association.VpcId, association.FirewallRuleGroupId = aws.String(network.FirewallAssociation), aws.String(base+"firewall-rule-group-association/"+network.FirewallAssociation), aws.String(network.VPC), aws.String(network.FirewallGroup)
	f.GetFirewallRuleGroupValue.FirewallRuleGroup.Id, f.GetFirewallRuleGroupValue.FirewallRuleGroup.Arn = aws.String(network.FirewallGroup), aws.String(base+"firewall-rule-group/"+network.FirewallGroup)
	for index, id := range []string{network.AllowList, network.BlockList} {
		f.ListFirewallRulesValue.FirewallRules[index].FirewallRuleGroupId = aws.String(network.FirewallGroup)
		f.ListFirewallRulesValue.FirewallRules[index].FirewallDomainListId = aws.String(id)
		list := f.allowList
		if index == 1 {
			list = f.blockList
		}
		list.FirewallDomainList.Id, list.FirewallDomainList.Arn = aws.String(id), aws.String(base+"firewall-domain-list/"+id)
	}
	f.allowDomains.Domains = network.Domains(config)
}

func TestECSEnvironmentPackageMatchesNativeNetworkAndTaskAdmission(t *testing.T) {
	file := os.Getenv("TESTSTUDIO_ECS_ENVIRONMENT_PACKAGE_FIXTURE")
	if file == "" {
		t.Skip("requires the local environment package preview")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Settings     Settings
		Observations packageObservations
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	settings, err := fixture.Settings.bind()
	if err != nil || settings.Environment == nil {
		t.Fatal("rendered settings did not bind", err)
	}
	_, base, _, task := network(t)
	env := settings.Environment
	packageDNS(base, env.Worker, env.Network)
	client := &packageEC2{networkFixture: base, values: fixture.Observations}
	client.values.Groups = append(client.values.Groups, copyValue(base.DescribeSecurityGroupsValue.SecurityGroups[4]))
	subnet := selectPackage(client.values.Subnets, env.Gateway.Subnets[:1], func(v e.Subnet) string { return aws.ToString(v.SubnetId) })[0]
	prefix, err := netip.ParsePrefix(aws.ToString(subnet.CidrBlock))
	if err != nil {
		t.Fatal(err)
	}
	ip := prefix.Addr()
	for range 21 {
		ip = ip.Next()
	}
	client.gatewayENI = e.NetworkInterface{NetworkInterfaceId: aws.String("eni-abcdef12345678901"), VpcId: aws.String(env.Network.Gateway.VPC), SubnetId: subnet.SubnetId, OwnerId: aws.String(env.Worker.Account), PrivateIpAddress: aws.String(ip.String()), Groups: []e.GroupIdentifier{{GroupId: aws.String(env.Gateway.SecurityGroup)}}, InterfaceType: "interface", Status: "in-use", SourceDestCheck: aws.Bool(true)}
	task.Attachments[0].Details = []types.KeyValuePair{{Name: aws.String("networkInterfaceId"), Value: client.gatewayENI.NetworkInterfaceId}, {Name: aws.String("subnetId"), Value: subnet.SubnetId}, {Name: aws.String("privateIPv4Address"), Value: client.gatewayENI.PrivateIpAddress}}
	task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address = client.gatewayENI.PrivateIpAddress
	guard, err := NewNetworkGuard(env.Worker, env.Network, client, base, base, base)
	if err != nil {
		t.Fatal(err)
	}
	address, err := guard.VerifyGatewayTask(t.Context(), env.Worker, env.Gateway, task)
	if err != nil || address != ip.String() {
		t.Fatal("rendered environment disagrees with native admission", address, err)
	}
}
