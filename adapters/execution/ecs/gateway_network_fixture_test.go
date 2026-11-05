package ecsworker

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type gatewayNetworkFixture struct {
	*networkFixture
	gateway    GatewayNetworkSettings
	vpc        e.Vpc
	subnets    []e.Subnet
	groups     []e.SecurityGroup
	routes     []e.RouteTable
	endpoint   e.VpcEndpoint
	eni        e.NetworkInterface
	peer       e.VpcPeeringConnection
	nat        e.NatGateway
	internet   e.InternetGateway
	requestIDs []string
}

func peerRule(account, group, vpc, peering string, port int32) e.IpPermission {
	value := sgRule(account, group, port)
	value.UserIdGroupPairs[0].VpcId = aws.String(vpc)
	value.UserIdGroupPairs[0].VpcPeeringConnectionId = aws.String(peering)
	value.UserIdGroupPairs[0].PeeringStatus = aws.String("active")
	return value
}

func gatewayNetwork(t *testing.T) (*NetworkGuard, *gatewayNetworkFixture, Config, GatewayConfig, types.Task) {
	t.Helper()
	previous, base, worker, task := network(t)
	_, gateway, _, _, _ := gatewayPlanFixture(t)
	s := GatewayNetworkSettings{VPC: "vpc-cccccccc", Subnets: []string{"subnet-cccccccc"}, SecurityGroup: "sg-ffffffff", PeeringConnection: "pcx-aaaaaaaa", ExchangeEndpoint: "vpce-66666666666666666", ExchangeGroup: "sg-11111111", NATGateways: []string{"nat-aaaaaaaa"}}
	gateway.Subnets, gateway.SecurityGroup = s.Subnets, s.SecurityGroup
	settings := previous.settings
	settings.Gateway = &s
	worker.NetworkProfile = ""
	var err error
	worker, err = settings.Bind(worker)
	if err != nil {
		t.Fatal(err)
	}
	f := &gatewayNetworkFixture{networkFixture: base, gateway: s}
	f.settings = settings
	f.vpc = e.Vpc{VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), State: "available", CidrBlock: aws.String("10.60.0.0/16"), DhcpOptionsId: aws.String("dopt-aaaaaaaa")}
	f.subnets = []e.Subnet{{SubnetId: aws.String(s.Subnets[0]), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), State: "available", CidrBlock: aws.String("10.60.1.0/24")}, {SubnetId: aws.String("subnet-dddddddd"), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), State: "available", CidrBlock: aws.String("10.60.10.0/24"), MapPublicIpOnLaunch: aws.Bool(true)}}
	f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress = append(f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress, peerRule(worker.Account, s.SecurityGroup, s.VPC, s.PeeringConnection, 8044))
	f.groups = []e.SecurityGroup{{GroupId: aws.String(s.SecurityGroup), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), IpPermissions: []e.IpPermission{peerRule(worker.Account, worker.SecurityGroups[0], settings.VPC, s.PeeringConnection, 8044)}, IpPermissionsEgress: []e.IpPermission{{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(1), ToPort: aws.Int32(65535), IpRanges: []e.IpRange{{CidrIp: aws.String("0.0.0.0/0")}}}}}, {GroupId: aws.String(s.ExchangeGroup), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), IpPermissions: []e.IpPermission{sgRule(worker.Account, s.SecurityGroup, 443)}}}
	f.DescribeRouteTablesValue.RouteTables[0].Routes = append(f.DescribeRouteTablesValue.RouteTables[0].Routes, e.Route{DestinationCidrBlock: aws.String("10.60.1.0/24"), VpcPeeringConnectionId: aws.String(s.PeeringConnection), State: "active"})
	f.routes = []e.RouteTable{{RouteTableId: aws.String("rtb-bbbbbbbb"), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), Associations: []e.RouteTableAssociation{{SubnetId: aws.String(s.Subnets[0])}}, Routes: []e.Route{{GatewayId: aws.String("local"), DestinationCidrBlock: aws.String("10.60.0.0/16"), State: "active"}, {DestinationCidrBlock: aws.String("10.40.1.0/24"), VpcPeeringConnectionId: aws.String(s.PeeringConnection), State: "active"}, {DestinationCidrBlock: aws.String("10.40.2.0/24"), VpcPeeringConnectionId: aws.String(s.PeeringConnection), State: "active"}, {DestinationCidrBlock: aws.String("0.0.0.0/0"), NatGatewayId: aws.String(s.NATGateways[0]), State: "active"}}}, {RouteTableId: aws.String("rtb-cccccccc"), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), Associations: []e.RouteTableAssociation{{SubnetId: f.subnets[1].SubnetId}}, Routes: []e.Route{{GatewayId: aws.String("local"), DestinationCidrBlock: aws.String("10.60.0.0/16"), State: "active"}, {DestinationCidrBlock: aws.String("0.0.0.0/0"), GatewayId: aws.String("igw-aaaaaaaa"), State: "active"}}}}
	f.peer = e.VpcPeeringConnection{VpcPeeringConnectionId: aws.String(s.PeeringConnection), Status: &e.VpcPeeringConnectionStateReason{Code: "active"}, RequesterVpcInfo: &e.VpcPeeringConnectionVpcInfo{VpcId: aws.String(settings.VPC), OwnerId: aws.String(worker.Account), Region: aws.String(worker.Region), CidrBlock: aws.String("10.40.0.0/16")}, AccepterVpcInfo: &e.VpcPeeringConnectionVpcInfo{VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), Region: aws.String(worker.Region), CidrBlock: f.vpc.CidrBlock}}
	f.nat = e.NatGateway{NatGatewayId: aws.String(s.NATGateways[0]), VpcId: aws.String(s.VPC), SubnetId: f.subnets[1].SubnetId, State: "available", ConnectivityType: "public", NatGatewayAddresses: []e.NatGatewayAddress{{AllocationId: aws.String("eipalloc-aaaaaaaa"), NetworkInterfaceId: aws.String("eni-dddddddd"), PublicIp: aws.String("203.0.113.2"), Status: "succeeded"}}}
	f.internet = e.InternetGateway{InternetGatewayId: aws.String("igw-aaaaaaaa"), OwnerId: aws.String(worker.Account), Attachments: []e.InternetGatewayAttachment{{VpcId: aws.String(s.VPC), State: "available"}}}
	f.endpoint = e.VpcEndpoint{VpcEndpointId: aws.String(s.ExchangeEndpoint), VpcId: aws.String(s.VPC), OwnerId: aws.String(worker.Account), State: e.StateAvailable, ServiceName: aws.String(settings.ExchangeService), VpcEndpointType: "Interface", PrivateDnsEnabled: aws.Bool(true), IpAddressType: "ipv4", Groups: []e.SecurityGroupIdentifier{{GroupId: aws.String(s.ExchangeGroup)}}, SubnetIds: s.Subnets, NetworkInterfaceIds: []string{"eni-eeeeeeee"}}
	f.eni = copyValue(f.workerENI.NetworkInterfaces[0])
	f.eni.NetworkInterfaceId, f.eni.VpcId, f.eni.SubnetId, f.eni.PrivateIpAddress, f.eni.Groups = aws.String("eni-ffffffff"), aws.String(s.VPC), aws.String(s.Subnets[0]), aws.String("10.60.1.21"), []e.GroupIdentifier{{GroupId: aws.String(s.SecurityGroup)}}
	for i := range task.Attachments[0].Details {
		field := &task.Attachments[0].Details[i]
		switch aws.ToString(field.Name) {
		case "networkInterfaceId":
			field.Value = f.eni.NetworkInterfaceId
		case "subnetId":
			field.Value = f.eni.SubnetId
		case "privateIPv4Address":
			field.Value = f.eni.PrivateIpAddress
		}
	}
	task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address = f.eni.PrivateIpAddress
	guard, err := NewNetworkGuard(worker, settings, f, f, f, f)
	if err != nil {
		t.Fatal(err)
	}
	return guard, f, worker, gateway, task
}

