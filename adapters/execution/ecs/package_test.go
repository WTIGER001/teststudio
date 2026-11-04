package ecsworker

import (
	"encoding/json"
	"net/netip"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	l "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	d "github.com/aws/aws-sdk-go-v2/service/route53resolver/types"
)

type packageProperties struct {
	VpcId                                                                                  string
	CidrBlock                                                                              string
	SecurityGroupIngress, SecurityGroupEgress                                              []packagePermission
	SubnetIds, SecurityGroupIds                                                            []string
	IpAddressType, ServiceName, VpcEndpointType                                            string
	PrivateDnsEnabled                                                                      bool
	RouteTableIds                                                                          []string
	PolicyDocument                                                                         json.RawMessage
	Scheme, Type, EnforceSecurityGroupInboundRulesOnPrivateLinkTraffic                     string
	SecurityGroups                                                                         []string
	Protocol, SslPolicy, TargetType, HealthCheckPort, HealthCheckProtocol, HealthCheckPath string
	Port                                                                                   int32
	HealthCheckEnabled                                                                     bool
	Matcher                                                                                struct{ HttpCode string }
	Certificates                                                                           []struct{ CertificateArn string }
	DefaultActions                                                                         []struct{ Type, TargetGroupArn string }
	AcceptanceRequired                                                                     bool
	AllowedPrincipals, Domains                                                             []string
	FirewallRules                                                                          []struct {
		Action, BlockResponse, FirewallDomainListId, FirewallDomainRedirectionAction string
		Priority                                                                     int32
	}
	TargetGroupAttributes []struct{ Key, Value string }
	MutationProtection    string
}

type packagePermission struct {
	IpProtocol                                                                                             string
	FromPort, ToPort                                                                                       int32
	CidrIp, SourceSecurityGroupId, DestinationSecurityGroupId, SourcePrefixListId, DestinationPrefixListId string
}

func packagedPermissions(values []packagePermission, account string) []e.IpPermission {
	result := []e.IpPermission{}
	for _, value := range values {
		permission := e.IpPermission{IpProtocol: aws.String(value.IpProtocol), FromPort: aws.Int32(value.FromPort), ToPort: aws.Int32(value.ToPort)}
		for _, group := range []string{value.SourceSecurityGroupId, value.DestinationSecurityGroupId} {
			if group != "" {
				permission.UserIdGroupPairs = append(permission.UserIdGroupPairs, e.UserIdGroupPair{GroupId: aws.String(group), UserId: aws.String(account)})
			}
		}
		for _, prefix := range []string{value.SourcePrefixListId, value.DestinationPrefixListId} {
			if prefix != "" {
				permission.PrefixListIds = append(permission.PrefixListIds, e.PrefixListId{PrefixListId: aws.String(prefix)})
			}
		}
		if value.CidrIp != "" {
			permission.IpRanges = []e.IpRange{{CidrIp: aws.String(value.CidrIp)}}
		}
		result = append(result, permission)
	}
	return result
}

