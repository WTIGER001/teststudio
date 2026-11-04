package azureworker

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureTopologySDKRestrictsReadScopeAndSupportsEscapedGroups(t *testing.T) {
	for _, group := range []string{"studio", "studio(group)"} {
		t.Run(group, func(t *testing.T) {
			c, s, _, _ := topologyTest(t)
			c, s = setTopologyResourceGroup(t, c, s, group)
			calls := 0
			resources := map[string]string{c.WorkerEnvironment: "2025-07-01", c.ControllerEnvironment: "2025-07-01", s.WorkerVNet: "2025-03-01", s.ControllerVNet: "2025-03-01", s.WorkerSubnet: "2025-03-01", s.ControllerSubnet: "2025-03-01"}
			wire := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				version, ok := resources[r.URL.Path]
				if !ok || r.URL.RawQuery != "api-version="+version || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
					t.Fatal("SDK read escaped its six configured resources")
				}
				return deadlineResponse(r, 200, map[string]any{"id": r.URL.Path}), nil
			})
			reader, err := NewSDKTopologyReader(c, s, fixtureCredential{}, wire)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			for _, id := range []string{c.WorkerEnvironment, c.ControllerEnvironment} {
				v, err := reader.Environment(t.Context(), id)
				if err != nil || value(v.ID) != id {
					t.Fatal("environment read failed", err)
				}
			}
			for _, id := range []string{s.WorkerVNet, s.ControllerVNet} {
				v, err := reader.VirtualNetwork(t.Context(), id)
				if err != nil || value(v.ID) != id {
					t.Fatal("vnet read failed", err)
				}
			}
			for _, id := range []string{s.WorkerSubnet, s.ControllerSubnet} {
				v, err := reader.Subnet(t.Context(), id)
				if err != nil || value(v.ID) != id {
					t.Fatal("subnet read failed", err)
				}
			}
			if calls != 6 {
				t.Fatal("SDK repeated scoped reads", calls)
			}
			if _, err := reader.Environment(t.Context(), s.WorkerVNet); err != hostedexecution.ErrOwnership {
				t.Fatal("wrong resource kind was read", err)
			}
			if _, err := reader.VirtualNetwork(t.Context(), s.WorkerSubnet); err != hostedexecution.ErrOwnership {
				t.Fatal("child substituted for vnet", err)
			}
			if _, err := reader.Subnet(t.Context(), s.WorkerVNet); err != hostedexecution.ErrOwnership {
				t.Fatal("vnet substituted for child", err)
			}
			if calls != 6 {
				t.Fatal("rejected resource reached transport")
			}
		})
	}
}

func TestAzureTopologyHTTPRejectsMutationsAndForeignReads(t *testing.T) {
	c, s, _, _ := topologyTest(t)
	calls := 0
	wire := transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return deadlineResponse(r, 200, nil), nil })
	h := newTopologyHTTP(c, s, wire)
	cases := map[string]func(*http.Request){
		"host":              func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"insecure":          func(r *http.Request) { r.URL.Scheme = "http" },
		"mutation":          func(r *http.Request) { r.Method = "PUT" },
		"list":              func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.WorkerSubnet, "/workers") },
		"child-action":      func(r *http.Request) { r.URL.Path += "/listSecrets" },
		"wrong-version":     func(r *http.Request) { r.URL.RawQuery = "api-version=2025-07-01" },
		"extra-query":       func(r *http.Request) { r.URL.RawQuery += "&expand=secrets" },
		"userinfo":          func(r *http.Request) { r.URL.User = url.UserPassword("private", "private") },
		"encoded-separator": func(r *http.Request) { r.URL.RawPath = strings.Replace(r.URL.Path, "/subnets/", "/subnets%2F", 1) },
		"foreign-subnet":    func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.WorkerSubnet, "/workers") + "/foreign" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.WorkerSubnet+"?api-version=2025-03-01", nil)
			mutate(r)
			_, err := h.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if calls != 0 {
		t.Fatal("unsafe network inspection reached transport", calls)
	}
}
