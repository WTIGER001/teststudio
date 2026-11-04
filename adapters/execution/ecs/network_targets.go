package ecsworker

import (
	"context"
	"net/netip"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	l "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func exchangeTargetAttributes() map[string]string {
	return map[string]string{
		"load_balancing.cross_zone.enabled":                                              "true",
		"deregistration_delay.timeout_seconds":                                           "30",
		"deregistration_delay.connection_termination.enabled":                            "true",
		"preserve_client_ip.enabled":                                                     "false",
		"proxy_protocol_v2.enabled":                                                      "false",
		"target_group_health.dns_failover.minimum_healthy_targets.count":                 "1",
		"target_group_health.dns_failover.minimum_healthy_targets.percentage":            "off",
		"target_group_health.unhealthy_state_routing.minimum_healthy_targets.count":      "1",
		"target_group_health.unhealthy_state_routing.minimum_healthy_targets.percentage": "off",
	}
}

func (g *NetworkGuard) checkTargetAttributes(ctx context.Context) error {
	result, err := g.elb.DescribeTargetGroupAttributes(ctx, &elasticloadbalancingv2.DescribeTargetGroupAttributesInput{TargetGroupArn: aws.String(g.settings.TargetGroup)}, oneELB)
	if err != nil || result == nil {
		return hostedexecution.ErrUnavailable
	}
	wanted, seen := exchangeTargetAttributes(), map[string]bool{}
	for _, attribute := range result.Attributes {
		key, value := aws.ToString(attribute.Key), aws.ToString(attribute.Value)
		if key == "" || seen[key] {
			return hostedexecution.ErrOwnership
		}
		seen[key] = true
		if expected, required := wanted[key]; required && expected != value {
			return hostedexecution.ErrOwnership
		}
	}
	for key := range wanted {
		if !seen[key] {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func targetTransition(health *l.TargetHealth) (healthy, retiring, accepted bool) {
	if health == nil {
		return
	}
	switch health.State {
	case l.TargetHealthStateEnumHealthy:
		return true, false, health.Reason == ""
	case l.TargetHealthStateEnumInitial:
		return false, false, health.Reason == "Elb.RegistrationInProgress" || health.Reason == "Elb.InitialHealthChecking"
	case l.TargetHealthStateEnumDraining:
		return false, true, health.Reason == "Target.DeregistrationInProgress"
	case l.TargetHealthStateEnumUnhealthy:
		return false, false, health.Reason == "Target.ResponseCodeMismatch" || health.Reason == "Target.Timeout" || health.Reason == "Target.FailedHealthChecks" || health.Reason == "Elb.InternalError"
	}
	return
}

func (g *NetworkGuard) checkTargets(ctx context.Context) error {
	result, err := g.elb.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{TargetGroupArn: aws.String(g.settings.TargetGroup)}, oneELB)
	if err != nil || result == nil || len(result.TargetHealthDescriptions) == 0 || len(result.TargetHealthDescriptions) > 32 {
		return hostedexecution.ErrUnavailable
	}
	retiring, addresses, healthy := map[string]bool{}, []string{}, 0
	for _, target := range result.TargetHealthDescriptions {
		if target.Target == nil {
			return hostedexecution.ErrOwnership
		}
		ip := aws.ToString(target.Target.Id)
		address, err := netip.ParseAddr(ip)
		if _, duplicate := retiring[ip]; duplicate || err != nil || !address.Is4() || aws.ToInt32(target.Target.Port) != 8042 {
			return hostedexecution.ErrOwnership
		}
		ready, draining, accepted := targetTransition(target.TargetHealth)
		if !accepted {
			return hostedexecution.ErrUnavailable
		}
		if ready {
			healthy++
		}
		retiring[ip] = draining
		addresses = append(addresses, ip)
	}
	interfaces, err := g.ec2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{Filters: []e.Filter{{Name: aws.String("addresses.private-ip-address"), Values: addresses}, {Name: aws.String("vpc-id"), Values: []string{g.settings.ControllerVPC}}}}, oneEC2)
	if err != nil || interfaces == nil {
		return hostedexecution.ErrUnavailable
	}
	if interfaces.NextToken != nil || len(interfaces.NetworkInterfaces) > len(addresses) {
		return hostedexecution.ErrOwnership
	}
	seen := map[string]bool{}
	for _, eni := range interfaces.NetworkInterfaces {
		ip := aws.ToString(eni.PrivateIpAddress)
		if _, expected := retiring[ip]; !expected || seen[ip] || aws.ToString(eni.OwnerId) != g.worker.Account || aws.ToString(eni.VpcId) != g.settings.ControllerVPC || eni.Association != nil || len(eni.Groups) != 1 || aws.ToString(eni.Groups[0].GroupId) != g.settings.ControllerGroup {
			return hostedexecution.ErrOwnership
		}
		for _, address := range eni.PrivateIpAddresses {
			if address.Association != nil {
				return hostedexecution.ErrOwnership
			}
		}
		seen[ip] = true
	}
	for ip, draining := range retiring {
		if !seen[ip] && !draining {
			return hostedexecution.ErrOwnership
		}
	}
	if healthy == 0 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
