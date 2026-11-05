package ecsworker

import (
	"context"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func gatewayTargetEgress(values []types.IpPermission) bool {
	if len(values) != 1 {
		return false
	}
	rule := values[0]
	return (aws.ToString(rule.IpProtocol) == "tcp" || aws.ToString(rule.IpProtocol) == "6") && aws.ToInt32(rule.FromPort) == 1 && aws.ToInt32(rule.ToPort) == 65535 && len(rule.Ipv6Ranges)+len(rule.PrefixListIds)+len(rule.UserIdGroupPairs) == 0 && len(rule.IpRanges) == 1 && aws.ToString(rule.IpRanges[0].CidrIp) == "0.0.0.0/0"
}

func (g *NetworkGuard) checkGatewayGroups(ctx context.Context) error {
	s := g.settings.Gateway
	result, err := g.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{s.SecurityGroup, s.ExchangeGroup}}, oneEC2)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.SecurityGroups) != 2 {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, group := range result.SecurityGroups {
		id := aws.ToString(group.GroupId)
		if seen[id] || id != s.SecurityGroup && id != s.ExchangeGroup || aws.ToString(group.VpcId) != s.VPC || aws.ToString(group.OwnerId) != g.worker.Account {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
		if id == s.SecurityGroup {
			ingress, err := peerPermissions(group.IpPermissions, g.worker.Account, s.VPC, g.worker.SecurityGroups[0], g.settings.VPC, s.PeeringConnection)
			if err != nil || !slices.Equal(ingress, []string{"8044:sg:" + g.worker.SecurityGroups[0]}) || !gatewayTargetEgress(group.IpPermissionsEgress) {
				return hostedexecution.ErrOwnership
			}
		} else {
			ingress, err := permissions(group.IpPermissions, g.worker.Account, s.VPC)
			if err != nil || !slices.Equal(ingress, []string{"443:sg:" + s.SecurityGroup}) {
				return hostedexecution.ErrOwnership
			}
			egress, err := permissions(group.IpPermissionsEgress, g.worker.Account, s.VPC)
			if err != nil || len(egress) != 0 && !slices.Equal(egress, []string{"443:sg:" + s.SecurityGroup}) {
				return hostedexecution.ErrOwnership
			}
		}
	}
	return nil
}