func TestPackagedECSNetworkMatchesAdmissionAfterDocumentedDNSSetup(t *testing.T) {
	file := os.Getenv("TESTSTUDIO_AWS_PACKAGE_FIXTURE")
	if file == "" {
		t.Skip("requires the local CloudFormation preview fixture")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Settings   Settings
		Properties map[string]packageProperties
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	s := fixture.Settings
	c, err := s.Network.Bind(s.Worker)
	if err != nil {
		t.Fatal(err)
	}
	_, f, _, task := network(t)
	f.settings = s.Network
	p := fixture.Properties
	f.DescribeVpcsValue.Vpcs[0].CidrBlock = aws.String(p["WorkerVpc"].CidrBlock)
	for index, name := range []string{"WorkerSubnetOne", "WorkerSubnetTwo"} {
		f.DescribeSubnetsValue.Subnets[index].CidrBlock = aws.String(p[name].CidrBlock)
		f.DescribeSubnetsValue.Subnets[index].VpcId = aws.String(p[name].VpcId)
	}
	f.DescribeRouteTablesValue.RouteTables[0].Routes[0].DestinationCidrBlock = aws.String(p["WorkerVpc"].CidrBlock)
	cidr, err := netip.ParsePrefix(p["WorkerSubnetOne"].CidrBlock)
	if err != nil {
		t.Fatal(err)
	}
	address := cidr.Addr()
	for range 21 {
		address = address.Next()
	}
	f.workerENI.NetworkInterfaces[0].PrivateIpAddress = aws.String(address.String())
	for index, field := range task.Attachments[0].Details {
		if aws.ToString(field.Name) == "privateIPv4Address" {
			task.Attachments[0].Details[index].Value = aws.String(address.String())
		}
	}
	task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address = aws.String(address.String())
	for index, name := range []string{"WorkerGroup", "ExchangeGroup", "RegistryGroup", "ExchangeLoadBalancerGroup"} {
		group := &f.DescribeSecurityGroupsValue.SecurityGroups[index]
		value := p[name]
		group.VpcId = aws.String(value.VpcId)
		group.IpPermissions = packagedPermissions(value.SecurityGroupIngress, c.Account)
		egress := value.SecurityGroupEgress
		if name == "WorkerGroup" {
			for _, resource := range []string{"WorkerExchangeEgress", "WorkerRegistryEgress"} {
				var permission packagePermission
				var raw struct{ Properties map[string]json.RawMessage }
				json.Unmarshal(data, &raw)
				if json.Unmarshal(raw.Properties[resource], &permission) != nil {
					t.Fatal("invalid external group rule")
				}
				egress = append(egress, permission)
			}
		}
		group.IpPermissionsEgress = packagedPermissions(egress, c.Account)
	}
	var raw struct{ Properties map[string]json.RawMessage }
	json.Unmarshal(data, &raw)
	var ingress packagePermission
	json.Unmarshal(raw.Properties["ControllerWorkerIngress"], &ingress)
	f.DescribeSecurityGroupsValue.SecurityGroups[4].IpPermissions = packagedPermissions([]packagePermission{ingress}, c.Account)
	for index := range f.DescribeVpcEndpointsValue.VpcEndpoints {
		endpoint := &f.DescribeVpcEndpointsValue.VpcEndpoints[index]
		name := map[string]string{s.Network.APIEndpoint: "EcrApi", s.Network.DKREndpoint: "EcrDocker", s.Network.S3Endpoint: "ImageLayers", s.Network.ExchangeEndpoint: "ExchangeEndpoint"}[aws.ToString(endpoint.VpcEndpointId)]
		value := p[name]
		endpoint.VpcId = aws.String(value.VpcId)
		endpoint.SubnetIds = value.SubnetIds
		endpoint.RouteTableIds = value.RouteTableIds
		endpoint.IpAddressType = e.IpAddressType(value.IpAddressType)
		endpoint.PrivateDnsEnabled = aws.Bool(value.PrivateDnsEnabled)
		endpoint.VpcEndpointType = e.VpcEndpointType(value.VpcEndpointType)
		endpoint.ServiceName = aws.String(value.ServiceName)
		endpoint.Groups = nil
		for _, id := range value.SecurityGroupIds {
			endpoint.Groups = append(endpoint.Groups, e.SecurityGroupIdentifier{GroupId: aws.String(id)})
		}
		if len(value.PolicyDocument) > 0 {
			endpoint.PolicyDocument = aws.String(string(value.PolicyDocument))
		}
	}
	f.DescribeVpcEndpointServiceConfigurationsValue.ServiceConfigurations[0].AcceptanceRequired = aws.Bool(p["ExchangeService"].AcceptanceRequired)
	f.DescribeVpcEndpointServicePermissionsValue.AllowedPrincipals = nil
	for _, principal := range p["ExchangePermissions"].AllowedPrincipals {
		f.DescribeVpcEndpointServicePermissionsValue.AllowedPrincipals = append(f.DescribeVpcEndpointServicePermissionsValue.AllowedPrincipals, e.AllowedPrincipal{Principal: aws.String(principal)})
	}
	loadBalancer := &f.DescribeLoadBalancersValue.LoadBalancers[0]
	value := p["ExchangeLoadBalancer"]
	loadBalancer.Scheme = l.LoadBalancerSchemeEnum(value.Scheme)
	loadBalancer.Type = l.LoadBalancerTypeEnum(value.Type)
	loadBalancer.IpAddressType = l.IpAddressType(value.IpAddressType)
	loadBalancer.SecurityGroups = value.SecurityGroups
	loadBalancer.EnforceSecurityGroupInboundRulesOnPrivateLinkTraffic = aws.String(value.EnforceSecurityGroupInboundRulesOnPrivateLinkTraffic)
	listener := &f.DescribeListenersValue.Listeners[0]
	value = p["ExchangeListener"]
	listener.Port = aws.Int32(value.Port)
	listener.Protocol = l.ProtocolEnum(value.Protocol)
	listener.SslPolicy = aws.String(value.SslPolicy)
	listener.Certificates = nil
	for _, certificate := range value.Certificates {
		listener.Certificates = append(listener.Certificates, l.Certificate{CertificateArn: aws.String(certificate.CertificateArn)})
	}
	listener.DefaultActions = nil
	for _, action := range value.DefaultActions {
		listener.DefaultActions = append(listener.DefaultActions, l.Action{Type: l.ActionTypeEnum(action.Type), TargetGroupArn: aws.String(action.TargetGroupArn)})
	}
	target := &f.DescribeTargetGroupsValue.TargetGroups[0]
	value = p["ExchangeTarget"]
	target.VpcId = aws.String(value.VpcId)
	target.Port = aws.Int32(value.Port)
	target.Protocol = l.ProtocolEnum(value.Protocol)
	target.TargetType = l.TargetTypeEnum(value.TargetType)
	target.IpAddressType = l.TargetGroupIpAddressTypeEnum(value.IpAddressType)
	target.HealthCheckEnabled = aws.Bool(value.HealthCheckEnabled)
	target.HealthCheckProtocol = l.ProtocolEnum(value.HealthCheckProtocol)
	target.HealthCheckPort = aws.String(value.HealthCheckPort)
	target.HealthCheckPath = aws.String(value.HealthCheckPath)
	target.Matcher = &l.Matcher{HttpCode: aws.String(value.Matcher.HttpCode)}
	f.DescribeTargetGroupAttributesValue.Attributes = nil
	for _, attribute := range value.TargetGroupAttributes {
		f.DescribeTargetGroupAttributesValue.Attributes = append(f.DescribeTargetGroupAttributesValue.Attributes, l.TargetGroupAttribute{Key: aws.String(attribute.Key), Value: aws.String(attribute.Value)})
	}
	f.allowDomains.Domains = p["AllowedDomains"].Domains
	f.blockDomains.Domains = p["BlockedDomains"].Domains
	f.allowList.FirewallDomainList.DomainCount = aws.Int32(int32(len(f.allowDomains.Domains)))
	f.blockList.FirewallDomainList.DomainCount = aws.Int32(int32(len(f.blockDomains.Domains)))
	f.ListFirewallRulesValue.FirewallRules = nil
	for _, rule := range p["DnsRules"].FirewallRules {
		f.ListFirewallRulesValue.FirewallRules = append(f.ListFirewallRulesValue.FirewallRules, d.FirewallRule{FirewallRuleGroupId: aws.String(s.Network.FirewallGroup), FirewallDomainListId: aws.String(rule.FirewallDomainListId), Action: d.Action(rule.Action), Priority: aws.Int32(rule.Priority), BlockResponse: d.BlockResponse(rule.BlockResponse), FirewallDomainRedirectionAction: d.FirewallDomainRedirectionAction(rule.FirewallDomainRedirectionAction)})
	}
	f.ListFirewallRuleGroupAssociationsValue.FirewallRuleGroupAssociations[0].VpcId = aws.String(p["DnsAssociation"].VpcId)
	f.ListFirewallRuleGroupAssociationsValue.FirewallRuleGroupAssociations[0].MutationProtection = d.MutationProtectionStatus(p["DnsAssociation"].MutationProtection)
	g, err := NewNetworkGuard(c, s.Network, f, f, f, f)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.VerifyTask(t.Context(), c, task); err != nil {
		t.Fatal("packaged network disagrees with production admission", err, f.calls)
	}
}
