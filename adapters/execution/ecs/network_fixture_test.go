package ecsworker

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	l "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	d "github.com/aws/aws-sdk-go-v2/service/route53resolver/types"
)

func policy(value any) *string { data, _ := json.Marshal(value); return aws.String(string(data)) }
func sgRule(account, id string, port int32) e.IpPermission {
	return e.IpPermission{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(port), ToPort: aws.Int32(port), UserIdGroupPairs: []e.UserIdGroupPair{{GroupId: aws.String(id), UserId: aws.String(account)}}}
}
func network(t *testing.T) (*NetworkGuard, *networkFixture, Config, types.Task) {
	t.Helper()
	c, _, _ := fixture(t, "go", "")
	s := NetworkSettings{VPC: "vpc-aaaaaaaa", ControllerVPC: "vpc-bbbbbbbb", ExchangeGroup: "sg-bbbbbbbb", RegistryGroup: "sg-cccccccc", LoadBalancerGroup: "sg-dddddddd", ControllerGroup: "sg-eeeeeeee", ExchangeEndpoint: "vpce-11111111111111111", APIEndpoint: "vpce-22222222222222222", DKREndpoint: "vpce-33333333333333333", S3Endpoint: "vpce-44444444444444444", S3PrefixList: "pl-aaaaaaaa", RepositoryPrefix: "teststudio", ExchangeService: "com.amazonaws.vpce.us-east-1.vpce-svc-55555555555555555", LoadBalancer: "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/worker/aaaaaaaaaaaaaaaa", TargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/worker/bbbbbbbbbbbbbbbb", FirewallGroup: "rslvr-frg-11111111111111111", FirewallAssociation: "rslvr-frgassoc-22222222222222222", AllowList: "rslvr-fdl-33333333333333333", BlockList: "rslvr-fdl-44444444444444444"}
	var err error
	c, err = s.Bind(c)
	if err != nil {
		t.Fatal(err)
	}
	f := &networkFixture{calls: map[string]int{}, settings: s}
	f.DescribeVpcsValue = &ec2.DescribeVpcsOutput{Vpcs: []e.Vpc{{VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), State: "available", CidrBlock: aws.String("10.40.0.0/16"), DhcpOptionsId: aws.String("dopt-aaaaaaaa")}}}
	f.DescribeDhcpOptionsValue = &ec2.DescribeDhcpOptionsOutput{DhcpOptions: []e.DhcpOptions{{DhcpOptionsId: aws.String("dopt-aaaaaaaa"), OwnerId: aws.String(c.Account), DhcpConfigurations: []e.DhcpConfiguration{{Key: aws.String("domain-name-servers"), Values: []e.AttributeValue{{Value: aws.String("AmazonProvidedDNS")}}}}}}}
	f.DescribeSubnetsValue = &ec2.DescribeSubnetsOutput{Subnets: []e.Subnet{
		{SubnetId: aws.String(c.Subnets[0]), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), State: "available", CidrBlock: aws.String("10.40.1.0/24")},
		{SubnetId: aws.String(c.Subnets[1]), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), State: "available", CidrBlock: aws.String("10.40.2.0/24")},
	}}
	f.DescribeRouteTablesValue = &ec2.DescribeRouteTablesOutput{RouteTables: []e.RouteTable{{RouteTableId: aws.String("rtb-aaaaaaaa"), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), Associations: []e.RouteTableAssociation{{Main: aws.Bool(true)}}, Routes: []e.Route{{GatewayId: aws.String("local"), DestinationCidrBlock: aws.String("10.40.0.0/16"), State: "active"}, {GatewayId: aws.String(s.S3Endpoint), DestinationPrefixListId: aws.String(s.S3PrefixList), State: "active"}}}}}
	f.DescribeSecurityGroupsValue = &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []e.SecurityGroup{
		{GroupId: aws.String(c.SecurityGroups[0]), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), IpPermissionsEgress: []e.IpPermission{sgRule(c.Account, s.ExchangeGroup, 443), sgRule(c.Account, s.RegistryGroup, 443), {IpProtocol: aws.String("tcp"), FromPort: aws.Int32(443), ToPort: aws.Int32(443), PrefixListIds: []e.PrefixListId{{PrefixListId: aws.String(s.S3PrefixList)}}}}},
		{GroupId: aws.String(s.ExchangeGroup), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), IpPermissions: []e.IpPermission{sgRule(c.Account, c.SecurityGroups[0], 443)}, IpPermissionsEgress: []e.IpPermission{sgRule(c.Account, c.SecurityGroups[0], 443)}},
		{GroupId: aws.String(s.RegistryGroup), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), IpPermissions: []e.IpPermission{sgRule(c.Account, c.SecurityGroups[0], 443)}, IpPermissionsEgress: []e.IpPermission{sgRule(c.Account, c.SecurityGroups[0], 443)}},
		{GroupId: aws.String(s.LoadBalancerGroup), VpcId: aws.String(s.ControllerVPC), OwnerId: aws.String(c.Account), IpPermissionsEgress: []e.IpPermission{sgRule(c.Account, s.ControllerGroup, 8042)}},
		{GroupId: aws.String(s.ControllerGroup), VpcId: aws.String(s.ControllerVPC), OwnerId: aws.String(c.Account), IpPermissions: []e.IpPermission{sgRule(c.Account, s.LoadBalancerGroup, 8042)}},
	}}
	f.DescribePrefixListsValue = &ec2.DescribePrefixListsOutput{PrefixLists: []e.PrefixList{{PrefixListId: aws.String(s.S3PrefixList), PrefixListName: aws.String("com.amazonaws.us-east-1.s3")}}}
	f.DescribeVpcEndpointsValue = &ec2.DescribeVpcEndpointsOutput{}
	for id, service := range map[string]string{s.APIEndpoint: "com.amazonaws.us-east-1.ecr.api", s.DKREndpoint: "com.amazonaws.us-east-1.ecr.dkr", s.ExchangeEndpoint: s.ExchangeService} {
		group := s.RegistryGroup
		if id == s.ExchangeEndpoint {
			group = s.ExchangeGroup
		}
		f.DescribeVpcEndpointsValue.VpcEndpoints = append(f.DescribeVpcEndpointsValue.VpcEndpoints, e.VpcEndpoint{VpcEndpointId: aws.String(id), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), State: e.StateAvailable, ServiceName: aws.String(service), VpcEndpointType: "Interface", PrivateDnsEnabled: aws.Bool(true), IpAddressType: "ipv4", Groups: []e.SecurityGroupIdentifier{{GroupId: aws.String(group)}}, SubnetIds: c.Subnets, NetworkInterfaceIds: []string{"eni-bbbbbbbb", "eni-cccccccc"}, PolicyDocument: policy(s.ECRPolicy(c))})
	}
	f.DescribeVpcEndpointsValue.VpcEndpoints = append(f.DescribeVpcEndpointsValue.VpcEndpoints, e.VpcEndpoint{VpcEndpointId: aws.String(s.S3Endpoint), VpcId: aws.String(s.VPC), OwnerId: aws.String(c.Account), State: e.StateAvailable, ServiceName: aws.String("com.amazonaws.us-east-1.s3"), VpcEndpointType: "Gateway", IpAddressType: "ipv4", RouteTableIds: []string{"rtb-aaaaaaaa"}, PolicyDocument: policy(s.S3Policy(c))})
	f.DescribeVpcEndpointServiceConfigurationsValue = &ec2.DescribeVpcEndpointServiceConfigurationsOutput{ServiceConfigurations: []e.ServiceConfiguration{{ServiceId: aws.String("vpce-svc-55555555555555555"), ServiceName: aws.String(s.ExchangeService), ServiceState: "Available", AcceptanceRequired: aws.Bool(true), ServiceType: []e.ServiceTypeDetail{{ServiceType: "Interface"}}, SupportedIpAddressTypes: []e.ServiceConnectivityType{"ipv4"}, NetworkLoadBalancerArns: []string{s.LoadBalancer}, PrivateDnsName: aws.String(s.Domains(c)[0]), PrivateDnsNameConfiguration: &e.PrivateDnsNameConfiguration{State: "verified"}}}}
	f.DescribeVpcEndpointServicePermissionsValue = &ec2.DescribeVpcEndpointServicePermissionsOutput{AllowedPrincipals: []e.AllowedPrincipal{{Principal: aws.String("arn:aws:iam::123456789012:root")}}}
	f.DescribeLoadBalancersValue = &elasticloadbalancingv2.DescribeLoadBalancersOutput{LoadBalancers: []l.LoadBalancer{{LoadBalancerArn: aws.String(s.LoadBalancer), VpcId: aws.String(s.ControllerVPC), Type: "network", Scheme: "internal", IpAddressType: "ipv4", State: &l.LoadBalancerState{Code: "active"}, SecurityGroups: []string{s.LoadBalancerGroup}, EnforceSecurityGroupInboundRulesOnPrivateLinkTraffic: aws.String("off")}}}
	f.DescribeListenersValue = &elasticloadbalancingv2.DescribeListenersOutput{Listeners: []l.Listener{{LoadBalancerArn: aws.String(s.LoadBalancer), Port: aws.Int32(443), Protocol: "TLS", SslPolicy: aws.String("ELBSecurityPolicy-TLS13-1-2-2021-06"), Certificates: []l.Certificate{{CertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")}}, DefaultActions: []l.Action{{Type: "forward", TargetGroupArn: aws.String(s.TargetGroup)}}}}}
	f.DescribeTargetGroupsValue = &elasticloadbalancingv2.DescribeTargetGroupsOutput{TargetGroups: []l.TargetGroup{{TargetGroupArn: aws.String(s.TargetGroup), VpcId: aws.String(s.ControllerVPC), Protocol: "TCP", Port: aws.Int32(8042), TargetType: "ip", IpAddressType: "ipv4", LoadBalancerArns: []string{s.LoadBalancer}, HealthCheckEnabled: aws.Bool(true), HealthCheckProtocol: "HTTP", HealthCheckPort: aws.String("traffic-port"), HealthCheckPath: aws.String("/healthz"), Matcher: &l.Matcher{HttpCode: aws.String("404")}}}}
	f.DescribeTargetHealthValue = &elasticloadbalancingv2.DescribeTargetHealthOutput{TargetHealthDescriptions: []l.TargetHealthDescription{{Target: &l.TargetDescription{Id: aws.String("10.50.1.11"), Port: aws.Int32(8042)}, TargetHealth: &l.TargetHealth{State: "healthy"}}}}
	f.backendENI = &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []e.NetworkInterface{{OwnerId: aws.String(c.Account), VpcId: aws.String(s.ControllerVPC), PrivateIpAddress: aws.String("10.50.1.11"), Groups: []e.GroupIdentifier{{GroupId: aws.String(s.ControllerGroup)}}}}}
	f.workerENI = &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []e.NetworkInterface{{NetworkInterfaceId: aws.String("eni-aaaaaaaa"), OwnerId: aws.String(c.Account), VpcId: aws.String(s.VPC), SubnetId: aws.String(c.Subnets[0]), PrivateIpAddress: aws.String("10.40.1.21"), Groups: []e.GroupIdentifier{{GroupId: aws.String(c.SecurityGroups[0])}}, InterfaceType: "interface", Status: "in-use", SourceDestCheck: aws.Bool(true)}}}
	f.GetFirewallConfigValue = &route53resolver.GetFirewallConfigOutput{FirewallConfig: &d.FirewallConfig{OwnerId: aws.String(c.Account), ResourceId: aws.String(s.VPC), FirewallFailOpen: "DISABLED"}}
	f.ListFirewallRuleGroupAssociationsValue = &route53resolver.ListFirewallRuleGroupAssociationsOutput{FirewallRuleGroupAssociations: []d.FirewallRuleGroupAssociation{{Id: aws.String(s.FirewallAssociation), Arn: aws.String("arn:aws:route53resolver:" + c.Region + ":" + c.Account + ":firewall-rule-group-association/" + s.FirewallAssociation), VpcId: aws.String(s.VPC), FirewallRuleGroupId: aws.String(s.FirewallGroup), Status: "COMPLETE", MutationProtection: "ENABLED"}}}
	f.GetFirewallRuleGroupValue = &route53resolver.GetFirewallRuleGroupOutput{FirewallRuleGroup: &d.FirewallRuleGroup{Id: aws.String(s.FirewallGroup), Arn: aws.String("arn:aws:route53resolver:" + c.Region + ":" + c.Account + ":firewall-rule-group/" + s.FirewallGroup), OwnerId: aws.String(c.Account), Status: "COMPLETE"}}
	f.ListFirewallRulesValue = &route53resolver.ListFirewallRulesOutput{FirewallRules: []d.FirewallRule{{FirewallRuleGroupId: aws.String(s.FirewallGroup), FirewallDomainListId: aws.String(s.AllowList), Priority: aws.Int32(100), Action: "ALLOW", FirewallDomainRedirectionAction: "TRUST_REDIRECTION_DOMAIN"}, {FirewallRuleGroupId: aws.String(s.FirewallGroup), FirewallDomainListId: aws.String(s.BlockList), Priority: aws.Int32(200), Action: "BLOCK", BlockResponse: "NXDOMAIN", FirewallDomainRedirectionAction: "TRUST_REDIRECTION_DOMAIN"}}}
	f.allowList = &route53resolver.GetFirewallDomainListOutput{FirewallDomainList: &d.FirewallDomainList{Id: aws.String(s.AllowList), Arn: aws.String("arn:aws:route53resolver:" + c.Region + ":" + c.Account + ":firewall-domain-list/" + s.AllowList), DomainCount: aws.Int32(5), Status: "COMPLETE"}}
	f.blockList = &route53resolver.GetFirewallDomainListOutput{FirewallDomainList: &d.FirewallDomainList{Id: aws.String(s.BlockList), Arn: aws.String("arn:aws:route53resolver:" + c.Region + ":" + c.Account + ":firewall-domain-list/" + s.BlockList), DomainCount: aws.Int32(1), Status: "COMPLETE"}}
	f.allowDomains = &route53resolver.ListFirewallDomainsOutput{Domains: s.Domains(c)}
	f.blockDomains = &route53resolver.ListFirewallDomainsOutput{Domains: []string{"*"}}
	g, err := NewNetworkGuard(c, s, f, f, f, f)
	if err != nil {
		t.Fatal(err)
	}
	task := types.Task{Attachments: []types.Attachment{{Id: aws.String("attach-one"), Type: aws.String("ElasticNetworkInterface"), Status: aws.String("ATTACHED"), Details: []types.KeyValuePair{{Name: aws.String("networkInterfaceId"), Value: aws.String("eni-aaaaaaaa")}, {Name: aws.String("subnetId"), Value: aws.String(c.Subnets[0])}, {Name: aws.String("privateIPv4Address"), Value: aws.String("10.40.1.21")}}}}, Containers: []types.Container{{NetworkInterfaces: []types.NetworkInterface{{AttachmentId: aws.String("attach-one"), PrivateIpv4Address: aws.String("10.40.1.21")}}}}}
	return g, f, c, task
}
