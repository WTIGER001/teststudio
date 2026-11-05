package ecsworker

import (
	"net/netip"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func selectedRouteTables(tables []types.RouteTable, subnets []string, vpc, account string) ([]types.RouteTable, error) {
	byID := map[string]types.RouteTable{}
	explicit := map[string]string{}
	main := ""
	for _, table := range tables {
		id := aws.ToString(table.RouteTableId)
		if id == "" || aws.ToString(table.VpcId) != vpc || aws.ToString(table.OwnerId) != account {
			return nil, hostedexecution.ErrOwnership
		}
		if _, duplicate := byID[id]; duplicate {
			return nil, hostedexecution.ErrOwnership
		}
		byID[id] = table
		for _, a := range table.Associations {
			if a.AssociationState != nil && a.AssociationState.State != types.RouteTableAssociationStateCodeAssociated {
				return nil, hostedexecution.ErrUnavailable
			}
			if aws.ToBool(a.Main) {
				if main != "" {
					return nil, hostedexecution.ErrOwnership
				}
				main = id
			}
			if subnet := aws.ToString(a.SubnetId); slices.Contains(subnets, subnet) {
				if explicit[subnet] != "" {
					return nil, hostedexecution.ErrOwnership
				}
				explicit[subnet] = id
			}
		}
	}
	selected := []types.RouteTable{}
	seen := map[string]bool{}
	for _, subnet := range subnets {
		id := explicit[subnet]
		if id == "" {
			id = main
		}
		if id == "" {
			return nil, hostedexecution.ErrOwnership
		}
		if !seen[id] {
			selected = append(selected, byID[id])
			seen[id] = true
		}
	}
	return selected, nil
}

func privatePrefix(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil || !prefix.Addr().Is4() || !prefix.Addr().IsPrivate() || prefix != prefix.Masked() {
		return netip.Prefix{}, hostedexecution.ErrOwnership
	}
	return prefix, nil
}

func vpcCIDRs(vpc types.Vpc) ([]string, error) {
	values := []string{aws.ToString(vpc.CidrBlock)}
	for _, association := range vpc.CidrBlockAssociationSet {
		if association.CidrBlockState == nil || association.CidrBlockState.State != types.VpcCidrBlockStateCodeAssociated {
			return nil, hostedexecution.ErrOwnership
		}
		cidr := aws.ToString(association.CidrBlock)
		if !slices.Contains(values, cidr) {
			values = append(values, cidr)
		}
	}
	for _, cidr := range values {
		if _, err := privatePrefix(cidr); err != nil {
			return nil, err
		}
	}
	slices.Sort(values)
	return values, nil
}

func subnetCIDRs(subnets map[string]types.Subnet) ([]string, error) {
	values := []string{}
	for _, subnet := range subnets {
		cidr := aws.ToString(subnet.CidrBlock)
		prefix, err := privatePrefix(cidr)
		if err != nil {
			return nil, err
		}
		for _, other := range values {
			previous, _ := netip.ParsePrefix(other)
			if prefix.Overlaps(previous) {
				return nil, hostedexecution.ErrOwnership
			}
		}
		values = append(values, cidr)
	}
	slices.Sort(values)
	return values, nil
}
