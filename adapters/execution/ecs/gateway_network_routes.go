package ecsworker

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func gatewayRoutes(table types.RouteTable, local, workers []string, peering string, natIDs []string) (string, error) {
	found := map[string]bool{}
	nat := ""
	for _, route := range table.Routes {
		route.Origin = ""
		expected := types.Route{State: types.RouteStateActive}
		cidr := aws.ToString(route.DestinationCidrBlock)
		switch {
		case slices.Contains(local, cidr):
			expected.DestinationCidrBlock, expected.GatewayId = aws.String(cidr), aws.String("local")
		case slices.Contains(workers, cidr):
			expected.DestinationCidrBlock, expected.VpcPeeringConnectionId = aws.String(cidr), aws.String(peering)
		case cidr == "0.0.0.0/0":
			nat = aws.ToString(route.NatGatewayId)
			if !slices.Contains(natIDs, nat) {
				return "", hostedexecution.ErrOwnership
			}
			expected.DestinationCidrBlock, expected.NatGatewayId = aws.String(cidr), aws.String(nat)
		default:
			return "", hostedexecution.ErrOwnership
		}
		if fingerprint(document(route)) != fingerprint(document(expected)) || found[cidr] {
			return "", hostedexecution.ErrOwnership
		}
		found[cidr] = true
	}
	if nat == "" || len(found) != len(local)+len(workers)+1 {
		return "", hostedexecution.ErrOwnership
	}
	return nat, nil
}

func (g *NetworkGuard) checkGatewayRoutes(ctx context.Context, role *NetworkGuard, vpc types.Vpc) error {
	tables, err := role.routeTables(ctx)
	if err != nil {
		return err
	}
	selected, err := selectedRouteTables(tables, role.worker.Subnets, role.settings.VPC, g.worker.Account)
	if err != nil {
		return err
	}
	local, err := vpcCIDRs(vpc)
	if err != nil {
		return err
	}
	workerSubnets, err := g.subnets(ctx)
	if err != nil {
		return err
	}
	workers, err := subnetCIDRs(workerSubnets)
	if err != nil {
		return err
	}
	used := []string{}
	for _, table := range selected {
		nat, err := gatewayRoutes(table, local, workers, g.settings.Gateway.PeeringConnection, g.settings.Gateway.NATGateways)
		if err != nil {
			return err
		}
		if !slices.Contains(used, nat) {
			used = append(used, nat)
		}
	}
	if !equalSet(used, g.settings.Gateway.NATGateways) {
		return hostedexecution.ErrOwnership
	}
	return g.checkGatewayNAT(ctx, local, tables)
}
