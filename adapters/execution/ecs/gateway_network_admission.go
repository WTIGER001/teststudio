package ecsworker

import (
	"context"
	"net/netip"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *NetworkGuard) checkGatewayNetwork(ctx context.Context) error {
	role := g.gatewayRole()
	vpc, err := role.vpcIdentity(ctx)
	if err != nil {
		return err
	}
	subnets, err := role.subnets(ctx)
	if err != nil {
		return err
	}
	if err := g.checkGatewayPeering(ctx, vpc, subnets); err != nil {
		return err
	}
	if err := g.checkGatewayGroups(ctx); err != nil {
		return err
	}
	if err := g.checkGatewayRoutes(ctx, role, vpc); err != nil {
		return err
	}
	return g.checkGatewayExchange(ctx)
}

func peeringCIDRs(info *types.VpcPeeringConnectionVpcInfo, vpc, account, region string) ([]string, error) {
	if info == nil || aws.ToString(info.VpcId) != vpc || aws.ToString(info.OwnerId) != account || aws.ToString(info.Region) != region || len(info.Ipv6CidrBlockSet) != 0 {
		return nil, hostedexecution.ErrOwnership
	}
	values := []string{aws.ToString(info.CidrBlock)}
	for _, entry := range info.CidrBlockSet {
		cidr := aws.ToString(entry.CidrBlock)
		if !slices.Contains(values, cidr) {
			values = append(values, cidr)
		}
	}
	for _, cidr := range values {
		if _, err := privatePrefix(cidr); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func containedSubnets(subnets map[string]types.Subnet, cidrs []string) bool {
	for _, subnet := range subnets {
		prefix, err := privatePrefix(aws.ToString(subnet.CidrBlock))
		if err != nil {
			return false
		}
		found := false
		for _, cidr := range cidrs {
			parent, _ := netip.ParsePrefix(cidr)
			found = found || parent.Bits() <= prefix.Bits() && parent.Contains(prefix.Addr())
		}
		if !found {
			return false
		}
	}
	return true
}

func (g *NetworkGuard) checkGatewayPeering(ctx context.Context, vpc types.Vpc, gatewaySubnets map[string]types.Subnet) error {
	s := g.settings.Gateway
	result, err := g.ec2.(GatewayEC2).DescribeVpcPeeringConnections(ctx, &ec2.DescribeVpcPeeringConnectionsInput{VpcPeeringConnectionIds: []string{s.PeeringConnection}}, oneEC2)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.VpcPeeringConnections) != 1 {
		return hostedexecution.ErrOwnership
	}
	peer := result.VpcPeeringConnections[0]
	if aws.ToString(peer.VpcPeeringConnectionId) != s.PeeringConnection || peer.Status == nil || peer.Status.Code != types.VpcPeeringConnectionStateReasonCodeActive {
		return hostedexecution.ErrOwnership
	}
	workerInfo, gatewayInfo := peer.RequesterVpcInfo, peer.AccepterVpcInfo
	if workerInfo != nil && aws.ToString(workerInfo.VpcId) == s.VPC {
		workerInfo, gatewayInfo = gatewayInfo, workerInfo
	}
	workerCIDRs, err := peeringCIDRs(workerInfo, g.settings.VPC, g.worker.Account, g.worker.Region)
	if err != nil {
		return err
	}
	gatewayCIDRs, err := peeringCIDRs(gatewayInfo, s.VPC, g.worker.Account, g.worker.Region)
	if err != nil {
		return err
	}
	actual, err := vpcCIDRs(vpc)
	if err != nil || !equalSet(gatewayCIDRs, actual) || !containedSubnets(gatewaySubnets, gatewayCIDRs) {
		return hostedexecution.ErrOwnership
	}
	workerSubnets, err := g.subnets(ctx)
	if err != nil {
		return err
	}
	if !containedSubnets(workerSubnets, workerCIDRs) {
		return hostedexecution.ErrOwnership
	}
	for _, left := range workerCIDRs {
		prefix, _ := netip.ParsePrefix(left)
		for _, right := range gatewayCIDRs {
			other, _ := netip.ParsePrefix(right)
			if prefix.Overlaps(other) {
				return hostedexecution.ErrOwnership
			}
		}
	}
	return nil
}

func (g *NetworkGuard) checkGatewayExchange(ctx context.Context) error {
	s := g.settings.Gateway
	result, err := g.ec2.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{VpcEndpointIds: []string{s.ExchangeEndpoint}}, oneEC2)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.VpcEndpoints) != 1 {
		return hostedexecution.ErrOwnership
	}
	endpoint := result.VpcEndpoints[0]
	if aws.ToString(endpoint.VpcEndpointId) != s.ExchangeEndpoint || aws.ToString(endpoint.VpcId) != s.VPC || aws.ToString(endpoint.OwnerId) != g.worker.Account || endpoint.State != types.StateAvailable || aws.ToString(endpoint.ServiceName) != g.settings.ExchangeService || endpoint.VpcEndpointType != types.VpcEndpointTypeInterface || !aws.ToBool(endpoint.PrivateDnsEnabled) || endpoint.IpAddressType != types.IpAddressTypeIpv4 || len(endpoint.Ipv6Prefixes) != 0 || len(endpoint.Groups) != 1 || aws.ToString(endpoint.Groups[0].GroupId) != s.ExchangeGroup || !equalSet(endpoint.SubnetIds, s.Subnets) || len(endpoint.NetworkInterfaceIds) != len(s.Subnets) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
