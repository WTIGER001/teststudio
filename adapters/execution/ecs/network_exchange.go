package ecsworker

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *NetworkGuard) checkExchange(ctx context.Context) error {
	s := g.settings
	result, err := g.elb.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{LoadBalancerArns: []string{s.LoadBalancer}}, oneELB)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	if len(result.LoadBalancers) != 1 || result.NextMarker != nil {
		return hostedexecution.ErrOwnership
	}
	lb := result.LoadBalancers[0]
	if aws.ToString(lb.LoadBalancerArn) != s.LoadBalancer || aws.ToString(lb.VpcId) != s.ControllerVPC || lb.Type != types.LoadBalancerTypeEnumNetwork || lb.Scheme != types.LoadBalancerSchemeEnumInternal || lb.IpAddressType != types.IpAddressTypeIpv4 || lb.State == nil || lb.State.Code != types.LoadBalancerStateEnumActive || !equalSet(lb.SecurityGroups, []string{s.LoadBalancerGroup}) || aws.ToString(lb.EnforceSecurityGroupInboundRulesOnPrivateLinkTraffic) != "off" {
		return hostedexecution.ErrOwnership
	}
	listeners, err := g.elb.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: aws.String(s.LoadBalancer)}, oneELB)
	if err != nil || listeners == nil {
		return hostedexecution.ErrUnavailable
	}
	if listeners.NextMarker != nil || len(listeners.Listeners) != 1 {
		return hostedexecution.ErrOwnership
	}
	l := listeners.Listeners[0]
	if aws.ToString(l.LoadBalancerArn) != s.LoadBalancer || aws.ToInt32(l.Port) != 443 || l.Protocol != types.ProtocolEnumTls || aws.ToString(l.SslPolicy) != "ELBSecurityPolicy-TLS13-1-2-2021-06" || len(l.Certificates) != 1 || !strings.HasPrefix(aws.ToString(l.Certificates[0].CertificateArn), "arn:aws:acm:"+g.worker.Region+":"+g.worker.Account+":certificate/") || len(l.DefaultActions) != 1 || l.DefaultActions[0].Type != types.ActionTypeEnumForward || aws.ToString(l.DefaultActions[0].TargetGroupArn) != s.TargetGroup || l.DefaultActions[0].ForwardConfig != nil {
		return hostedexecution.ErrOwnership
	}
	groups, err := g.elb.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{TargetGroupArns: []string{s.TargetGroup}}, oneELB)
	if err != nil || groups == nil {
		return hostedexecution.ErrUnavailable
	}
	if groups.NextMarker != nil || len(groups.TargetGroups) != 1 {
		return hostedexecution.ErrOwnership
	}
	tg := groups.TargetGroups[0]
	if aws.ToString(tg.TargetGroupArn) != s.TargetGroup || aws.ToString(tg.VpcId) != s.ControllerVPC || tg.Protocol != types.ProtocolEnumTcp || aws.ToInt32(tg.Port) != 8042 || tg.TargetType != types.TargetTypeEnumIp || tg.IpAddressType != types.TargetGroupIpAddressTypeEnumIpv4 || !equalSet(tg.LoadBalancerArns, []string{s.LoadBalancer}) || !aws.ToBool(tg.HealthCheckEnabled) || tg.HealthCheckProtocol != types.ProtocolEnumHttp || aws.ToString(tg.HealthCheckPort) != "traffic-port" || aws.ToString(tg.HealthCheckPath) != "/healthz" || tg.Matcher == nil || aws.ToString(tg.Matcher.HttpCode) != "404" {
		return hostedexecution.ErrOwnership
	}
	if err := g.checkTargetAttributes(ctx); err != nil {
		return err
	}
	return g.checkTargets(ctx)
}
