package ecsworker

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	e "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	l "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func addBackend(f *networkFixture, ip, state, reason string) {
	target := *copyValue(&f.DescribeTargetHealthValue.TargetHealthDescriptions[0])
	target.Target.Id = aws.String(ip)
	target.TargetHealth = &l.TargetHealth{State: l.TargetHealthStateEnum(state), Reason: l.TargetHealthReasonEnum(reason)}
	f.DescribeTargetHealthValue.TargetHealthDescriptions = append(f.DescribeTargetHealthValue.TargetHealthDescriptions, target)
	eni := *copyValue(&f.backendENI.NetworkInterfaces[0])
	eni.PrivateIpAddress = aws.String(ip)
	f.backendENI.NetworkInterfaces = append(f.backendENI.NetworkInterfaces, eni)
}

func TestECSNetworkMaintainsAdmissionAcrossBackendRolloutAndRestart(t *testing.T) {
	g, f, c, task := network(t)
	check := func(label string) {
		t.Helper()
		g.validUntil = time.Time{}
		before := f.calls["DescribeNetworkInterfaces"]
		if err := g.VerifyTask(t.Context(), c, task); err != nil {
			t.Fatal(label, err)
		}
		if f.calls["DescribeNetworkInterfaces"]-before != 2 {
			t.Fatal("task verification must use one backend batch and one worker lookup")
		}
	}
	check("original backend")
	addBackend(f, "10.50.2.12", "initial", "Elb.RegistrationInProgress")
	check("replacement registering")
	f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth.Reason = "Elb.InitialHealthChecking"
	check("replacement checking health")
	f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth = &l.TargetHealth{State: "healthy"}
	check("both backends ready")
	f.DescribeTargetHealthValue.TargetHealthDescriptions[0].TargetHealth = &l.TargetHealth{State: "draining", Reason: "Target.DeregistrationInProgress"}
	check("old backend draining")
	f.backendENI.NetworkInterfaces = f.backendENI.NetworkInterfaces[1:]
	check("retiring interface removed")
	var err error
	g, err = NewNetworkGuard(c, f.settings, f, f, f, f)
	if err != nil {
		t.Fatal(err)
	}
	check("new controller during draining")
	f.DescribeTargetHealthValue.TargetHealthDescriptions = f.DescribeTargetHealthValue.TargetHealthDescriptions[1:]
	check("replacement only")
}

func TestECSNetworkBatchesMaximumBackendPool(t *testing.T) {
	g, f, c, _ := network(t)
	wanted := []string{"10.50.1.11"}
	for i := 1; i < 32; i++ {
		ip := fmt.Sprintf("10.50.2.%d", i)
		addBackend(f, ip, "initial", "Elb.InitialHealthChecking")
		wanted = append(wanted, ip)
	}
	if err := g.Admit(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	if f.calls["DescribeNetworkInterfaces"] != 1 || f.backendInput == nil || len(f.backendInput.Filters) != 2 || len(f.backendInput.NetworkInterfaceIds) != 0 {
		t.Fatal("backend lookups were not batched")
	}
	filters := f.backendInput.Filters
	if aws.ToString(filters[0].Name) != "addresses.private-ip-address" || !slices.Equal(filters[0].Values, wanted) || aws.ToString(filters[1].Name) != "vpc-id" || !slices.Equal(filters[1].Values, []string{f.settings.ControllerVPC}) {
		t.Fatal("backend lookup widened its scope", filters)
	}
}

func TestECSNetworkRolloutRejectsOwnershipDriftAndUncertainTargets(t *testing.T) {
	cases := map[string]func(*networkFixture){
		"initial-foreign-account": func(f *networkFixture) { f.backendENI.NetworkInterfaces[1].OwnerId = aws.String("999999999999") },
		"initial-other-vpc":       func(f *networkFixture) { f.backendENI.NetworkInterfaces[1].VpcId = aws.String("vpc-cccccccc") },
		"initial-other-group": func(f *networkFixture) {
			f.backendENI.NetworkInterfaces[1].Groups[0].GroupId = aws.String("sg-ffffffff")
		},
		"initial-extra-group": func(f *networkFixture) {
			f.backendENI.NetworkInterfaces[1].Groups = append(f.backendENI.NetworkInterfaces[1].Groups, e.GroupIdentifier{GroupId: aws.String("sg-ffffffff")})
		},
		"initial-public": func(f *networkFixture) {
			f.backendENI.NetworkInterfaces[1].Association = &e.NetworkInterfaceAssociation{PublicIp: aws.String("198.51.100.1")}
		},
		"initial-secondary-public": func(f *networkFixture) {
			f.backendENI.NetworkInterfaces[1].PrivateIpAddresses = []e.NetworkInterfacePrivateIpAddress{{Association: &e.NetworkInterfaceAssociation{PublicIp: aws.String("198.51.100.1")}}}
		},
		"initial-missing": func(f *networkFixture) { f.backendENI.NetworkInterfaces = f.backendENI.NetworkInterfaces[:1] },
		"healthy-missing": func(f *networkFixture) { f.backendENI.NetworkInterfaces = f.backendENI.NetworkInterfaces[1:] },
		"secondary-match": func(f *networkFixture) { f.backendENI.NetworkInterfaces[1].PrivateIpAddress = aws.String("10.50.2.99") },
		"duplicate-target": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].Target.Id = aws.String("10.50.1.11")
		},
		"wrong-port": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].Target.Port = aws.Int32(8040)
		},
		"ipv6": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].Target.Id = aws.String("2001:db8::1")
		},
		"missing-target": func(f *networkFixture) { f.DescribeTargetHealthValue.TargetHealthDescriptions[1].Target = nil },
		"missing-health": func(f *networkFixture) { f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth = nil },
		"unknown-state": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth.State = "new-state"
		},
		"unknown-reason": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth.Reason = "new-reason"
		},
		"unused": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth = &l.TargetHealth{State: "unused", Reason: "Target.NotInUse"}
		},
		"unavailable": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth = &l.TargetHealth{State: "unavailable", Reason: "Elb.InternalError"}
		},
		"draining-foreign-interface": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth = &l.TargetHealth{State: "draining", Reason: "Target.DeregistrationInProgress"}
			f.backendENI.NetworkInterfaces[1].Groups[0].GroupId = aws.String("sg-ffffffff")
		},
		"draining-without-proof": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[1].TargetHealth = &l.TargetHealth{State: "draining"}
			f.backendENI.NetworkInterfaces = f.backendENI.NetworkInterfaces[:1]
		},
		"duplicate-interface":   func(f *networkFixture) { f.backendENI.NetworkInterfaces[1] = f.backendENI.NetworkInterfaces[0] },
		"pagination":            func(f *networkFixture) { f.backendENI.NextToken = aws.String("private-network-canary") },
		"health-unavailable":    func(f *networkFixture) { f.fail = "DescribeTargetHealth" },
		"interface-unavailable": func(f *networkFixture) { f.fail = "DescribeNetworkInterfaces" },
		"no-healthy": func(f *networkFixture) {
			f.DescribeTargetHealthValue.TargetHealthDescriptions[0].TargetHealth = &l.TargetHealth{State: "draining", Reason: "Target.DeregistrationInProgress"}
		},
		"oversized": func(f *networkFixture) {
			for range 31 {
				addBackend(f, "10.50.2.13", "healthy", "")
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			g, f, c, _ := network(t)
			addBackend(f, "10.50.2.12", "initial", "Elb.InitialHealthChecking")
			mutate(f)
			err := g.Admit(t.Context(), c)
			if err == nil || !errors.Is(err, hostedexecution.ErrOwnership) && !errors.Is(err, hostedexecution.ErrUnavailable) || strings.Contains(err.Error(), "canary") {
				t.Fatal("unsafe or uncertain rollout admitted or leaked diagnostics", err)
			}
		})
	}
}

