package ecsworker

import (
	"context"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type GatewayEC2 interface {
	EC2
	DescribeVpcPeeringConnections(context.Context, *ec2.DescribeVpcPeeringConnectionsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error)
	DescribeNatGateways(context.Context, *ec2.DescribeNatGatewaysInput, ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)
	DescribeInternetGateways(context.Context, *ec2.DescribeInternetGatewaysInput, ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
}

func (g *NetworkGuard) gatewayRole() *NetworkGuard {
	settings := g.settings.Gateway
	worker := canonical(g.worker)
	worker.Subnets, worker.SecurityGroups = slices.Clone(settings.Subnets), []string{settings.SecurityGroup}
	return &NetworkGuard{worker: worker, settings: NetworkSettings{VPC: settings.VPC}, ec2: g.ec2}
}

func (g *NetworkGuard) AdmitGateway(ctx context.Context, worker Config, task GatewayConfig) error {
	s := g.settings.Gateway
	if s == nil || validateGateway(worker, task) != nil || task.SecurityGroup != s.SecurityGroup || !equalSet(task.Subnets, s.Subnets) {
		return hostedexecution.ErrOwnership
	}
	return g.Admit(ctx, worker)
}

func (g *NetworkGuard) VerifyGatewayTask(ctx context.Context, worker Config, gateway GatewayConfig, task types.Task) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := g.AdmitGateway(ctx, worker, gateway); err != nil {
		return "", err
	}
	role := g.gatewayRole()
	return role.verifyInterface(ctx, role.worker, task)
}

var _ GatewayNetwork = (*NetworkGuard)(nil)
