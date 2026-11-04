package azureworker

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dnsresolver/armdnsresolver"
)

type dnsFixture struct {
	t                    *testing.T
	mu                   sync.Mutex
	settings             DNSSettings
	policy               d.Policy
	allow, block         d.DomainList
	rules                []d.DNSSecurityRule
	links                []d.PolicyVirtualNetworkLink
	attached             []d.SubResource
	reads                int
	rulePages, linkPages int
	mode                 string
}

func dnsTest(t *testing.T) (Config, DNSSettings, *dnsFixture, *DNSGuard) {
	t.Helper()
	c, firewall, _, guard := firewallTest(t)
	s := DNSSettings{RegistryDataRegion: "westus2", Firewall: firewall, Policy: c.resourceID("Microsoft.Network", "dnsResolverPolicies", "studio-dns"), AllowList: c.resourceID("Microsoft.Network", "dnsResolverDomainLists", "studio-dns-allow"), BlockList: c.resourceID("Microsoft.Network", "dnsResolverDomainLists", "studio-dns-block")}
	f := &dnsFixture{t: t, settings: s}
	f.policy, _ = s.PolicyDefinition(c)
	f.policy.ID, f.policy.Name, f.policy.Type = to.Ptr(s.Policy), to.Ptr("studio-dns"), to.Ptr("Microsoft.Network/dnsResolverPolicies")
	f.policy.Properties = &d.PolicyProperties{ProvisioningState: to.Ptr(d.ProvisioningStateSucceeded)}
	for _, item := range []struct {
		allow    bool
		target   *d.DomainList
		id, name string
	}{{true, &f.allow, s.AllowList, "studio-dns-allow"}, {false, &f.block, s.BlockList, "studio-dns-block"}} {
		*item.target, _ = s.DomainListDefinition(c, item.allow)
		item.target.ID, item.target.Name, item.target.Type = to.Ptr(item.id), to.Ptr(item.name), to.Ptr("Microsoft.Network/dnsResolverDomainLists")
		item.target.Properties.ProvisioningState = to.Ptr(d.ProvisioningStateSucceeded)
	}
	for _, allow := range []bool{true, false} {
		rule, _ := s.RuleDefinition(c, allow)
		name := "worker-default-block"
		if allow {
			name = "worker-allow-approved"
		}
		rule.ID, rule.Name, rule.Type = to.Ptr(s.rulePath()+"/"+name), to.Ptr(name), to.Ptr("Microsoft.Network/dnsResolverPolicies/dnsSecurityRules")
		rule.Properties.ProvisioningState = to.Ptr(d.ProvisioningStateSucceeded)
		f.rules = append(f.rules, rule)
	}
	link, _ := s.LinkDefinition(c)
	link.ID, link.Name, link.Type = to.Ptr(s.linkPath()+"/worker-network"), to.Ptr("worker-network"), to.Ptr("Microsoft.Network/dnsResolverPolicies/virtualNetworkLinks")
	link.Properties.ProvisioningState = to.Ptr(d.ProvisioningStateSucceeded)
	f.links = []d.PolicyVirtualNetworkLink{link}
	f.attached = []d.SubResource{{ID: to.Ptr(s.Policy)}}
	reader, err := NewSDKDNSReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	dns, err := NewDNSGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, dns
}

func (f *dnsFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	method := "GET"
	if r.URL.Path == f.settings.vnetPolicyPath() {
		method = "POST"
	}
	if r.Method != method || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-05-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("DNS SDK escaped its explicit read scope")
	}
	if r.Body != nil && r.Body != http.NoBody {
		f.t.Error("read-only inventory gained request input")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-provider-canary"}}), nil
	}
	if f.mode == "missing" {
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var object any
	switch r.URL.Path {
	case f.settings.Policy:
		object = f.policy
	case f.settings.AllowList:
		object = f.allow
	case f.settings.BlockList:
		object = f.block
	case f.settings.rulePath():
		f.rulePages++
		page := map[string]any{"value": f.rules}
		switch f.mode {
		case "paged", "cycle":
			if r.URL.Query().Get("$skipToken") == "" {
				page["value"] = f.rules[:1]
				page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-05-01&%24skipToken=one"
			} else {
				page["value"] = f.rules[1:]
				if f.mode == "cycle" {
					page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-05-01&%24skipToken=one"
				}
			}
		case "foreign-page":
			page["nextLink"] = "https://foreign.invalid/private?api-version=2025-05-01&%24skipToken=one"
		case "foreign-parent":
			page["nextLink"] = "https://management.azure.com" + f.settings.Policy + "-foreign/dnsSecurityRules?api-version=2025-05-01&%24skipToken=one"
		case "endless":
			page["value"] = []any{}
			page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-05-01&%24skipToken=" + strconv.Itoa(f.rulePages)
		}
		object = page
	case f.settings.linkPath():
		f.linkPages++
		object = map[string]any{"value": f.links}
	case f.settings.vnetPolicyPath():
		page := map[string]any{"value": f.attached}
		if f.mode == "incomplete-vnet-inventory" {
			page["nextLink"] = "https://management.azure.com" + r.URL.Path + "?api-version=2025-05-01&%24skipToken=one"
		}
		object = page
	default:
		f.t.Error("DNS reader selected an unconfigured resource")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, object)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}
