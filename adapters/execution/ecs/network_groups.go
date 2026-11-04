package ecsworker

import (
	"context"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func permissions(values []types.IpPermission, account, vpc string) ([]string, error) {
	result := []string{}
	for _, p := range values {
		if aws.ToString(p.IpProtocol) != "tcp" && aws.ToString(p.IpProtocol) != "6" || p.FromPort == nil || p.ToPort == nil || *p.FromPort != *p.ToPort || len(p.IpRanges)+len(p.Ipv6Ranges) != 0 {
			return nil, hostedexecution.ErrOwnership
		}
		if len(p.UserIdGroupPairs)+len(p.PrefixListIds) == 0 {
			return nil, hostedexecution.ErrOwnership
		}
		for _, group := range p.UserIdGroupPairs {
			if aws.ToString(group.UserId) != account || aws.ToString(group.VpcPeeringConnectionId) != "" || aws.ToString(group.PeeringStatus) != "" || aws.ToString(group.VpcId) != "" && aws.ToString(group.VpcId) != vpc {
				return nil, hostedexecution.ErrOwnership
			}
			result = append(result, fmt.Sprintf("%d:sg:%s", *p.FromPort, aws.ToString(group.GroupId)))
		}
		for _, prefix := range p.PrefixListIds {
			result = append(result, fmt.Sprintf("%d:pl:%s", *p.FromPort, aws.ToString(prefix.PrefixListId)))
		}
	}
	slices.Sort(result)
	for i := 1; i < len(result); i++ {
		if result[i] == result[i-1] {
			return nil, hostedexecution.ErrOwnership
		}
	}
	return result, nil
}

func (g *NetworkGuard) checkGroups(ctx context.Context) error {
	s := g.settings
	c := g.worker
	ids := []string{c.SecurityGroups[0], s.ExchangeGroup, s.RegistryGroup, s.LoadBalancerGroup, s.ControllerGroup}
	values, err := g.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: ids}, oneEC2)
	if err != nil || values == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(values.SecurityGroups) != len(ids) {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, group := range values.SecurityGroups {
		id := aws.ToString(group.GroupId)
		vpc := s.VPC
		if id == s.LoadBalancerGroup || id == s.ControllerGroup {
			vpc = s.ControllerVPC
		}
		if !slices.Contains(ids, id) || seen[id] || aws.ToString(group.OwnerId) != c.Account || aws.ToString(group.VpcId) != vpc {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
		if id == s.ControllerGroup {
			port := []types.IpPermission{}
			for _, rule := range group.IpPermissions {
				if aws.ToString(rule.IpProtocol) == "-1" || (aws.ToString(rule.IpProtocol) == "tcp" || aws.ToString(rule.IpProtocol) == "6") && rule.FromPort != nil && rule.ToPort != nil && *rule.FromPort <= 8042 && *rule.ToPort >= 8042 {
					port = append(port, rule)
				}
			}
			actual, err := permissions(port, c.Account, vpc)
			if err != nil || !slices.Equal(actual, []string{"8042:sg:" + s.LoadBalancerGroup}) {
				return hostedexecution.ErrOwnership
			}
			continue
		}
		ingress, err := permissions(group.IpPermissions, c.Account, vpc)
		if err != nil {
			return err
		}
		egress, err := permissions(group.IpPermissionsEgress, c.Account, vpc)
		if err != nil {
			return err
		}
		wantIn, wantOut := []string{}, []string{}
		switch id {
		case c.SecurityGroups[0]:
			wantOut = []string{"443:sg:" + s.ExchangeGroup, "443:sg:" + s.RegistryGroup, "443:pl:" + s.S3PrefixList}
		case s.ExchangeGroup, s.RegistryGroup:
			wantIn = []string{"443:sg:" + c.SecurityGroups[0]}
			if len(egress) > 0 {
				wantOut = []string{"443:sg:" + c.SecurityGroups[0]}
			}
		case s.LoadBalancerGroup:
			wantOut = []string{"8042:sg:" + s.ControllerGroup}
		}
		slices.Sort(wantIn)
		slices.Sort(wantOut)
		if !slices.Equal(ingress, wantIn) || !slices.Equal(egress, wantOut) {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}
