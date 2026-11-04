package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type perimeterFixture struct {
	t        *testing.T
	mu       sync.Mutex
	settings PerimeterSettings
	group    n.SecurityGroup
	routes   n.RouteTable
	reads    int
	mode     string
}

func perimeterTest(t *testing.T) (Config, PerimeterSettings, *topologyFixture, *perimeterFixture, *PerimeterGuard) {
	t.Helper()
	c, topology, topologyFixture, guard := topologyTest(t)
	s := PerimeterSettings{Topology: topology, FirewallIP: "10.20.1.4", PrivateEndpointIPs: []string{"10.20.2.6", "10.20.2.4", "10.20.2.5"}}
	f := &perimeterFixture{t: t, settings: s}
	f.group, _ = s.SecurityGroupDefinition(c)
	f.group.ID, f.group.Name, f.group.Type = to.Ptr(s.Topology.SecurityGroup), to.Ptr("studio-workers"), to.Ptr("Microsoft.Network/networkSecurityGroups")
	f.group.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.group.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(s.Topology.WorkerSubnet)}}
	f.group.Properties.DefaultSecurityRules = []*n.SecurityRule{{Name: to.Ptr("AllowInternetOutBound"), Properties: &n.SecurityRulePropertiesFormat{Access: to.Ptr(n.SecurityRuleAccessAllow), Priority: to.Ptr[int32](65001)}}}
	for _, rule := range f.group.Properties.SecurityRules {
		rule.ID = to.Ptr(s.Topology.SecurityGroup + "/securityRules/" + value(rule.Name))
		rule.Type = to.Ptr("Microsoft.Network/networkSecurityGroups/securityRules")
		rule.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
		rule.Properties.Description = to.Ptr("operator description")
	}
	f.routes, _ = s.RouteTableDefinition(c)
	f.routes.ID, f.routes.Name, f.routes.Type = to.Ptr(s.Topology.RouteTable), to.Ptr("studio-workers"), to.Ptr("Microsoft.Network/routeTables")
	f.routes.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	f.routes.Properties.Subnets = []*n.Subnet{{ID: to.Ptr(s.Topology.WorkerSubnet)}}
	route := f.routes.Properties.Routes[0]
	route.ID, route.Type = to.Ptr(s.Topology.RouteTable+"/routes/worker-egress"), to.Ptr("Microsoft.Network/routeTables/routes")
	route.Properties.ProvisioningState = to.Ptr(n.ProvisioningStateSucceeded)
	reader, err := NewSDKPerimeterReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	perimeter, err := NewPerimeterGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, topologyFixture, f, perimeter
}

func (f *perimeterFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2025-03-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("perimeter SDK escaped its read-only resource boundary")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	var object any
	switch r.URL.Path {
	case f.settings.Topology.SecurityGroup:
		object = f.group
	case f.settings.Topology.RouteTable:
		object = f.routes
	default:
		f.t.Error("perimeter reader selected a foreign resource")
		return deadlineResponse(r, 404, nil), nil
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
	response := deadlineResponse(r, 200, object)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}

func (f *perimeterFixture) rule(name string) *n.SecurityRule {
	for _, rule := range f.group.Properties.SecurityRules {
		if strings.EqualFold(value(rule.Name), name) {
			return rule
		}
	}
	f.t.Fatal("fixture rule does not exist", name)
	return nil
}
