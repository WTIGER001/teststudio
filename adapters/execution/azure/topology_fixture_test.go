package azureworker

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type topologyFixture struct {
	t                  *testing.T
	mu                 sync.Mutex
	worker, controller a.ManagedEnvironment
	wnet, cnet         n.VirtualNetwork
	wsub, csub         n.Subnet
	reads              int
	mode               string
}

func topologyTest(t *testing.T) (Config, TopologySettings, *topologyFixture, *TopologyGuard) {
	t.Helper()
	c, _, _, _ := fixture(t, "go", "")
	s := TopologySettings{WorkerVNet: c.resourceID("Microsoft.Network", "virtualNetworks", "studio-workers"), ControllerVNet: c.resourceID("Microsoft.Network", "virtualNetworks", "studio-core"), WorkerVNetPrefix: "10.20.0.0/16", WorkerSubnetPrefix: "10.20.0.0/24", SecurityGroup: c.resourceID("Microsoft.Network", "networkSecurityGroups", "studio-workers"), RouteTable: c.resourceID("Microsoft.Network", "routeTables", "studio-workers"), InfrastructureGroup: "studio-worker-platform"}
	s.WorkerSubnet = s.WorkerVNet + "/subnets/workers"
	s.ControllerSubnet = s.ControllerVNet + "/subnets/core"
	f := &topologyFixture{t: t}
	for _, entry := range []struct {
		target            *a.ManagedEnvironment
		id, subnet, infra string
	}{{&f.worker, c.WorkerEnvironment, s.WorkerSubnet, s.InfrastructureGroup}, {&f.controller, c.ControllerEnvironment, s.ControllerSubnet, "studio-core-platform"}} {
		_, name := resourceParts(entry.id)
		*entry.target = a.ManagedEnvironment{ID: to.Ptr(entry.id), Name: to.Ptr(name), Type: to.Ptr("Microsoft.App/managedEnvironments"), Location: to.Ptr(c.Region), Properties: &a.ManagedEnvironmentProperties{ProvisioningState: to.Ptr(a.EnvironmentProvisioningStateSucceeded), InfrastructureResourceGroup: to.Ptr(entry.infra), VnetConfiguration: &a.VnetConfiguration{InfrastructureSubnetID: to.Ptr(entry.subnet), Internal: to.Ptr(true)}, PublicNetworkAccess: to.Ptr(a.PublicNetworkAccessDisabled), StaticIP: to.Ptr("10.20.0.4"), WorkloadProfiles: []*a.WorkloadProfile{{Name: to.Ptr("Consumption"), WorkloadProfileType: to.Ptr("Consumption")}}, AppLogsConfiguration: &a.AppLogsConfiguration{Destination: to.Ptr("none")}}}
	}
	for _, entry := range []struct {
		vnet                         *n.VirtualNetwork
		subnet                       *n.Subnet
		id, subid, prefix, subprefix string
	}{{&f.wnet, &f.wsub, s.WorkerVNet, s.WorkerSubnet, s.WorkerVNetPrefix, s.WorkerSubnetPrefix}, {&f.cnet, &f.csub, s.ControllerVNet, s.ControllerSubnet, "10.21.0.0/16", "10.21.0.0/24"}} {
		_, name := resourceParts(entry.id)
		*entry.subnet = n.Subnet{ID: to.Ptr(entry.subid), Name: to.Ptr(strings.Split(entry.subid, "/")[10]), Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"), Properties: &n.SubnetPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), AddressPrefix: to.Ptr(entry.subprefix), DefaultOutboundAccess: to.Ptr(false), NetworkSecurityGroup: &n.SecurityGroup{ID: to.Ptr(s.SecurityGroup)}, RouteTable: &n.RouteTable{ID: to.Ptr(s.RouteTable)}, Delegations: []*n.Delegation{{Name: to.Ptr("aca"), Properties: &n.ServiceDelegationPropertiesFormat{ServiceName: to.Ptr("Microsoft.App/environments"), ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded)}}}}}
		*entry.vnet = n.VirtualNetwork{ID: to.Ptr(entry.id), Name: to.Ptr(name), Type: to.Ptr("Microsoft.Network/virtualNetworks"), Location: to.Ptr(c.Region), Properties: &n.VirtualNetworkPropertiesFormat{ProvisioningState: to.Ptr(n.ProvisioningStateSucceeded), AddressSpace: &n.AddressSpace{AddressPrefixes: []*string{to.Ptr(entry.prefix)}}, Subnets: []*n.Subnet{entry.subnet}}}
	}
	reader, err := NewSDKTopologyReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	guard, err := NewTopologyGuard(c, s, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, guard
}

func (f *topologyFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("topology SDK escaped its read-only endpoint or identity")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-provider-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	if f.mode == "missing" {
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	}
	resources := map[string]any{value(f.worker.ID): f.worker, value(f.controller.ID): f.controller, value(f.wnet.ID): f.wnet, value(f.cnet.ID): f.cnet, value(f.wsub.ID): f.wsub, value(f.csub.ID): f.csub}
	v, ok := resources[r.URL.Path]
	if !ok {
		f.t.Error("reader requested an unconfigured resource")
		return deadlineResponse(r, 404, nil), nil
	}
	if strings.Contains(r.URL.Path, "Microsoft.App/") && r.URL.RawQuery != "api-version=2025-07-01" || strings.Contains(r.URL.Path, "Microsoft.Network/") && r.URL.RawQuery != "api-version=2025-03-01" {
		f.t.Error("SDK changed the selected management API version")
	}
	response := deadlineResponse(r, 200, v)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}

func (f *topologyFixture) detachSubnet() {
	f.wnet.Properties.Subnets[0] = to.Ptr(clone(f.wsub))
}

func setTopologyResourceGroup(t *testing.T, c Config, s TopologySettings, group string) (Config, TopologySettings) {
	t.Helper()
	data, _ := json.Marshal([]any{c, s})
	data = []byte(strings.ReplaceAll(string(data), "/resourceGroups/"+c.ResourceGroup+"/", "/resourceGroups/"+group+"/"))
	var values []json.RawMessage
	if json.Unmarshal(data, &values) != nil || json.Unmarshal(values[0], &c) != nil || json.Unmarshal(values[1], &s) != nil {
		t.Fatal("cannot adjust fixture group")
	}
	c.ResourceGroup = group
	return c, s
}