func (f *gatewayNetworkFixture) DescribeVpcs(ctx context.Context, input *ec2.DescribeVpcsInput, options ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	if len(input.VpcIds) == 1 && input.VpcIds[0] == f.gateway.VPC {
		if err := f.before(ctx, "GatewayVPC"); err != nil {
			return nil, err
		}
		return &ec2.DescribeVpcsOutput{Vpcs: []e.Vpc{copyValue(f.vpc)}}, nil
	}
	return f.networkFixture.DescribeVpcs(ctx, input, options...)
}

func (f *gatewayNetworkFixture) DescribeSubnets(ctx context.Context, input *ec2.DescribeSubnetsInput, options ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	if len(input.SubnetIds) == 1 && (input.SubnetIds[0] == f.gateway.Subnets[0] || input.SubnetIds[0] == "subnet-dddddddd") {
		if err := f.before(ctx, "GatewaySubnets"); err != nil {
			return nil, err
		}
		result := &ec2.DescribeSubnetsOutput{}
		for _, subnet := range f.subnets {
			if aws.ToString(subnet.SubnetId) == input.SubnetIds[0] {
				result.Subnets = append(result.Subnets, copyValue(subnet))
			}
		}
		return result, nil
	}
	return f.networkFixture.DescribeSubnets(ctx, input, options...)
}

