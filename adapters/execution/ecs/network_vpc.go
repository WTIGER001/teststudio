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
	if len(result.Subnets) != len(g.worker.Subnets) {
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
	result, err := g.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: []string{g.settings.VPC}}, oneEC2)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(result.Vpcs) != 1 {
		return hostedexecution.ErrOwnership
	}
	vpc := result.Vpcs[0]
	if aws.ToString(vpc.VpcId) != g.settings.VPC || aws.ToString(vpc.OwnerId) != g.worker.Account || vpc.State != types.VpcStateAvailable || aws.ToBool(vpc.IsDefault) || len(vpc.Ipv6CidrBlockAssociationSet) != 0 {
		return hostedexecution.ErrOwnership
	}
	for _, attribute := range []types.VpcAttributeName{types.VpcAttributeNameEnableDnsSupport, types.VpcAttributeNameEnableDnsHostnames} {
		value, err := g.ec2.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{VpcId: aws.String(g.settings.VPC), Attribute: attribute}, oneEC2)
		if err != nil || value == nil {
			return hostedexecution.ErrUnavailable
		}
		if aws.ToString(value.VpcId) != g.settings.VPC {
			return hostedexecution.ErrOwnership
		}
		if attribute == types.VpcAttributeNameEnableDnsSupport && (value.EnableDnsSupport == nil || !aws.ToBool(value.EnableDnsSupport.Value)) || attribute == types.VpcAttributeNameEnableDnsHostnames && (value.EnableDnsHostnames == nil || !aws.ToBool(value.EnableDnsHostnames.Value)) {
			return hostedexecution.ErrOwnership
		}
	}
	dhcp, err := g.ec2.DescribeDhcpOptions(ctx, &ec2.DescribeDhcpOptionsInput{DhcpOptionsIds: []string{aws.ToString(vpc.DhcpOptionsId)}}, oneEC2)
	if err != nil || dhcp == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(dhcp.DhcpOptions) != 1 || aws.ToString(dhcp.DhcpOptions[0].DhcpOptionsId) != aws.ToString(vpc.DhcpOptionsId) || aws.ToString(dhcp.DhcpOptions[0].OwnerId) != g.worker.Account {
		return hostedexecution.ErrOwnership
	}
	servers := 0
	for _, option := range dhcp.DhcpOptions[0].DhcpConfigurations {
		if aws.ToString(option.Key) == "domain-name-servers" {
			servers++
			if len(option.Values) != 1 || aws.ToString(option.Values[0].Value) != "AmazonProvidedDNS" {
				return hostedexecution.ErrOwnership
			}
		}
	}
	if servers != 1 {
		return hostedexecution.ErrOwnership
	}
	if _, err := g.subnets(ctx); err != nil {
		return err
	}
	g.routeIDs, err = g.routes(ctx, vpc)
	return err
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
	byID := map[string]types.RouteTable{}
	explicit := map[string]string{}
	main := ""
	for _, table := range tables {
		id := aws.ToString(table.RouteTableId)
		if id == "" || aws.ToString(table.VpcId) != g.settings.VPC || aws.ToString(table.OwnerId) != g.worker.Account {
			return nil, hostedexecution.ErrOwnership
		}
		if _, duplicate := byID[id]; duplicate {
			return nil, hostedexecution.ErrOwnership
		}
		byID[id] = table
		for _, a := range table.Associations {
			if aws.ToBool(a.Main) {
				if main != "" {
					return nil, hostedexecution.ErrOwnership
				}
				main = id
			}
			if subnet := aws.ToString(a.SubnetId); slices.Contains(g.worker.Subnets, subnet) {
				if explicit[subnet] != "" {
					return nil, hostedexecution.ErrOwnership
				}
				explicit[subnet] = id
			}
		}
	}
	local := map[string]bool{aws.ToString(vpc.CidrBlock): true}
	for _, a := range vpc.CidrBlockAssociationSet {
		if a.CidrBlockState == nil || a.CidrBlockState.State != types.VpcCidrBlockStateCodeAssociated {
			return nil, hostedexecution.ErrOwnership
		}
		local[aws.ToString(a.CidrBlock)] = true
	}
	selected := []string{}
	for _, subnet := range g.worker.Subnets {
		id := explicit[subnet]
		if id == "" {
			id = main
		}
		if id == "" {
			return nil, hostedexecution.ErrOwnership
		}
		if slices.Contains(selected, id) {
			continue
		}
		selected = append(selected, id)
		found := map[string]bool{}
		for _, r := range byID[id].Routes {
			r.Origin = ""
			expected := types.Route{State: types.RouteStateActive}
			key := "s3"
			if cidr := aws.ToString(r.DestinationCidrBlock); local[cidr] {
				expected.DestinationCidrBlock = aws.String(cidr)
				expected.GatewayId = aws.String("local")
				key = cidr
			} else {
				expected.DestinationPrefixListId = aws.String(g.settings.S3PrefixList)
				expected.GatewayId = aws.String(g.settings.S3Endpoint)
			}
			if fingerprint(document(r)) != fingerprint(document(expected)) || found[key] {
				return nil, hostedexecution.ErrOwnership
			}
			found[key] = true
		}
		if len(found) != len(local)+1 || !found["s3"] {
			return nil, hostedexecution.ErrOwnership
		}
	}
	slices.Sort(selected)
	return selected, nil
}
