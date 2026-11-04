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

func TestAzureFirewallVerifiesBoundResourcesAndCompleteGroupsFreshly(t *testing.T) {
	c, s, f, g := firewallTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 5 {
		t.Fatal("valid firewall rejected", err, f.reads)
	}
	s.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			if err := g.Check(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.reads != 20 {
		t.Fatal("firewall reused stale admission", f.reads)
	}
	slices.Reverse(f.groups)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("group ordering changed semantics", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("partial firewall stage became execution authority")
	}
	c.NetworkProfile = "changed"
	reads := f.reads
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != reads {
		t.Fatal("changed config reached Azure")
	}
}

func TestAzureFirewallRejectsResourceRouteAndSubnetSubstitution(t *testing.T) {
	cases := map[string]func(*firewallFixture){
		"wrong-private-ip": func(f *firewallFixture) {
			f.firewall.Properties.IPConfigurations[0].Properties.PrivateIPAddress = to.Ptr("10.20.1.5")
		},
		"wrong-policy": func(f *firewallFixture) { f.firewall.Properties.FirewallPolicy.ID = to.Ptr("foreign") },
		"classic-network": func(f *firewallFixture) {
			f.firewall.Properties.NetworkRuleCollections = []*n.AzureFirewallNetworkRuleCollection{{}}
		},
		"classic-nat": func(f *firewallFixture) {
			f.firewall.Properties.NatRuleCollections = []*n.AzureFirewallNatRuleCollection{{}}
		},
		"classic-applications": func(f *firewallFixture) {
			f.firewall.Properties.ApplicationRuleCollections = []*n.AzureFirewallApplicationRuleCollection{{}}
		},
		"management-bypass": func(f *firewallFixture) {
			f.firewall.Properties.ManagementIPConfiguration = &n.AzureFirewallIPConfiguration{}
		},
		"virtual-hub": func(f *firewallFixture) { f.firewall.Properties.VirtualHub = &n.SubResource{} },
		"extra-ip": func(f *firewallFixture) {
			f.firewall.Properties.IPConfigurations = append(f.firewall.Properties.IPConfigurations, &n.AzureFirewallIPConfiguration{})
		},
		"wrong-public-ip": func(f *firewallFixture) {
			f.firewall.Properties.IPConfigurations[0].Properties.PublicIPAddress.ID = to.Ptr("foreign")
		},
		"foreign-ip-owner": func(f *firewallFixture) { f.public.Properties.IPConfiguration.ID = to.Ptr("foreign") },
		"dynamic-public-ip": func(f *firewallFixture) {
			f.public.Properties.PublicIPAllocationMethod = to.Ptr(n.IPAllocationMethodDynamic)
		},
		"private-public-ip":      func(f *firewallFixture) { f.public.Properties.IPAddress = to.Ptr("10.20.1.9") },
		"subnet-route-bypass":    func(f *firewallFixture) { f.subnet.Properties.RouteTable = &n.RouteTable{} },
		"subnet-security-bypass": func(f *firewallFixture) { f.subnet.Properties.NetworkSecurityGroup = &n.SecurityGroup{} },
		"subnet-address-drift":   func(f *firewallFixture) { f.subnet.Properties.AddressPrefix = to.Ptr("10.20.3.0/26") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := firewallTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
			if f.pages != 0 {
				t.Fatal("unsafe firewall proceeded to groups")
			}
		})
	}
}

