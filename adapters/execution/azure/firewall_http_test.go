package azureworker

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureFirewallSDKScopesPagingAndEscapedGroups(t *testing.T) {
	c, s, _, _ := firewallTest(t)
	original := c.ResourceGroup
	c, s.Perimeter.Topology = setTopologyResourceGroup(t, c, s.Perimeter.Topology, "studio(group)")
	s.Firewall = strings.Replace(s.Firewall, "/resourceGroups/"+original+"/", "/resourceGroups/studio(group)/", 1)
	s.Policy = strings.Replace(s.Policy, "/resourceGroups/"+original+"/", "/resourceGroups/studio(group)/", 1)
	s.PublicIP = strings.Replace(s.PublicIP, "/resourceGroups/"+original+"/", "/resourceGroups/studio(group)/", 1)
	calls := 0
	resources := map[string]bool{s.Firewall: true, s.Policy: true, s.PublicIP: true, s.subnet(): true, s.Policy + "/ruleCollectionGroups": true}
	wire := transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if !resources[r.URL.Path] || r.URL.Host != "management.azure.com" || r.Method != "GET" || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
			t.Fatal("escaped resource read widened its scope")
		}
		if r.URL.Path == s.Policy+"/ruleCollectionGroups" {
			return deadlineResponse(r, 200, map[string]any{"value": []any{}}), nil
		}
		return deadlineResponse(r, 200, map[string]any{"id": r.URL.Path}), nil
	})
	reader, err := NewSDKFirewallReader(c, s, fixtureCredential{}, wire)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if _, err := reader.Firewall(t.Context(), s.Firewall); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Policy(t.Context(), s.Policy); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Subnet(t.Context(), s.subnet()); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.PublicIP(t.Context(), s.PublicIP); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.RuleGroups(t.Context(), s.Policy); err != nil {
		t.Fatal(err)
	}
	if calls != 5 {
		t.Fatal("SDK retried reads", calls)
	}
	if _, err := reader.Firewall(t.Context(), s.Policy); err != hostedexecution.ErrOwnership {
		t.Fatal("policy substituted for firewall")
	}
	path := s.Policy + "/ruleCollectionGroups"
	for _, raw := range []string{"api-version=2025-03-01&%24skipToken=one", "%24skipToken=one&api-version=2025-03-01"} {
		u, _ := url.Parse("https://management.azure.com" + path + "?" + raw)
		if !reader.http.queryAllowed(u, "2025-03-01") {
			t.Fatal("safe opaque paging token rejected")
		}
	}
	for _, raw := range []string{"api-version=2025-03-01&%24skipToken=", "api-version=2025-03-01&%24skipToken=one&extra=true", "api-version=2025-03-01&%24skipToken=one&%24skipToken=two", "api-version=2025-03-01&%24skipToken=%0D%0A", "api-version=2025-03-01&%24skipToken=" + strings.Repeat("a", 2049)} {
		u, _ := url.Parse("https://management.azure.com" + path + "?" + raw)
		if reader.http.queryAllowed(u, "2025-03-01") {
			t.Fatal("unsafe paging query accepted")
		}
	}
	for _, h := range []*armReadHTTP{reader.http, newTopologyHTTP(c, s.Perimeter.Topology, wire), newARMReadHTTP(c, map[string]string{s.Perimeter.Topology.SecurityGroup: "2025-03-01"}, wire)} {
		selected := s.Firewall
		if h != reader.http {
			for id := range h.resources {
				selected = id
				break
			}
		}
		u, _ := url.Parse("https://management.azure.com" + selected + "?api-version=" + h.resources[selected] + "&%24skipToken=one")
		if h.queryAllowed(u, h.resources[selected]) {
			t.Fatal("paging escaped the exact rule-group collection")
		}
	}
	if calls != 5 {
		t.Fatal("rejected SDK scope reached transport")
	}
}

func TestAzureFirewallHTTPRejectsForeignAndMutableRequests(t *testing.T) {
	c, s, _, g := firewallTest(t)
	reader := g.reader.(*SDKFirewallReader)
	calls := 0
	reader.http.base = transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return deadlineResponse(r, 200, nil), nil })
	cases := map[string]func(*http.Request){
		"mutation":          func(r *http.Request) { r.Method = "PUT" },
		"foreign-origin":    func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"unowned-list":      func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.Firewall, "/studio-firewall") },
		"child-action":      func(r *http.Request) { r.URL.Path += "/listKeys" },
		"foreign-policy":    func(r *http.Request) { r.URL.Path = s.Policy + "-foreign" },
		"alternate-version": func(r *http.Request) { r.URL.RawQuery = "api-version=2026-01-01" },
		"api-resource":      func(r *http.Request) { r.URL.Path = c.ControllerEnvironment; r.URL.RawQuery = "api-version=2025-07-01" },
		"userinfo":          func(r *http.Request) { r.URL.User = url.User("foreign") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Firewall+"?api-version=2025-03-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if calls != 0 {
		t.Fatal("unsafe firewall inspection reached transport")
	}
}
