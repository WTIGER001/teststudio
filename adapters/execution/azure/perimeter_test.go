package azureworker

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePerimeterReadsExactRulesAndRoutesWithoutCaching(t *testing.T) {
	c, s, top, f, g := perimeterTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 2 || top.reads != 6 {
		t.Fatal("valid perimeter was rejected", err)
	}
	s.PrivateEndpointIPs[0] = "192.168.1.1"
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			if err := g.Check(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.reads != 8 || top.reads != 24 {
		t.Fatal("perimeter reused a stale inspection", f.reads, top.reads)
	}
	slices.Reverse(f.group.Properties.SecurityRules)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("response rule order changed priority semantics", err)
	}
	c.NetworkProfile = "changed"
	reads := top.reads + f.reads
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if reads != top.reads+f.reads {
		t.Fatal("changed configuration reached Azure")
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("partial perimeter became complete admission")
	}
}

func TestAzurePerimeterRejectsSecurityRuleBypasses(t *testing.T) {
	cases := map[string]func(*perimeterFixture){
		"extra-rule": func(f *perimeterFixture) {
			f.group.Properties.SecurityRules = append(f.group.Properties.SecurityRules, &n.SecurityRule{Name: to.Ptr("escape")})
		},
		"missing-rule": func(f *perimeterFixture) { f.group.Properties.SecurityRules = f.group.Properties.SecurityRules[1:] },
		"duplicate-rule": func(f *perimeterFixture) {
			f.group.Properties.SecurityRules[0] = clone(f.group.Properties.SecurityRules[1])
		},
		"internet-allow": func(f *perimeterFixture) {
			f.rule("worker-platform-registry").Properties.DestinationAddressPrefix = to.Ptr("Internet")
		},
		"storage-bypass": func(f *perimeterFixture) {
			f.rule("worker-platform-registry").Properties.DestinationAddressPrefix = to.Ptr("Storage")
		},
		"wrong-direction": func(f *perimeterFixture) {
			f.rule("worker-deny-out").Properties.Direction = to.Ptr(n.SecurityRuleDirectionInbound)
		},
		"wrong-priority": func(f *perimeterFixture) { f.rule("worker-deny-out").Properties.Priority = to.Ptr[int32](4095) },
		"deny-became-allow": func(f *perimeterFixture) {
			f.rule("worker-deny-out").Properties.Access = to.Ptr(n.SecurityRuleAccessAllow)
		},
		"allow-metadata": func(f *perimeterFixture) {
			f.rule("worker-deny-metadata").Properties.Access = to.Ptr(n.SecurityRuleAccessAllow)
		},
		"unfiltered-dns": func(f *perimeterFixture) {
			f.rule("worker-platform-dns-udp").Properties.DestinationAddressPrefix = to.Ptr("*")
		},
		"extra-ports": func(f *perimeterFixture) {
			f.rule("worker-platform-registry").Properties.DestinationPortRanges = []*string{to.Ptr("443"), to.Ptr("80")}
		},
		"extra-source": func(f *perimeterFixture) {
			f.rule("worker-platform-registry").Properties.SourceAddressPrefixes = []*string{to.Ptr("0.0.0.0/0")}
		},
		"private-endpoint-bypass": func(f *perimeterFixture) {
			f.rule("worker-private-endpoints").Properties.DestinationAddressPrefixes = append(f.rule("worker-private-endpoints").Properties.DestinationAddressPrefixes, to.Ptr("10.21.0.4"))
		},
		"application-group": func(f *perimeterFixture) {
			f.rule("worker-private-endpoints").Properties.DestinationApplicationSecurityGroups = []*n.ApplicationSecurityGroup{{}}
		},
		"foreign-child":               func(f *perimeterFixture) { f.rule("worker-deny-out").ID = to.Ptr("foreign") },
		"connections-not-reevaluated": func(f *perimeterFixture) { f.group.Properties.FlushConnection = to.Ptr(false) },
		"shared-subnet": func(f *perimeterFixture) {
			f.group.Properties.Subnets = append(f.group.Properties.Subnets, &n.Subnet{ID: to.Ptr("foreign")})
		},
		"attached-interface": func(f *perimeterFixture) { f.group.Properties.NetworkInterfaces = []*n.Interface{{}} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, _, f, g := perimeterTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
			if f.reads != 1 {
				t.Fatal("unsafe NSG proceeded to routing")
			}
		})
	}
}

func TestAzurePerimeterRejectsRoutingBypasses(t *testing.T) {
	cases := map[string]func(*perimeterFixture){
		"internet-hop": func(f *perimeterFixture) {
			f.routes.Properties.Routes[0].Properties.NextHopType = to.Ptr(n.RouteNextHopTypeInternet)
		},
		"wrong-appliance": func(f *perimeterFixture) {
			f.routes.Properties.Routes[0].Properties.NextHopIPAddress = to.Ptr("10.20.1.5")
		},
		"service-tag-prefix": func(f *perimeterFixture) {
			f.routes.Properties.Routes[0].Properties.AddressPrefix = to.Ptr("AzureCloud")
		},
		"missing-default": func(f *perimeterFixture) { f.routes.Properties.Routes = nil },
		"extra-specific-route": func(f *perimeterFixture) {
			f.routes.Properties.Routes = append(f.routes.Properties.Routes, &n.Route{Name: to.Ptr("bypass"), Properties: &n.RoutePropertiesFormat{AddressPrefix: to.Ptr("8.8.8.8/32"), NextHopType: to.Ptr(n.RouteNextHopTypeInternet)}})
		},
		"bgp-enabled":   func(f *perimeterFixture) { f.routes.Properties.DisableBgpRoutePropagation = to.Ptr(false) },
		"unknown-bgp":   func(f *perimeterFixture) { f.routes.Properties.DisableBgpRoutePropagation = nil },
		"bgp-override":  func(f *perimeterFixture) { f.routes.Properties.Routes[0].Properties.HasBgpOverride = to.Ptr(true) },
		"foreign-child": func(f *perimeterFixture) { f.routes.Properties.Routes[0].ID = to.Ptr("foreign") },
		"shared-table": func(f *perimeterFixture) {
			f.routes.Properties.Subnets = append(f.routes.Properties.Subnets, &n.Subnet{})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, _, f, g := perimeterTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzurePerimeterPreservesTopologyReadAndProvisioningUncertainty(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "group-pending", "rule-pending", "table-pending", "route-pending", "topology-drift", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			c, _, top, f, g := perimeterTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "group-pending":
				f.group.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "rule-pending":
				f.rule("worker-deny-out").Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "table-pending":
				f.routes.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "route-pending":
				f.routes.Properties.Routes[0].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "topology-drift":
				top.wnet.Properties.VirtualNetworkPeerings = []*n.VirtualNetworkPeering{{}}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "topology-drift" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, g.Check(ctx, c), expected)
			if f.reads > 2 || (mode == "topology-drift" || mode == "cancelled") && f.reads != 0 {
				t.Fatal("unproven topology proceeded or retried", f.reads)
			}
		})
	}
}