func (f *gatewayNetworkFixture) DescribeRouteTables(ctx context.Context, input *ec2.DescribeRouteTablesInput, options ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	if len(input.Filters) == 1 && input.Filters[0].Values[0] == f.gateway.VPC {
		if err := f.before(ctx, "GatewayRoutes"); err != nil {
			return nil, err
		}
		return &ec2.DescribeRouteTablesOutput{RouteTables: copyValue(f.routes)}, nil
	}
	return f.networkFixture.DescribeRouteTables(ctx, input, options...)
}

func (f *gatewayNetworkFixture) DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, options ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	if len(input.GroupIds) == 2 && input.GroupIds[0] == f.gateway.SecurityGroup {
		if err := f.before(ctx, "GatewayGroups"); err != nil {
			return nil, err
		}
		return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: copyValue(f.groups)}, nil
	}
	return f.networkFixture.DescribeSecurityGroups(ctx, input, options...)
}

func (f *gatewayNetworkFixture) DescribeVpcEndpoints(ctx context.Context, input *ec2.DescribeVpcEndpointsInput, options ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	if len(input.VpcEndpointIds) == 1 && input.VpcEndpointIds[0] == f.gateway.ExchangeEndpoint {
		if err := f.before(ctx, "GatewayExchange"); err != nil {
			return nil, err
		}
		return &ec2.DescribeVpcEndpointsOutput{VpcEndpoints: []e.VpcEndpoint{copyValue(f.endpoint)}}, nil
	}
	return f.networkFixture.DescribeVpcEndpoints(ctx, input, options...)
}

func (f *gatewayNetworkFixture) DescribeNetworkInterfaces(ctx context.Context, input *ec2.DescribeNetworkInterfacesInput, options ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	if len(input.NetworkInterfaceIds) == 1 && input.NetworkInterfaceIds[0] == aws.ToString(f.eni.NetworkInterfaceId) {
		if err := f.before(ctx, "GatewayInterface"); err != nil {
			return nil, err
		}
		return &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []e.NetworkInterface{copyValue(f.eni)}}, nil
	}
	return f.networkFixture.DescribeNetworkInterfaces(ctx, input, options...)
}

func (f *gatewayNetworkFixture) DescribeVpcPeeringConnections(ctx context.Context, input *ec2.DescribeVpcPeeringConnectionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	if err := f.before(ctx, "GatewayPeering"); err != nil {
		return nil, err
	}
	f.requestIDs = append(f.requestIDs, input.VpcPeeringConnectionIds...)
	return &ec2.DescribeVpcPeeringConnectionsOutput{VpcPeeringConnections: []e.VpcPeeringConnection{copyValue(f.peer)}}, nil
}

func (f *gatewayNetworkFixture) DescribeNatGateways(ctx context.Context, input *ec2.DescribeNatGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	if err := f.before(ctx, "GatewayNAT"); err != nil {
		return nil, err
	}
	f.requestIDs = append(f.requestIDs, input.NatGatewayIds...)
	return &ec2.DescribeNatGatewaysOutput{NatGateways: []e.NatGateway{copyValue(f.nat)}}, nil
}

func (f *gatewayNetworkFixture) DescribeInternetGateways(ctx context.Context, input *ec2.DescribeInternetGatewaysInput, _ ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
	if err := f.before(ctx, "GatewayInternet"); err != nil {
		return nil, err
	}
	f.requestIDs = append(f.requestIDs, input.InternetGatewayIds...)
	return &ec2.DescribeInternetGatewaysOutput{InternetGateways: []e.InternetGateway{copyValue(f.internet)}}, nil
}
