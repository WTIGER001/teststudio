package ecsworker

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestECSGatewayNetworkAdmitsNarrowPeeringAndVerifiedPrivateTask(t *testing.T) {
	g, f, worker, gateway, task := gatewayNetwork(t)
	if err := g.Admit(t.Context(), worker); err != nil {
		t.Fatal("owned environment network rejected", err, f.calls)
	}
	address, err := g.VerifyGatewayTask(t.Context(), worker, gateway, task)
	if err != nil || address != "10.60.1.21" || len(f.requestIDs) != 3 || f.requestIDs[0] != f.gateway.PeeringConnection || f.requestIDs[1] != f.gateway.NATGateways[0] || f.requestIDs[2] != aws.ToString(f.internet.InternetGatewayId) {
		t.Fatal("gateway task did not retain its owned private path", address, err)
	}
	f.groups[0].IpPermissions[0].FromPort = aws.Int32(80)
	g.validUntil = time.Time{}
	if err := g.AdmitGateway(t.Context(), worker, gateway); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("expired network snapshot concealed gateway ingress drift", err)
	}
}

func TestECSGatewayNetworkRejectsDirectEgressForeignPeersAndUnverifiedExchange(t *testing.T) {
	cases := map[string]func(*gatewayNetworkFixture){
		"worker-nat": func(f *gatewayNetworkFixture) {
			f.DescribeRouteTablesValue.RouteTables[0].Routes = append(f.DescribeRouteTablesValue.RouteTables[0].Routes, e.Route{DestinationCidrBlock: aws.String("0.0.0.0/0"), NatGatewayId: f.nat.NatGatewayId, State: "active"})
		},
		"whole-peer-vpc": func(f *gatewayNetworkFixture) {
			f.DescribeRouteTablesValue.RouteTables[0].Routes[2].DestinationCidrBlock = f.vpc.CidrBlock
		},
		"worker-target-port": func(f *gatewayNetworkFixture) {
			f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress[3].FromPort = aws.Int32(443)
			f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress[3].ToPort = aws.Int32(443)
		},
		"worker-arbitrary-egress": func(f *gatewayNetworkFixture) {
			f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress = append(f.DescribeSecurityGroupsValue.SecurityGroups[0].IpPermissionsEgress, e.IpPermission{IpProtocol: aws.String("-1"), IpRanges: []e.IpRange{{CidrIp: aws.String("0.0.0.0/0")}}})
		},
		"foreign-peer-account": func(f *gatewayNetworkFixture) { f.peer.AccepterVpcInfo.OwnerId = aws.String("999999999999") },
		"cross-region-peer":    func(f *gatewayNetworkFixture) { f.peer.AccepterVpcInfo.Region = aws.String("us-west-2") },
		"inactive-peer":        func(f *gatewayNetworkFixture) { f.peer.Status.Code = "deleted" },
		"public-task-subnet":   func(f *gatewayNetworkFixture) { f.subnets[0].MapPublicIpOnLaunch = aws.Bool(true) },
		"wrong-peer-sg": func(f *gatewayNetworkFixture) {
			f.groups[0].IpPermissions[0].UserIdGroupPairs[0].VpcPeeringConnectionId = aws.String("pcx-bbbbbbbb")
		},
		"public-gateway-ingress": func(f *gatewayNetworkFixture) {
			f.groups[0].IpPermissions[0].IpRanges = []e.IpRange{{CidrIp: aws.String("0.0.0.0/0")}}
		},
		"no-return-route": func(f *gatewayNetworkFixture) {
			f.routes[0].Routes = append(f.routes[0].Routes[:1], f.routes[0].Routes[2:]...)
		},
		"gateway-direct-internet": func(f *gatewayNetworkFixture) {
			f.routes[0].Routes[3].NatGatewayId = nil
			f.routes[0].Routes[3].GatewayId = f.internet.InternetGatewayId
		},
		"foreign-nat-vpc":      func(f *gatewayNetworkFixture) { f.nat.VpcId = aws.String(f.settings.ControllerVPC) },
		"private-nat":          func(f *gatewayNetworkFixture) { f.nat.ConnectivityType = "private" },
		"unattached-internet":  func(f *gatewayNetworkFixture) { f.internet.Attachments = nil },
		"foreign-exchange":     func(f *gatewayNetworkFixture) { f.endpoint.ServiceName = aws.String("another-service") },
		"unaccepted-exchange":  func(f *gatewayNetworkFixture) { f.endpoint.State = "pendingAcceptance" },
		"missing-private-dns":  func(f *gatewayNetworkFixture) { f.endpoint.PrivateDnsEnabled = aws.Bool(false) },
		"worker-dns-wildcard":  func(f *gatewayNetworkFixture) { f.allowDomains.Domains = append(f.allowDomains.Domains, "*") },
		"private-native-error": func(f *gatewayNetworkFixture) { f.fail = "GatewayPeering" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			g, f, worker, gateway, _ := gatewayNetwork(t)
			mutate(f)
			err := g.AdmitGateway(t.Context(), worker, gateway)
			if err == nil || strings.Contains(err.Error(), "canary") {
				t.Fatal("changed native boundary passed or exposed private diagnostics", err)
			}
		})
	}
}

func TestECSGatewayNetworkBindsTaskInterfaceAndFreezesConfiguredScope(t *testing.T) {
	g, f, worker, gateway, task := gatewayNetwork(t)
	if _, err := g.VerifyGatewayTask(t.Context(), worker, gateway, task); err != nil {
		t.Fatal(err)
	}
	f.eni.Association = &e.NetworkInterfaceAssociation{PublicIp: aws.String("203.0.113.5")}
	if _, err := g.VerifyGatewayTask(t.Context(), worker, gateway, task); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("public gateway interface retained its private binding", err)
	}
	f.eni.Association = nil
	f.eni.Groups = append(f.eni.Groups, e.GroupIdentifier{GroupId: aws.String(worker.SecurityGroups[0])})
	if _, err := g.VerifyGatewayTask(t.Context(), worker, gateway, task); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("another security group acquired a gateway identity", err)
	}
	settings := g.settings
	settings.Gateway = nil
	if _, err := settings.Bind(worker); !errors.Is(err, hostedexecution.ErrConfiguration) {
		t.Fatal("removing gateway controls preserved the environment network identity", err)
	}
	if err := g.AdmitGateway(t.Context(), worker, GatewayConfig{}); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("another task configuration acquired network admission", err)
	}
	if _, ok := any(&ec2.Client{}).(GatewayEC2); !ok {
		t.Fatal("native SDK cannot supply gateway network observations")
	}
}

func TestECSGatewayNetworkCannotSelectIncompleteProductionRegistration(t *testing.T) {
	sdk, _, store, credentials := sdkProtocol(t)
	guard, _, worker, _, _ := gatewayNetwork(t)
	settings := Settings{Worker: worker, Network: guard.settings}
	if _, err := settings.open(t.Context(), store, nil, credentials, sdk, sdk); !errors.Is(err, hostedexecution.ErrConfiguration) || len(sdk.calls) != 0 {
		t.Fatal("gateway settings selected an offline production executor", err)
	}
}
