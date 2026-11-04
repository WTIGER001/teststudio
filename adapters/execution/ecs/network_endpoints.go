package ecsworker

import (
	"context"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func equalSet(left, right []string) bool {
	left = slices.Clone(left)
	right = slices.Clone(right)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func samePolicy(raw *string, expected any) bool {
	if raw == nil || len(*raw) > 32<<10 {
		return false
	}
	var value map[string]any
	if definitions.DecodeJSON([]byte(*raw), &value, 32<<10, 12, 2048) != nil {
		return false
	}
	return fingerprint(value) == fingerprint(expected)
}

func (g *NetworkGuard) checkEndpoints(ctx context.Context) error {
	s, c := g.settings, g.worker
	ids := []string{s.ExchangeEndpoint, s.APIEndpoint, s.DKREndpoint, s.S3Endpoint}
	result, err := g.ec2.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{VpcEndpointIds: ids}, oneEC2)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(result.VpcEndpoints) != len(ids) || result.NextToken != nil {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, e := range result.VpcEndpoints {
		id := aws.ToString(e.VpcEndpointId)
		if !slices.Contains(ids, id) || seen[id] || aws.ToString(e.VpcId) != s.VPC || aws.ToString(e.OwnerId) != c.Account || e.State != types.StateAvailable || len(e.Ipv6Prefixes) != 0 {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
		service := "com.amazonaws." + c.Region + "."
		if id == s.S3Endpoint {
			if e.IpAddressType != types.IpAddressTypeIpv4 || e.VpcEndpointType != types.VpcEndpointTypeGateway || aws.ToString(e.ServiceName) != service+"s3" || !equalSet(e.RouteTableIds, g.routeIDs) || !samePolicy(e.PolicyDocument, s.S3Policy(c)) {
				return hostedexecution.ErrOwnership
			}
			continue
		}
		group := s.RegistryGroup
		switch id {
		case s.APIEndpoint:
			service += "ecr.api"
		case s.DKREndpoint:
			service += "ecr.dkr"
		case s.ExchangeEndpoint:
			group = s.ExchangeGroup
			service = s.ExchangeService
		}
		if e.VpcEndpointType != types.VpcEndpointTypeInterface || aws.ToString(e.ServiceName) != service || !aws.ToBool(e.PrivateDnsEnabled) || e.IpAddressType != types.IpAddressTypeIpv4 || len(e.Groups) != 1 || aws.ToString(e.Groups[0].GroupId) != group || len(e.NetworkInterfaceIds) != len(c.Subnets) || !equalSet(e.SubnetIds, c.Subnets) {
			return hostedexecution.ErrOwnership
		}
		for _, id := range e.SubnetIds {
			if !slices.Contains(c.Subnets, id) {
				return hostedexecution.ErrOwnership
			}
		}
		if id != s.ExchangeEndpoint && !samePolicy(e.PolicyDocument, s.ECRPolicy(c)) {
			return hostedexecution.ErrOwnership
		}
	}
	prefix, err := g.ec2.DescribePrefixLists(ctx, &ec2.DescribePrefixListsInput{PrefixListIds: []string{s.S3PrefixList}}, oneEC2)
	if err != nil || prefix == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(prefix.PrefixLists) != 1 || aws.ToString(prefix.PrefixLists[0].PrefixListId) != s.S3PrefixList || aws.ToString(prefix.PrefixLists[0].PrefixListName) != "com.amazonaws."+c.Region+".s3" {
		return hostedexecution.ErrOwnership
	}
	serviceID := s.ExchangeService[strings.LastIndex(s.ExchangeService, ".")+1:]
	config, err := g.ec2.DescribeVpcEndpointServiceConfigurations(ctx, &ec2.DescribeVpcEndpointServiceConfigurationsInput{ServiceIds: []string{serviceID}}, oneEC2)
	if err != nil || config == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(config.ServiceConfigurations) != 1 || config.NextToken != nil {
		return hostedexecution.ErrOwnership
	}
	service := config.ServiceConfigurations[0]
	if aws.ToString(service.ServiceId) != serviceID || aws.ToString(service.ServiceName) != s.ExchangeService || service.ServiceState != types.ServiceStateAvailable || !aws.ToBool(service.AcceptanceRequired) || aws.ToBool(service.ManagesVpcEndpoints) || aws.ToBool(service.RemoteAccessEnabled) || !equalSet(service.NetworkLoadBalancerArns, []string{s.LoadBalancer}) || len(service.GatewayLoadBalancerArns) != 0 || len(service.ServiceType) != 1 || service.ServiceType[0].ServiceType != types.ServiceTypeInterface || !equalSet(connectivity(service.SupportedIpAddressTypes), []string{"ipv4"}) || service.PrivateDnsNameConfiguration == nil || service.PrivateDnsNameConfiguration.State != types.DnsNameStateVerified {
		return hostedexecution.ErrOwnership
	}
	if aws.ToString(service.PrivateDnsName) != s.Domains(c)[0] {
		return hostedexecution.ErrOwnership
	}
	permissions, err := g.ec2.DescribeVpcEndpointServicePermissions(ctx, &ec2.DescribeVpcEndpointServicePermissionsInput{ServiceId: aws.String(serviceID)}, oneEC2)
	if err != nil || permissions == nil {
		return hostedexecution.ErrUnavailable
	}
	if permissions.NextToken != nil || len(permissions.AllowedPrincipals) != 1 || aws.ToString(permissions.AllowedPrincipals[0].Principal) != "arn:aws:iam::"+c.Account+":root" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func connectivity(values []types.ServiceConnectivityType) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}
