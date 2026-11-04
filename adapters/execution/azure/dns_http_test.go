package azureworker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureDNSSDKPinsReadInventoryActionAndEscapedResources(t *testing.T) {
	for _, group := range []string{"studio", "studio(group)"} {
		t.Run(group, func(t *testing.T) {
			c, s, _, _ := dnsTest(t)
			data, _ := json.Marshal(struct {
				Config   Config
				Settings DNSSettings
			}{c, s})
			data = bytes.ReplaceAll(data, []byte("/resourceGroups/"+c.ResourceGroup+"/"), []byte("/resourceGroups/"+group+"/"))
			var configuration struct {
				Config   Config
				Settings DNSSettings
			}
			if json.Unmarshal(data, &configuration) != nil {
				t.Fatal("cannot adjust fixture groups")
			}
			c, s = configuration.Config, configuration.Settings
			c.ResourceGroup = group
			calls := 0
			resources := map[string]string{s.Policy: "GET", s.AllowList: "GET", s.BlockList: "GET", s.rulePath(): "GET", s.linkPath(): "GET", s.vnetPolicyPath(): "POST"}
			wire := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if resources[r.URL.Path] != r.Method || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2025-05-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" || r.Body != nil && r.Body != http.NoBody {
					t.Fatal("DNS SDK request widened read authority")
				}
				if r.URL.Path == s.rulePath() || r.URL.Path == s.linkPath() || r.URL.Path == s.vnetPolicyPath() {
					return deadlineResponse(r, 200, map[string]any{"value": []any{}}), nil
				}
				return deadlineResponse(r, 200, map[string]any{"id": r.URL.Path}), nil
			})
			reader, err := NewSDKDNSReader(c, s, fixtureCredential{}, wire)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if _, err := reader.Policy(t.Context(), s.Policy); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.DomainList(t.Context(), s.AllowList); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.DomainList(t.Context(), s.BlockList); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.Rules(t.Context(), s.Policy); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.Links(t.Context(), s.Policy); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.VNetPolicies(t.Context(), s.Firewall.Perimeter.Topology.WorkerVNet); err != nil {
				t.Fatal(err)
			}
			if calls != 6 {
				t.Fatal("SDK repeated or changed reads", calls)
			}
			if _, err := reader.Policy(t.Context(), s.AllowList); err != hostedexecution.ErrOwnership {
				t.Fatal("list substituted for policy")
			}
			if _, err := reader.DomainList(t.Context(), s.Policy); err != hostedexecution.ErrOwnership {
				t.Fatal("policy substituted for list")
			}
			if _, err := reader.VNetPolicies(t.Context(), s.Firewall.Perimeter.Topology.ControllerVNet); err != hostedexecution.ErrOwnership {
				t.Fatal("controller inventory was read")
			}
			if calls != 6 {
				t.Fatal("invalid typed scope reached transport")
			}
		})
	}
}

func TestAzureDNSHTTPRejectsWritesAndUnownedAssociationActions(t *testing.T) {
	c, s, _, g := dnsTest(t)
	reader := g.reader.(*SDKDNSReader)
	calls := 0
	reader.http.base = transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return deadlineResponse(r, 200, nil), nil })
	cases := map[string]func(*http.Request){
		"write":      func(r *http.Request) { r.Method = "PUT" },
		"delete":     func(r *http.Request) { r.Method = "DELETE" },
		"wrong-post": func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.AllowList },
		"wrong-get":  func(r *http.Request) { r.Method = "GET"; r.URL.Path = s.vnetPolicyPath() },
		"body-on-read-action": func(r *http.Request) {
			r.Method = "POST"
			r.URL.Path = s.vnetPolicyPath()
			r.Body = http.NoBody
			r.ContentLength = 1
		},
		"foreign-host": func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"bulk":         func(r *http.Request) { r.URL.Path = s.AllowList + "/bulk" },
		"unowned-rule": func(r *http.Request) { r.URL.Path = s.rulePath() + "/foreign" },
		"controller-action": func(r *http.Request) {
			r.Method = "POST"
			r.URL.Path = s.Firewall.Perimeter.Topology.ControllerVNet + "/listDnsResolverPolicies"
		},
		"extra-query":   func(r *http.Request) { r.URL.RawQuery += "&extra=secret" },
		"invalid-user":  func(r *http.Request) { r.URL.User = url.UserPassword("private", "private") },
		"wrong-version": func(r *http.Request) { r.URL.RawQuery = "api-version=2025-03-01" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Policy+"?api-version=2025-05-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	for _, h := range []*armReadHTTP{reader.http, newTopologyHTTP(c, s.Firewall.Perimeter.Topology, reader.http.base)} {
		r, _ := http.NewRequestWithContext(t.Context(), "POST", "https://management.azure.com"+c.WorkerEnvironment+"?api-version=2025-07-01", strings.NewReader("private"))
		_, err := h.Do(r)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	if calls != 0 {
		t.Fatal("unsafe DNS read reached transport", calls)
	}
}
