package azureworker

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePrivateDNSVerifiesSelectedAuthorityAndFreshInventories(t *testing.T) {
	c, s, f, g := privateDNSTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 6 {
		t.Fatal("valid selected zones rejected", err, f.reads)
	}
	s.DNS.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
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
		t.Fatal("private DNS inspection reused stale state", f.reads)
	}
	slices.Reverse(f.records[s.RegistryZone])
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("record inventory order changed authority", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("selected zone inspection became complete network authority")
	}
	c.NetworkProfile = "changed"
	reads := f.reads
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != reads {
		t.Fatal("changed configuration reached Azure")
	}
}

func TestAzurePrivateDNSRejectsAliasesWildcardRecordsAndForeignAddresses(t *testing.T) {
	cases := map[string]func(*privateDNSFixture){
		"wrong-exchange-address": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][1].Properties.ARecords[0].IPv4Address = to.Ptr(f.settings.RegistryIP)
		},
		"public-address": func(f *privateDNSFixture) {
			f.records[f.settings.RegistryZone][1].Properties.ARecords[0].IPv4Address = to.Ptr("8.8.8.8")
		},
		"extra-address": func(f *privateDNSFixture) {
			r := f.records[f.settings.ExchangeZone][1].Properties
			r.ARecords = append(r.ARecords, &p.ARecord{IPv4Address: to.Ptr("10.20.2.7")})
		},
		"wildcard": func(f *privateDNSFixture) { f.records[f.settings.ExchangeZone][1].Name = to.Ptr("*") },
		"alias": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][1].Properties.CnameRecord = &p.CnameRecord{Cname: to.Ptr("foreign.invalid")}
		},
		"ipv6": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][1].Properties.AaaaRecords = []*p.AaaaRecord{{IPv6Address: to.Ptr("fd00::4")}}
		},
		"extra-record": func(f *privateDNSFixture) {
			id := f.settings.ExchangeZone
			f.records[id] = append(f.records[id], clone(f.records[id][1]))
		},
		"duplicate-record": func(f *privateDNSFixture) { id := f.settings.ExchangeZone; f.records[id][0] = clone(f.records[id][1]) },
		"foreign-child":    func(f *privateDNSFixture) { f.records[f.settings.ExchangeZone][1].ID = to.Ptr("foreign") },
		"foreign-fqdn": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][1].Properties.Fqdn = to.Ptr("foreign.invalid.")
		},
		"auto-record": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][1].Properties.IsAutoRegistered = to.Ptr(true)
		},
		"long-positive-cache": func(f *privateDNSFixture) { f.records[f.settings.ExchangeZone][1].Properties.TTL = to.Ptr[int64](3600) },
		"long-negative-cache": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][0].Properties.SoaRecord.MinimumTTL = to.Ptr[int64](3600)
		},
		"mixed-soa": func(f *privateDNSFixture) {
			f.records[f.settings.ExchangeZone][0].Properties.CnameRecord = &p.CnameRecord{}
		},
		"wrong-zone-name": func(f *privateDNSFixture) {
			id := f.settings.ExchangeZone
			z := f.zones[id]
			z.Name = to.Ptr("foreign.invalid")
			f.zones[id] = z
		},
		"controller-link": func(f *privateDNSFixture) {
			f.links[f.settings.ExchangeZone][0].Properties.VirtualNetwork.ID = to.Ptr(f.settings.DNS.Firewall.Perimeter.Topology.ControllerVNet)
		},
		"registration": func(f *privateDNSFixture) {
			f.links[f.settings.ExchangeZone][0].Properties.RegistrationEnabled = to.Ptr(true)
		},
		"public-fallback": func(f *privateDNSFixture) {
			f.links[f.settings.RegistryZone][0].Properties.ResolutionPolicy = to.Ptr(p.ResolutionPolicyNxDomainRedirect)
		},
		"extra-link": func(f *privateDNSFixture) {
			id := f.settings.RegistryZone
			f.links[id] = append(f.links[id], clone(f.links[id][0]))
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := privateDNSTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzurePrivateDNSPreservesIncompleteAuthorityAndBoundedPaging(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "zone-pending", "link-pending", "link-propagating", "unknown-registration", "missing-link", "missing-record", "count-mismatch", "unknown-count", "cancelled", "paged", "foreign-page", "foreign-zone-page", "cycle", "endless"} {
		t.Run(mode, func(t *testing.T) {
			c, s, f, g := privateDNSTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "zone-pending":
				f.zones[s.ExchangeZone].Properties.ProvisioningState = to.Ptr(p.ProvisioningStateUpdating)
			case "link-pending":
				f.links[s.ExchangeZone][0].Properties.ProvisioningState = to.Ptr(p.ProvisioningStateUpdating)
			case "link-propagating":
				f.links[s.ExchangeZone][0].Properties.VirtualNetworkLinkState = to.Ptr(p.VirtualNetworkLinkStateInProgress)
			case "unknown-registration":
				f.links[s.ExchangeZone][0].Properties.RegistrationEnabled = nil
			case "missing-link":
				f.links[s.ExchangeZone] = nil
			case "missing-record":
				f.records[s.ExchangeZone] = f.records[s.ExchangeZone][:1]
			case "count-mismatch":
				f.zones[s.RegistryZone].Properties.NumberOfRecordSets = to.Ptr[int64](4)
			case "unknown-count":
				f.zones[s.RegistryZone].Properties.NumberOfVirtualNetworkLinks = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := g.Check(ctx, c)
			if mode == "paged" {
				if err != nil || f.pages[s.ExchangeZone+"/ALL"] != 2 {
					t.Fatal("complete safe inventory rejected", err)
				}
				return
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "foreign-page" || mode == "foreign-zone-page" || mode == "cycle" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, err, expected)
			if f.reads > 8 || f.pages[s.ExchangeZone+"/ALL"] > 3 {
				t.Fatal("inventory retried or escaped bounds", f.reads)
			}
		})
	}
}
