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

func (g *NetworkGuard) subnets(ctx context.Context) (map[string]types.Subnet, error) {
	result, err := g.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: slices.Clone(g.worker.Subnets)}, oneEC2)
	if err != nil || result == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.Subnets) != len(g.worker.Subnets) {
		return nil, hostedexecution.ErrOwnership
	}
	values := map[string]types.Subnet{}
	for _, s := range result.Subnets {
		id := aws.ToString(s.SubnetId)
		if !slices.Contains(g.worker.Subnets, id) || aws.ToString(s.VpcId) != g.settings.VPC || aws.ToString(s.OwnerId) != g.worker.Account || s.State != types.SubnetStateAvailable || aws.ToBool(s.MapPublicIpOnLaunch) || aws.ToBool(s.MapCustomerOwnedIpOnLaunch) || aws.ToBool(s.AssignIpv6AddressOnCreation) || aws.ToBool(s.EnableDns64) || aws.ToBool(s.Ipv6Native) || len(s.Ipv6CidrBlockAssociationSet) != 0 {
			return nil, hostedexecution.ErrOwnership
		}
		if _, duplicate := values[id]; duplicate {
			return nil, hostedexecution.ErrOwnership
		}
		if cidr, err := netip.ParsePrefix(aws.ToString(s.CidrBlock)); err != nil || !cidr.Addr().Is4() {
			return nil, hostedexecution.ErrOwnership
		}
		values[id] = s
	}
	return values, nil
}

func (g *NetworkGuard) checkVPC(ctx context.Context) error {
	vpc, err := g.vpcIdentity(ctx)
	if err != nil {
		return err
	}
	subnets, err := g.subnets(ctx)
	if err != nil {
		return err
	}
	cidrs, err := vpcCIDRs(vpc)
	if err != nil || !containedSubnets(subnets, cidrs) {
		return hostedexecution.ErrOwnership
	}
	g.routeIDs, err = g.routes(ctx, vpc)
	return err
}

func (g *NetworkGuard) vpcIdentity(ctx context.Context) (types.Vpc, error) {
	result, err := g.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: []string{g.settings.VPC}}, oneEC2)
	if err != nil || result == nil {
		return types.Vpc{}, hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.Vpcs) != 1 {
		return types.Vpc{}, hostedexecution.ErrOwnership
	}
	vpc := result.Vpcs[0]
	if aws.ToString(vpc.VpcId) != g.settings.VPC || aws.ToString(vpc.OwnerId) != g.worker.Account || vpc.State != types.VpcStateAvailable || aws.ToBool(vpc.IsDefault) || len(vpc.Ipv6CidrBlockAssociationSet) != 0 {
		return types.Vpc{}, hostedexecution.ErrOwnership
	}
	for _, attribute := range []types.VpcAttributeName{types.VpcAttributeNameEnableDnsSupport, types.VpcAttributeNameEnableDnsHostnames} {
		value, err := g.ec2.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{VpcId: aws.String(g.settings.VPC), Attribute: attribute}, oneEC2)
		if err != nil || value == nil {
			return types.Vpc{}, hostedexecution.ErrUnavailable
		}
		if aws.ToString(value.VpcId) != g.settings.VPC {
			return types.Vpc{}, hostedexecution.ErrOwnership
		}
		if attribute == types.VpcAttributeNameEnableDnsSupport && (value.EnableDnsSupport == nil || !aws.ToBool(value.EnableDnsSupport.Value)) || attribute == types.VpcAttributeNameEnableDnsHostnames && (value.EnableDnsHostnames == nil || !aws.ToBool(value.EnableDnsHostnames.Value)) {
			return types.Vpc{}, hostedexecution.ErrOwnership
		}
	}
	dhcp, err := g.ec2.DescribeDhcpOptions(ctx, &ec2.DescribeDhcpOptionsInput{DhcpOptionsIds: []string{aws.ToString(vpc.DhcpOptionsId)}}, oneEC2)
	if err != nil || dhcp == nil {
		return types.Vpc{}, hostedexecution.ErrUnavailable
	}
	if len(dhcp.DhcpOptions) != 1 || aws.ToString(dhcp.DhcpOptions[0].DhcpOptionsId) != aws.ToString(vpc.DhcpOptionsId) || aws.ToString(dhcp.DhcpOptions[0].OwnerId) != g.worker.Account {
		return types.Vpc{}, hostedexecution.ErrOwnership
	}
	servers := 0
	for _, option := range dhcp.DhcpOptions[0].DhcpConfigurations {
		if aws.ToString(option.Key) == "domain-name-servers" {
			servers++
			if len(option.Values) != 1 || aws.ToString(option.Values[0].Value) != "AmazonProvidedDNS" {
				return types.Vpc{}, hostedexecution.ErrOwnership
			}
		}
	}
	if servers != 1 {
		return types.Vpc{}, hostedexecution.ErrOwnership
	}
	return vpc, nil
}

