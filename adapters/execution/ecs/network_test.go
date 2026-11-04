package ecsworker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	l "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/route53profiles"
	p "github.com/aws/aws-sdk-go-v2/service/route53profiles/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestECSNetworkAdmitsOnlyBoundPrivatePoolAndExactTaskInterface(t *testing.T) {
	g, f, c, task := network(t)
	if err := g.Admit(t.Context(), c); err != nil {
		t.Fatal("owned private pool rejected", err, f.calls)
	}
	if err := g.VerifyTask(t.Context(), c, task); err != nil {
		t.Fatal("owned private task rejected", err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			if err := g.Admit(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.calls["DescribeVpcs"] != 1 {
		t.Fatal("concurrent checks repeated the pool snapshot")
	}
	c.NetworkProfile = ""
	if err := g.Admit(t.Context(), c); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("another isolation profile reused admission", err)
	}
}

func TestECSNetworkMutationStopsAdmissionWithoutPrivateDiagnostics(t *testing.T) {
	cases := map[string]func(*networkFixture){
		"nat-route": func(f *networkFixture) {
			f.DescribeRouteTablesValue.RouteTables[0].Routes = append(f.DescribeRouteTablesValue.RouteTables[0].Routes, e.Route{DestinationCidrBlock: aws.String("0.0.0.0/0"), NatGatewayId: aws.String("nat-aaaaaaaa"), State: "active"})
		},
		"public-subnet": func(f *networkFixture) { f.DescribeSubnetsValue.Subnets[0].MapPublicIpOnLaunch = aws.Bool(true) },
		"external-dns-server": func(f *networkFixture) {
			f.DescribeDhcpOptionsValue.DhcpOptions[0].DhcpConfigurations[0].Values[0].Value = aws.String("8.8.8.8")
		},
		"endpoint-internet-egress": func(f *networkFixture) {
			f.DescribeSecurityGroupsValue.SecurityGroups[1].IpPermissionsEgress = []e.IpPermission{{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(443), ToPort: aws.Int32(443), IpRanges: []e.IpRange{{CidrIp: aws.String("0.0.0.0/0")}}}}
		},
		"worker-inbound": func(f *networkFixture) {
			f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissions = []e.IpPermission{sgRule("123456789012", "sg-bbbbbbbb", 443)}
		},
		"internet-egress": func(f *networkFixture) {
			f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress = append(f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress, e.IpPermission{IpProtocol: aws.String("-1"), IpRanges: []e.IpRange{{CidrIp: aws.String("0.0.0.0/0")}}})
		},
		"s3-write": func(f *networkFixture) {
			for i := range f.DescribeVpcEndpointsValue.VpcEndpoints {
				if *f.DescribeVpcEndpointsValue.VpcEndpoints[i].VpcEndpointId == f.settings.S3Endpoint {
					f.DescribeVpcEndpointsValue.VpcEndpoints[i].PolicyDocument = aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}`)
				}
			}
		},
		"ecr-unscoped": func(f *networkFixture) {
			for i := range f.DescribeVpcEndpointsValue.VpcEndpoints {
				if *f.DescribeVpcEndpointsValue.VpcEndpoints[i].VpcEndpointId == f.settings.APIEndpoint {
					f.DescribeVpcEndpointsValue.VpcEndpoints[i].PolicyDocument = aws.String(`{"Statement":[{"Effect":"Allow","Action":"*","Resource":"*","Principal":"*"}]}`)
				}
			}
		},
		"public-loadbalancer": func(f *networkFixture) {
			f.DescribeLoadBalancersValue.LoadBalancers[0].Scheme = l.LoadBalancerSchemeEnumInternetFacing
		},
		"plaintext-listener": func(f *networkFixture) { f.DescribeListenersValue.Listeners[0].Protocol = l.ProtocolEnumTcp },
		"wrong-target-port":  func(f *networkFixture) { f.DescribeTargetGroupsValue.TargetGroups[0].Port = aws.Int32(8040) },
		"foreign-service-principal": func(f *networkFixture) {
			f.DescribeVpcEndpointServicePermissionsValue.AllowedPrincipals[0].Principal = aws.String("*")
		},
		"unverified-private-dns": func(f *networkFixture) {
			f.DescribeVpcEndpointServiceConfigurationsValue.ServiceConfigurations[0].PrivateDnsNameConfiguration.State = "pendingVerification"
		},
		"foreign-firewall-owner": func(f *networkFixture) {
			f.GetFirewallRuleGroupValue.FirewallRuleGroup.Arn = aws.String("arn:aws:route53resolver:us-east-1:999999999999:firewall-rule-group/" + f.settings.FirewallGroup)
		},
		"foreign-domain-list": func(f *networkFixture) {
			f.allowList.FirewallDomainList.Arn = aws.String("arn:aws:route53resolver:us-west-2:123456789012:firewall-domain-list/" + f.settings.AllowList)
		},
		"health-on-api-port": func(f *networkFixture) {
			f.DescribeTargetGroupsValue.TargetGroups[0].HealthCheckPort = aws.String("8040")
		},
		"dual-stack-s3": func(f *networkFixture) {
			for i := range f.DescribeVpcEndpointsValue.VpcEndpoints {
				if *f.DescribeVpcEndpointsValue.VpcEndpoints[i].VpcEndpointId == f.settings.S3Endpoint {
					f.DescribeVpcEndpointsValue.VpcEndpoints[i].IpAddressType = e.IpAddressTypeDualstack
				}
			}
		},
		"duplicate-policy-key": func(f *networkFixture) {
			for i := range f.DescribeVpcEndpointsValue.VpcEndpoints {
				if *f.DescribeVpcEndpointsValue.VpcEndpoints[i].VpcEndpointId == f.settings.APIEndpoint {
					value := *f.DescribeVpcEndpointsValue.VpcEndpoints[i].PolicyDocument
					f.DescribeVpcEndpointsValue.VpcEndpoints[i].PolicyDocument = aws.String(`{"Version":"ignored",` + value[1:])
				}
			}
		},
		"dns-profile-overlay": func(f *networkFixture) {
			f.profiles = &route53profiles.ListProfileAssociationsOutput{ProfileAssociations: []p.ProfileAssociation{{ResourceId: aws.String(f.settings.VPC), Status: p.ProfileStatusComplete}}}
		},
		"dns-profile-unavailable": func(f *networkFixture) { f.fail = "ListProfileAssociations" },
		"dns-fail-open":           func(f *networkFixture) { f.GetFirewallConfigValue.FirewallConfig.FirewallFailOpen = "ENABLED" },
		"dns-allow-wildcard":      func(f *networkFixture) { f.allowDomains.Domains[0] = "*.example.test" },
		"dns-priority":            func(f *networkFixture) { f.ListFirewallRulesValue.FirewallRules[1].Priority = aws.Int32(50) },
		"dns-extra-group": func(f *networkFixture) {
			f.ListFirewallRuleGroupAssociationsValue.FirewallRuleGroupAssociations = append(f.ListFirewallRuleGroupAssociationsValue.FirewallRuleGroupAssociations, f.ListFirewallRuleGroupAssociationsValue.FirewallRuleGroupAssociations[0])
		},
		"private-sdk-error": func(f *networkFixture) { f.fail = "DescribeVpcEndpoints" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			g, f, c, _ := network(t)
			mutate(f)
			err := g.Admit(t.Context(), c)
			if err == nil || strings.Contains(err.Error(), "canary") {
				t.Fatal("mutated pool passed or exposed a private error", err)
			}
		})
	}
}

func TestECSNetworkRejectsTaskInterfaceDriftAndWaitsForAttachment(t *testing.T) {
	for _, name := range []string{"public-address", "extra-group", "wrong-subnet", "wrong-ip", "ipv6", "source-check"} {
		t.Run(name, func(t *testing.T) {
			g, f, c, task := network(t)
			eni := &f.workerENI.NetworkInterfaces[0]
			switch name {
			case "public-address":
				eni.Association = &e.NetworkInterfaceAssociation{PublicIp: aws.String("203.0.113.1")}
			case "extra-group":
				eni.Groups = append(eni.Groups, e.GroupIdentifier{GroupId: aws.String("sg-ffffffff")})
			case "wrong-subnet":
				eni.SubnetId = aws.String("subnet-cccccccc")
			case "wrong-ip":
				eni.PrivateIpAddress = aws.String("10.40.1.22")
			case "ipv6":
				eni.Ipv6Address = aws.String("2001:db8::1")
			case "source-check":
				eni.SourceDestCheck = aws.Bool(false)
			}
			if err := g.VerifyTask(t.Context(), c, task); !errors.Is(err, hostedexecution.ErrOwnership) {
				t.Fatal("task interface drift was admitted", err)
			}
		})
	}
	g, _, c, task := network(t)
	task.Attachments = nil
	if err := g.VerifyTask(t.Context(), c, task); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("pending interface was not retryable", err)
	}
}

func TestECSSettingsAreBoundedPrivateAndBindIsolationIdentity(t *testing.T) {
	g, _, c, _ := network(t)
	settings := Settings{Worker: c, Network: g.settings}
	settings.Worker.NetworkProfile = ""
	data, _ := json.Marshal(settings)
	file := filepath.Join(t.TempDir(), "workers.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSettings(file, "", nil)
	if err != nil || loaded.Worker.NetworkProfile != c.NetworkProfile {
		t.Fatal("network identity was not bound during load", err)
	}
	lookup := func(string) (string, bool) { return string(data), true }
	if _, err := LoadSettings("", "TESTSTUDIO_ECS_CONFIGURATION", lookup); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSettings(file, "TESTSTUDIO_ECS_CONFIGURATION", lookup); err == nil {
		t.Fatal("two configuration sources were accepted")
	}
	bad := strings.TrimSuffix(string(data), "}") + `,"private-canary":"must-not-appear"}`
	if _, err := LoadSettings("", "TESTSTUDIO_ECS_CONFIGURATION", func(string) (string, bool) { return bad, true }); err == nil || strings.Contains(err.Error(), "canary") {
		t.Fatal("unknown private configuration was not safely rejected", err)
	}
	settings.Network.ControllerVPC = settings.Network.VPC
	if _, err := settings.Network.Bind(settings.Worker); err == nil {
		t.Fatal("shared controller VPC could inherit the worker DNS policy")
	}
	settings.Network = g.settings
	settings.Network.RepositoryPrefix = "another-prefix"
	if _, err := settings.Network.Bind(c); err == nil {
		t.Fatal("changed network policy retained the previous identity")
	}
}

func TestECSNetworkCacheRechecksDriftAfterItsBound(t *testing.T) {
	g, f, c, _ := network(t)
	if err := g.Admit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	f.GetFirewallConfigValue.FirewallConfig.FirewallFailOpen = "ENABLED"
	g.mu.Lock()
	g.validUntil = time.Time{}
	g.mu.Unlock()
	if err := g.Admit(context.Background(), c); err == nil {
		t.Fatal("expired snapshot concealed policy drift")
	}
}

func TestECSNetworkSettingsRejectUnsupportedOriginsAndResourceScope(t *testing.T) {
	g, _, c, _ := network(t)
	for name, mutate := range map[string]func(*Config, *NetworkSettings){
		"ip-origin":      func(c *Config, s *NetworkSettings) { c.ExchangeOrigin = "https://10.40.1.5" },
		"non-https-port": func(c *Config, s *NetworkSettings) { c.ExchangeOrigin = "https://worker.example.test:8042" },
		"wildcard-host":  func(c *Config, s *NetworkSettings) { c.ExchangeOrigin = "https://*.example.test" },
		"foreign-region": func(c *Config, s *NetworkSettings) {
			s.LoadBalancer = strings.Replace(s.LoadBalancer, "us-east-1", "us-west-2", 1)
		},
		"invalid-lb-resource": func(c *Config, s *NetworkSettings) { s.LoadBalancer += "/extra" },
		"invalid-service": func(c *Config, s *NetworkSettings) {
			s.ExchangeService = strings.Replace(s.ExchangeService, "55555555555555555", "zzzzzzzzzzzzzzzzz", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := c
			c.NetworkProfile = ""
			s := g.settings
			mutate(&c, &s)
			if _, err := s.Bind(c); !errors.Is(err, hostedexecution.ErrConfiguration) {
				t.Fatal("invalid network settings accepted", err)
			}
		})
	}
	if _, err := LoadSettings("", "TESTSTUDIO_ECS_CONFIGURATION", nil); err == nil {
		t.Fatal("nil lookup accepted")
	}
}