func TestAzureFirewallRejectsEffectivePolicyBypasses(t *testing.T) {
	cases := map[string]func(*firewallFixture){
		"parent-policy": func(f *firewallFixture) { f.policy.Properties.BasePolicy = &n.SubResource{ID: to.Ptr("parent")} },
		"child-policy":  func(f *firewallFixture) { f.policy.Properties.ChildPolicies = []*n.SubResource{{}} },
		"shared-firewall": func(f *firewallFixture) {
			f.policy.Properties.Firewalls = append(f.policy.Properties.Firewalls, &n.SubResource{})
		},
		"proxy":      func(f *firewallFixture) { f.policy.Properties.ExplicitProxySettings = &n.ExplicitProxySettings{} },
		"dns-proxy":  func(f *firewallFixture) { f.policy.Properties.DNSSettings.EnableProxy = to.Ptr(true) },
		"custom-dns": func(f *firewallFixture) { f.policy.Properties.DNSSettings.Servers = []*string{to.Ptr("8.8.8.8")} },
		"missing-deny": func(f *firewallFixture) {
			f.groups[0].Properties.RuleCollections = f.groups[0].Properties.RuleCollections[:1]
		},
		"wrong-deny-action": func(f *firewallFixture) {
			f.groups[0].Properties.RuleCollections[1].(*n.FirewallPolicyFilterRuleCollection).Action.Type = to.Ptr(n.FirewallPolicyFilterRuleCollectionActionTypeAllow)
		},
		"wildcard-allow": func(f *firewallFixture) { f.allow().TargetFqdns = []*string{to.Ptr("*")} },
		"shared-blob": func(f *firewallFixture) {
			f.allow().TargetFqdns = append(f.allow().TargetFqdns, to.Ptr("*.blob.core.windows.net"))
		},
		"foreign-source": func(f *firewallFixture) { f.allow().SourceAddresses = []*string{to.Ptr("*")} },
		"ip-bypass":      func(f *firewallFixture) { f.allow().DestinationAddresses = []*string{to.Ptr("*")} },
		"source-group":   func(f *firewallFixture) { f.allow().SourceIPGroups = []*string{to.Ptr("foreign")} },
		"fqdn-tag":       func(f *firewallFixture) { f.allow().FqdnTags = []*string{to.Ptr("AzureCloud")} },
		"url-bypass":     func(f *firewallFixture) { f.allow().TargetUrls = []*string{to.Ptr("https://foreign.invalid/*")} },
		"default-group-rule": func(f *firewallFixture) {
			f.groups[1].Properties.RuleCollections = f.groups[0].Properties.RuleCollections
		},
		"nat-collection": func(f *firewallFixture) {
			f.groups[0].Properties.RuleCollections = append(f.groups[0].Properties.RuleCollections, &n.FirewallPolicyNatRuleCollection{Name: to.Ptr("nat"), RuleCollectionType: to.Ptr(n.FirewallPolicyRuleCollectionTypeFirewallPolicyNatRuleCollection)})
		},
		"unknown-group": func(f *firewallFixture) { f.groups[1].Name = to.Ptr("foreign") },
		"missing-group-reference": func(f *firewallFixture) {
			f.policy.Properties.RuleCollectionGroups = f.policy.Properties.RuleCollectionGroups[:3]
		},
		"duplicate-group": func(f *firewallFixture) { f.groups[1] = clone(f.groups[0]) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := firewallTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureFirewallPreservesUncertainResourcesAndSafePaging(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "firewall-pending", "policy-pending", "group-pending", "ip-pending", "cancelled", "paged", "foreign-page", "foreign-parent", "cycle", "overflow", "groups-hidden", "groups-missing", "endless"} {
		t.Run(mode, func(t *testing.T) {
			c, _, f, g := firewallTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "firewall-pending":
				f.firewall.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "policy-pending":
				f.policy.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "group-pending":
				f.groups[0].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "ip-pending":
				f.firewall.Properties.IPConfigurations[0].Properties.PrivateIPAddress = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := g.Check(ctx, c)
			if mode == "paged" {
				if err != nil || f.pages != 2 {
					t.Fatal("safe paging lost a complete policy", err, f.pages)
				}
				return
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "foreign-page" || mode == "foreign-parent" || mode == "cycle" || mode == "overflow" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, err, expected)
			if f.pages > 4 || f.reads > 8 {
				t.Fatal("inspection retried or exceeded the fixture bound", f.reads, f.pages)
			}
		})
	}
}