func TestECSNetworkAllowsOwnedUnhealthyReplacementOnlyWithHealthyBackend(t *testing.T) {
	for _, reason := range []string{"Target.ResponseCodeMismatch", "Target.Timeout", "Target.FailedHealthChecks", "Elb.InternalError"} {
		t.Run(reason, func(t *testing.T) {
			g, f, c, _ := network(t)
			addBackend(f, "10.50.2.12", "unhealthy", reason)
			if err := g.Admit(t.Context(), c); err != nil {
				t.Fatal(err)
			}
			g.validUntil = time.Time{}
			f.DescribeTargetHealthValue.TargetHealthDescriptions[0].TargetHealth = &l.TargetHealth{State: "unhealthy", Reason: "Target.FailedHealthChecks"}
			if err := g.Admit(t.Context(), c); !errors.Is(err, hostedexecution.ErrUnavailable) {
				t.Fatal("no healthy backend admitted", err)
			}
		})
	}
}

func TestECSNetworkRequiresExactRolloutAttributes(t *testing.T) {
	for key := range exchangeTargetAttributes() {
		for _, mode := range []string{"changed", "missing", "duplicate"} {
			t.Run(key+"/"+mode, func(t *testing.T) {
				g, f, c, _ := network(t)
				for i, attribute := range f.DescribeTargetGroupAttributesValue.Attributes {
					if aws.ToString(attribute.Key) != key {
						continue
					}
					switch mode {
					case "changed":
						f.DescribeTargetGroupAttributesValue.Attributes[i].Value = aws.String("changed")
					case "missing":
						f.DescribeTargetGroupAttributesValue.Attributes = slices.Delete(f.DescribeTargetGroupAttributesValue.Attributes, i, i+1)
					case "duplicate":
						f.DescribeTargetGroupAttributesValue.Attributes = append(f.DescribeTargetGroupAttributesValue.Attributes, attribute)
					}
					break
				}
				if err := g.Admit(t.Context(), c); !errors.Is(err, hostedexecution.ErrOwnership) {
					t.Fatal("routing drift admitted", err)
				}
			})
		}
	}
	g, f, c, _ := network(t)
	f.fail = "DescribeTargetGroupAttributes"
	if err := g.Admit(t.Context(), c); !errors.Is(err, hostedexecution.ErrUnavailable) || strings.Contains(err.Error(), "canary") {
		t.Fatal("private attribute failure escaped", err)
	}
}
