package azureworker

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureDNSVerifiesActualAssociationAndCompleteInventoriesFreshly(t *testing.T) {
	c, s, f, g := dnsTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 6 {
		t.Fatal("valid DNS profile rejected", err, f.reads)
	}
	s.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			if err := g.Check(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.reads != 24 {
		t.Fatal("DNS reused stale inspection", f.reads)
	}
	slices.Reverse(f.allow.Properties.Domains)
	slices.Reverse(f.rules)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("response ordering changed policy semantics", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("DNS stage alone authorized execution")
	}
	c.NetworkProfile = "changed"
	reads := f.reads
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != reads {
		t.Fatal("changed profile reached Azure")
	}
}

func TestAzureDNSRejectsDomainRuleAndAssociationBypasses(t *testing.T) {
	cases := map[string]func(*dnsFixture){
		"wildcard-allow": func(f *dnsFixture) { f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr(".")) },
		"public-application-domain": func(f *dnsFixture) {
			f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr("foreign.invalid"))
		},
		"shared-blob-domain": func(f *dnsFixture) {
			f.allow.Properties.Domains = append(f.allow.Properties.Domains, to.Ptr("*.blob.core.windows.net"))
		},
		"duplicate-domain": func(f *dnsFixture) {
			f.allow.Properties.Domains = append(f.allow.Properties.Domains, f.allow.Properties.Domains[0])
		},
		"not-default-block": func(f *dnsFixture) { f.block.Properties.Domains = []*string{to.Ptr("malicious.invalid")} },
		"bulk-url":          func(f *dnsFixture) { f.allow.Properties.DomainsURL = to.Ptr("https://foreign.invalid/credential") },
		"disabled-block": func(f *dnsFixture) {
			f.rules[1].Properties.DNSSecurityRuleState = to.Ptr(d.DNSSecurityRuleStateDisabled)
		},
		"allow-instead-of-block": func(f *dnsFixture) { f.rules[1].Properties.Action.ActionType = to.Ptr(d.ActionTypeAllow) },
		"wrong-order":            func(f *dnsFixture) { f.rules[1].Properties.Priority = to.Ptr[int32](100) },
		"foreign-list":           func(f *dnsFixture) { f.rules[0].Properties.DNSResolverDomainLists[0].ID = to.Ptr("foreign") },
		"extra-list": func(f *dnsFixture) {
			f.rules[0].Properties.DNSResolverDomainLists = append(f.rules[0].Properties.DNSResolverDomainLists, &d.SubResource{})
		},
		"extra-rule":        func(f *dnsFixture) { f.rules = append(f.rules, clone(f.rules[0])) },
		"duplicate-rule":    func(f *dnsFixture) { f.rules[1] = clone(f.rules[0]) },
		"foreign-policy":    func(f *dnsFixture) { f.attached[0].ID = to.Ptr("foreign") },
		"multiple-policies": func(f *dnsFixture) { f.attached = append(f.attached, d.SubResource{}) },
		"controller-link": func(f *dnsFixture) {
			f.links[0].Properties.VirtualNetwork.ID = to.Ptr(f.settings.Firewall.Perimeter.Topology.ControllerVNet)
		},
		"extra-link":     func(f *dnsFixture) { f.links = append(f.links, clone(f.links[0])) },
		"foreign-region": func(f *dnsFixture) { f.policy.Location = to.Ptr("westus") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := dnsTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureDNSPreservesPendingMissingAndIncompletePolicyState(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "policy-pending", "list-pending", "rule-pending", "link-pending", "no-rules", "no-links", "not-associated", "bulk-without-domains", "cancelled", "incomplete-vnet-inventory", "paged", "cycle", "foreign-page", "foreign-parent", "endless"} {
		t.Run(mode, func(t *testing.T) {
			c, _, f, g := dnsTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "policy-pending":
				f.policy.Properties.ProvisioningState = to.Ptr(d.ProvisioningStateUpdating)
			case "list-pending":
				f.allow.Properties.ProvisioningState = to.Ptr(d.ProvisioningStateUpdating)
			case "rule-pending":
				f.rules[0].Properties.ProvisioningState = to.Ptr(d.ProvisioningStateUpdating)
			case "link-pending":
				f.links[0].Properties.ProvisioningState = to.Ptr(d.ProvisioningStateUpdating)
			case "no-rules":
				f.rules = nil
			case "no-links":
				f.links = nil
			case "not-associated":
				f.attached = nil
			case "bulk-without-domains":
				f.allow.Properties.Domains = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := g.Check(ctx, c)
			if mode == "paged" {
				if err != nil || f.rulePages != 2 {
					t.Fatal("safe complete paging rejected", err, f.rulePages)
				}
				return
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "cycle" || mode == "foreign-page" || mode == "foreign-parent" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, err, expected)
			if f.reads > 7 || f.rulePages > 2 || f.linkPages > 1 {
				t.Fatal("DNS retries or bounds escaped", f.reads, f.rulePages, f.linkPages)
			}
		})
	}
}
