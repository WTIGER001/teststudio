package ecsworker

import (
	"context"
	"net/netip"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func internetRoute(table types.RouteTable, local []string) (string, error) {
	found := map[string]bool{}
	gateway := ""
	for _, route := range table.Routes {
		route.Origin = ""
		expected := types.Route{State: types.RouteStateActive}
		cidr := aws.ToString(route.DestinationCidrBlock)
		if slices.Contains(local, cidr) {
			expected.DestinationCidrBlock, expected.GatewayId = aws.String(cidr), aws.String("local")
		} else if cidr == "0.0.0.0/0" {
			gateway = aws.ToString(route.GatewayId)
			if !strings.HasPrefix(gateway, "igw-") || !resourceName.MatchString(gateway) {
				return "", hostedexecution.ErrOwnership
			}
			expected.DestinationCidrBlock, expected.GatewayId = aws.String(cidr), aws.String(gateway)
		} else {
			return "", hostedexecution.ErrOwnership
		}
		if found[cidr] || fingerprint(document(route)) != fingerprint(document(expected)) {
			return "", hostedexecution.ErrOwnership
		}
		found[cidr] = true
	}
	if gateway == "" || len(found) != len(local)+1 {
		return "", hostedexecution.ErrOwnership
	}
	return gateway, nil
}

func (g *NetworkGuard) checkGatewayNAT(ctx context.Context, local []string, tables []types.RouteTable) error {
	s := g.settings.Gateway
	client := g.ec2.(GatewayEC2)
	result, err := client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{NatGatewayIds: slices.Clone(s.NATGateways)}, oneEC2)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.NatGateways) != len(s.NATGateways) {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	publicSubnets := []string{}
	for _, nat := range result.NatGateways {
		id, subnetValue := aws.ToString(nat.NatGatewayId), aws.ToString(nat.SubnetId)
		if seen[id] || !slices.Contains(s.NATGateways, id) || aws.ToString(nat.VpcId) != s.VPC || nat.State != types.NatGatewayStateAvailable || nat.ConnectivityType != types.ConnectivityTypePublic || !subnet.MatchString(subnetValue) || slices.Contains(s.Subnets, subnetValue) || len(nat.NatGatewayAddresses) == 0 || len(nat.AttachedAppliances) != 0 {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
		if !slices.Contains(publicSubnets, subnetValue) {
			publicSubnets = append(publicSubnets, subnetValue)
		}
		for _, address := range nat.NatGatewayAddresses {
			ip, err := netip.ParseAddr(aws.ToString(address.PublicIp))
			if err != nil || !ip.Is4() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || aws.ToString(address.AllocationId) == "" || aws.ToString(address.NetworkInterfaceId) == "" || address.Status != "" && address.Status != types.NatGatewayAddressStatusSucceeded {
				return hostedexecution.ErrOwnership
			}
		}
	}
	subnets, err := g.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: publicSubnets}, oneEC2)
	if err != nil || subnets == nil {
		return hostedexecution.ErrUnavailable
	}
	if subnets.NextToken != nil || len(subnets.Subnets) != len(publicSubnets) {
		return hostedexecution.ErrOwnership
	}
	public := map[string]types.Subnet{}
	for _, subnet := range subnets.Subnets {
		id := aws.ToString(subnet.SubnetId)
		if _, duplicate := public[id]; duplicate || !slices.Contains(publicSubnets, id) || aws.ToString(subnet.VpcId) != s.VPC || aws.ToString(subnet.OwnerId) != g.worker.Account || subnet.State != types.SubnetStateAvailable {
			return hostedexecution.ErrOwnership
		}
		public[id] = subnet
	}
	if !containedSubnets(public, local) {
		return hostedexecution.ErrOwnership
	}
	selected, err := selectedRouteTables(tables, publicSubnets, s.VPC, g.worker.Account)
	if err != nil {
		return err
	}
	internet := []string{}
	for _, table := range selected {
		id, err := internetRoute(table, local)
		if err != nil {
			return err
		}
		if !slices.Contains(internet, id) {
			internet = append(internet, id)
		}
	}
	gateways, err := client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{InternetGatewayIds: internet}, oneEC2)
	if err != nil || gateways == nil {
		return hostedexecution.ErrUnavailable
	}
	if gateways.NextToken != nil || len(gateways.InternetGateways) != len(internet) {
		return hostedexecution.ErrOwnership
	}
	seen = map[string]bool{}
	for _, gateway := range gateways.InternetGateways {
		id := aws.ToString(gateway.InternetGatewayId)
		if seen[id] || !slices.Contains(internet, id) || aws.ToString(gateway.OwnerId) != g.worker.Account || len(gateway.Attachments) != 1 || aws.ToString(gateway.Attachments[0].VpcId) != s.VPC || gateway.Attachments[0].State != types.AttachmentStatus("available") {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
	}
	return nil
}
