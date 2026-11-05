package ecsworker

import (
	"context"
	"net/netip"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *NetworkGuard) VerifyTask(ctx context.Context, c Config, task ecstypes.Task) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := g.Admit(ctx, c); err != nil {
		return err
	}
	_, err := g.verifyInterface(ctx, c, task)
	return err
}

func (g *NetworkGuard) verifyInterface(ctx context.Context, c Config, task ecstypes.Task) (string, error) {
	if len(task.Attachments) == 0 {
		return "", hostedexecution.ErrUnavailable
	}
	if len(task.Attachments) != 1 || aws.ToString(task.Attachments[0].Type) != "ElasticNetworkInterface" {
		return "", hostedexecution.ErrOwnership
	}
	a := task.Attachments[0]
	if aws.ToString(a.Status) != "ATTACHED" {
		return "", hostedexecution.ErrUnavailable
	}
	fields := map[string]string{}
	for _, field := range a.Details {
		name := aws.ToString(field.Name)
		if _, duplicate := fields[name]; duplicate {
			return "", hostedexecution.ErrOwnership
		}
		fields[name] = aws.ToString(field.Value)
	}
	if fields["networkInterfaceId"] == "" || fields["ipv6Address"] != "" || fields["subnetId"] == "" || fields["privateIPv4Address"] == "" {
		return "", hostedexecution.ErrOwnership
	}
	result, err := g.ec2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: []string{fields["networkInterfaceId"]}}, oneEC2)
	if err != nil || result == nil {
		return "", hostedexecution.ErrUnavailable
	}
	if result.NextToken != nil || len(result.NetworkInterfaces) != 1 {
		return "", hostedexecution.ErrOwnership
	}
	eni := result.NetworkInterfaces[0]
	if !aws.ToBool(eni.SourceDestCheck) || len(eni.AssociatedSubnets) != 0 || aws.ToBool(eni.Ipv6Native) {
		return "", hostedexecution.ErrOwnership
	}
	if aws.ToString(eni.NetworkInterfaceId) != fields["networkInterfaceId"] || aws.ToString(eni.OwnerId) != c.Account || aws.ToString(eni.VpcId) != g.settings.VPC || aws.ToString(eni.SubnetId) != fields["subnetId"] || aws.ToString(eni.PrivateIpAddress) != fields["privateIPv4Address"] || eni.InterfaceType != types.NetworkInterfaceTypeInterface || eni.Status != types.NetworkInterfaceStatusInUse || eni.Association != nil || len(eni.Ipv6Addresses)+len(eni.Ipv6Prefixes)+len(eni.Ipv4Prefixes) != 0 || aws.ToString(eni.Ipv6Address) != "" || aws.ToString(eni.PublicDnsName) != "" || len(eni.Groups) != 1 || aws.ToString(eni.Groups[0].GroupId) != c.SecurityGroups[0] {
		return "", hostedexecution.ErrOwnership
	}
	for _, ip := range eni.PrivateIpAddresses {
		if ip.Association != nil || aws.ToString(ip.PrivateIpAddress) != fields["privateIPv4Address"] {
			return "", hostedexecution.ErrOwnership
		}
	}
	subnets, err := g.subnets(ctx)
	if err != nil {
		return "", err
	}
	subnet, found := subnets[fields["subnetId"]]
	if !found {
		return "", hostedexecution.ErrOwnership
	}
	cidr, err := netip.ParsePrefix(aws.ToString(subnet.CidrBlock))
	if err != nil {
		return "", hostedexecution.ErrOwnership
	}
	ip, err := netip.ParseAddr(fields["privateIPv4Address"])
	if err != nil || !ip.Is4() || !cidr.Contains(ip) {
		return "", hostedexecution.ErrOwnership
	}
	if len(task.Containers) != 1 || len(task.Containers[0].NetworkInterfaces) != 1 {
		return "", hostedexecution.ErrOwnership
	}
	if value := task.Containers[0].NetworkInterfaces[0]; aws.ToString(value.AttachmentId) != aws.ToString(a.Id) || aws.ToString(value.PrivateIpv4Address) != fields["privateIPv4Address"] || aws.ToString(value.Ipv6Address) != "" {
		return "", hostedexecution.ErrOwnership
	}
	return fields["privateIPv4Address"], nil
}
