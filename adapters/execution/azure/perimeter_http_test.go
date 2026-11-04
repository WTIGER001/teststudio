package azureworker

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePerimeterSDKPinsTwoResourcesAndSupportsEscapedGroups(t *testing.T) {
	for _, group := range []string{"studio", "studio(group)"} {
		t.Run(group, func(t *testing.T) {
			c, s, _, _, _ := perimeterTest(t)
			c, s.Topology = setTopologyResourceGroup(t, c, s.Topology, group)
			calls := 0
			wire := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.RawQuery != "api-version=2025-03-01" || r.URL.Path != s.Topology.SecurityGroup && r.URL.Path != s.Topology.RouteTable || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
					t.Fatal("perimeter read escaped its exact SDK scope")
				}
				return deadlineResponse(r, 200, map[string]any{"id": r.URL.Path}), nil
			})
			reader, err := NewSDKPerimeterReader(c, s, fixtureCredential{}, wire)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if v, err := reader.SecurityGroup(t.Context(), s.Topology.SecurityGroup); err != nil || value(v.ID) != s.Topology.SecurityGroup {
				t.Fatal(err)
			}
			if v, err := reader.RouteTable(t.Context(), s.Topology.RouteTable); err != nil || value(v.ID) != s.Topology.RouteTable {
				t.Fatal(err)
			}
			if _, err := reader.SecurityGroup(t.Context(), s.Topology.RouteTable); err != hostedexecution.ErrOwnership {
				t.Fatal("resource kinds were substituted")
			}
			if _, err := reader.RouteTable(t.Context(), s.Topology.SecurityGroup); err != hostedexecution.ErrOwnership {
				t.Fatal("resource kinds were substituted")
			}
			if calls != 2 {
				t.Fatal("SDK repeated or widened reads", calls)
			}
		})
	}
}

func TestAzurePerimeterHTTPRejectsUnownedOperationsBeforeTransport(t *testing.T) {
	c, s, _, _, g := perimeterTest(t)
	reader := g.reader.(*SDKPerimeterReader)
	calls := 0
	reader.http.base = transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return deadlineResponse(r, 200, nil), nil })
	cases := map[string]func(*http.Request){
		"mutation":          func(r *http.Request) { r.Method = "PUT" },
		"child-action":      func(r *http.Request) { r.URL.Path += "/securityRules/unowned" },
		"list":              func(r *http.Request) { r.URL.Path = strings.TrimSuffix(r.URL.Path, "/studio-workers") },
		"foreign-host":      func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"userinfo":          func(r *http.Request) { r.URL.User = url.User("unowned") },
		"extra-query":       func(r *http.Request) { r.URL.RawQuery += "&expand=secret" },
		"alternate-version": func(r *http.Request) { r.URL.RawQuery = "api-version=2026-01-01" },
		"topology-resource": func(r *http.Request) { r.URL.Path = c.WorkerEnvironment; r.URL.RawQuery = "api-version=2025-07-01" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Topology.SecurityGroup+"?api-version=2025-03-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if calls != 0 {
		t.Fatal("unowned operation reached transport")
	}
}