func (g *NetworkGuard) routeTables(ctx context.Context) ([]types.RouteTable, error) {
	values := []types.RouteTable{}
	next := ""
	seen := map[string]bool{}
	for page := 0; page < 4; page++ {
		result, err := g.ec2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{g.settings.VPC}}}, MaxResults: aws.Int32(100), NextToken: optional(next)}, oneEC2)
		if err != nil || result == nil || len(result.RouteTables) > 100 {
			return nil, hostedexecution.ErrUnavailable
		}
		values = append(values, result.RouteTables...)
		next = aws.ToString(result.NextToken)
		if next == "" {
			return values, nil
		}
		if seen[next] {
			return nil, hostedexecution.ErrUnavailable
		}
		seen[next] = true
	}
	return nil, hostedexecution.ErrUnavailable
}

func (g *NetworkGuard) routes(ctx context.Context, vpc types.Vpc) ([]string, error) {
	tables, err := g.routeTables(ctx)
	if err != nil {
		return nil, err
	}
	selected, err := selectedRouteTables(tables, g.worker.Subnets, g.settings.VPC, g.worker.Account)
	if err != nil {
		return nil, err
	}
	local, err := vpcCIDRs(vpc)
	if err != nil {
		return nil, err
	}
	peer := []string{}
	if g.settings.Gateway != nil {
		subnets, err := g.gatewayRole().subnets(ctx)
		if err != nil {
			return nil, err
		}
		peer, err = subnetCIDRs(subnets)
		if err != nil {
			return nil, err
		}
	}
	ids := []string{}
	for _, table := range selected {
		ids = append(ids, aws.ToString(table.RouteTableId))
		found := map[string]bool{}
		for _, route := range table.Routes {
			route.Origin = ""
			expected := types.Route{State: types.RouteStateActive}
			key := "s3"
			cidr := aws.ToString(route.DestinationCidrBlock)
			if slices.Contains(local, cidr) {
				expected.DestinationCidrBlock, expected.GatewayId = aws.String(cidr), aws.String("local")
				key = cidr
			} else if slices.Contains(peer, cidr) {
				expected.DestinationCidrBlock, expected.VpcPeeringConnectionId = aws.String(cidr), aws.String(g.settings.Gateway.PeeringConnection)
				key = cidr
			} else {
				expected.DestinationPrefixListId, expected.GatewayId = aws.String(g.settings.S3PrefixList), aws.String(g.settings.S3Endpoint)
			}
			if fingerprint(document(route)) != fingerprint(document(expected)) || found[key] {
				return nil, hostedexecution.ErrOwnership
			}
			found[key] = true
		}
		if len(found) != len(local)+len(peer)+1 || !found["s3"] {
			return nil, hostedexecution.ErrOwnership
		}
	}
	slices.Sort(ids)
	return ids, nil
}
