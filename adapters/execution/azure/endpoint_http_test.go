package azureworker

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureEndpointsSDKRestrictsResourceKindsAndTransportAuthority(t *testing.T) {
	c, s, f, g := endpointTest(t)
	reader := g.reader.(*SDKEndpointReader)
	if _, err := reader.Endpoint(t.Context(), s.RegistryNIC); err != hostedexecution.ErrOwnership {
		t.Fatal("NIC substituted for endpoint", err)
	}
	if _, err := reader.Interface(t.Context(), s.ExchangeEndpoint); err != hostedexecution.ErrOwnership {
		t.Fatal("endpoint substituted for NIC", err)
	}
	if _, err := reader.Subnet(t.Context(), s.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerSubnet); err != hostedexecution.ErrOwnership {
		t.Fatal("worker subnet substituted for endpoint subnet", err)
	}
	if _, err := reader.SecurityGroup(t.Context(), s.SecurityGroup+"-other"); err != hostedexecution.ErrOwnership {
		t.Fatal("unowned security group requested", err)
	}
	for name, mutate := range map[string]func(*http.Request){
		"create":          func(r *http.Request) { r.Method = "PUT" },
		"delete":          func(r *http.Request) { r.Method = "DELETE" },
		"post":            func(r *http.Request) { r.Method = "POST" },
		"list":            func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.RegistryEndpoint, "/worker-registry") },
		"child":           func(r *http.Request) { r.URL.Path = s.RegistryNIC + "/ipConfigurations/unowned" },
		"foreign-origin":  func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"foreign-service": func(r *http.Request) { r.URL.Path = s.Registry },
		"controller":      func(r *http.Request) { r.URL.Path = c.ControllerEnvironment },
		"expand":          func(r *http.Request) { r.URL.RawQuery += "&%24expand=all" },
		"paging":          func(r *http.Request) { r.URL.RawQuery += "&%24skipToken=one" },
		"wrong-version":   func(r *http.Request) { r.URL.RawQuery = "api-version=2025-07-01" },
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.RegistryEndpoint+"?api-version=2025-03-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if f.reads != 0 {
		t.Fatal("invalid scope reached transport", f.reads)
	}
	if err := g.Check(t.Context(), c); err != nil || f.reads != 6 {
		t.Fatal("valid SDK observation failed", err, f.reads)
	}
}
