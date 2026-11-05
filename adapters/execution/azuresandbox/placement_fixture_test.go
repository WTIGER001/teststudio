package azuresandbox

import (
	"context"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type placementWorld struct {
	t         *testing.T
	s         Settings
	mu        sync.Mutex
	resources map[string]map[string]any
	calls     map[string]int
	status    int
	pages     map[string]map[string]any
}

func newPlacementWorld(t *testing.T, s Settings) *placementWorld {
	t.Helper()
	w := &placementWorld{t: t, s: s, resources: map[string]map[string]any{}, calls: map[string]int{}}
	put := func(id, kind, region string, properties map[string]any) map[string]any {
		r := map[string]any{"id": id, "type": kind, "name": id[strings.LastIndex(id, "/")+1:], "properties": properties}
		if region != "" {
			r["location"] = region
		}
		w.resources[id] = r
		return r
	}
	c, p := s.Worker, s.Placement
	g := put(c.groupID(), "Microsoft.App/sandboxGroups", c.Region, map[string]any{"provisioningState": "Succeeded", "defaultCpu": "2", "defaultMemory": "4Gi", "defaultDisk": "40Gi", "maxSandboxCount": 16, "defaultTimeoutSeconds": 7200})
	g["identity"] = map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{s.PullIdentity: map[string]string{"clientId": p.PullClientID, "principalId": p.PullPrincipalID}}}
	put(c.connectionID(), "Microsoft.App/sandboxGroups/vnetConnections", c.Region, map[string]any{"provisioningState": "Succeeded", "subnetId": p.Subnet})
	put(p.Subnet, "Microsoft.Network/virtualNetworks/subnets", "", map[string]any{"provisioningState": "Succeeded", "addressPrefix": p.SubnetPrefix, "networkSecurityGroup": map[string]string{"id": p.SecurityGroup}, "routeTable": map[string]string{"id": p.RouteTable}, "defaultOutboundAccess": false, "delegations": []any{map[string]any{"name": "sandbox-delegation", "properties": map[string]string{"serviceName": "Microsoft.App/environments"}}}})
	put(s.PullIdentity, "Microsoft.ManagedIdentity/userAssignedIdentities", c.Region, map[string]any{"clientId": p.PullClientID, "principalId": p.PullPrincipalID, "tenantId": s.TenantID})
	reg := put(p.Registry, "Microsoft.ContainerRegistry/registries", c.Region, map[string]any{"provisioningState": "Succeeded", "loginServer": c.Registry, "publicNetworkAccess": "Disabled", "networkRuleBypassOptions": "None", "roleAssignmentMode": "LegacyRegistryPermissions", "adminUserEnabled": false, "anonymousPullEnabled": false, "dataEndpointEnabled": true})
	reg["sku"] = map[string]string{"name": "Premium"}
	put(p.PullAssignment, "Microsoft.Authorization/roleAssignments", "", map[string]any{"principalId": p.PullPrincipalID, "principalType": "ServicePrincipal", "scope": p.Registry, "roleDefinitionId": "/subscriptions/" + c.Subscription + pullRole})
	put("/subscriptions/"+c.Subscription+pullRole, "Microsoft.Authorization/roleDefinitions", "", map[string]any{"roleName": "AcrPull", "type": "BuiltInRole", "assignableScopes": []string{"/"}, "permissions": []any{map[string]any{"actions": []string{"Microsoft.ContainerRegistry/registries/pull/read"}}}})
	identities := []struct{ resource, client, principal string }{{s.Authority.Control.Resource, s.ControlClientID, s.Authority.Control.PrincipalID}, {s.Authority.Registry.Resource, s.RegistryClientID, s.Authority.Registry.PrincipalID}, {s.Authority.Metadata.Resource, s.MetadataClientID, s.Authority.Metadata.PrincipalID}, {s.Timer.StopIdentity, s.Authority.StopClientID, s.Authority.StopPrincipalID}}
	for _, identity := range identities {
		put(identity.resource, "Microsoft.ManagedIdentity/userAssignedIdentities", c.Region, map[string]any{"clientId": identity.client, "principalId": identity.principal, "tenantId": s.TenantID})
	}
	definitions, err := s.AuthorityDefinitions()
	if err != nil {
		t.Fatal(err)
	}
	for id, value := range definitions {
		w.resources[id] = value.(map[string]any)
	}
	for id, value := range s.Placement.networkDefinitions(c) {
		r := value.(map[string]any)
		kind, children := "Microsoft.Network/networkSecurityGroups", "securityRules"
		if id == p.RouteTable {
			kind, children = "Microsoft.Network/routeTables", "routes"
		}
		props := r["properties"].(map[string]any)
		props["provisioningState"] = "Succeeded"
		props["subnets"] = []any{map[string]any{"id": p.Subnet}}
		for _, raw := range props[children].([]any) {
			entry := raw.(map[string]any)
			entry["properties"].(map[string]any)["provisioningState"] = "Succeeded"
		}
		put(id, kind, c.Region, props)
	}
	w.addPrivate()
	w.addTopology()
	w.addResolver()
	w.addServices()
	w.addBackendEndpoint()
	w.addRelayNetwork()
	w.addHostRegistry()
	w.addHostDatabase()
	w.addHostNetwork()
	w.addRegistryService()
	w.addHost()
	w.addServiceWorkloads()
	return w
}

func (w *placementWorld) Do(r *http.Request) (*http.Response, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	version := placementVersions(w.s)[r.URL.Path]
	if version == "" {
		version = authorityVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = networkVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = privateVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = topologyVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = resolverVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = serviceVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = backendEndpointVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = relayNetworkVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = registryServiceVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = serviceWorkloadVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = hostVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = hostRegistryVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = hostNetworkVersions(w.s)[r.URL.Path]
	}
	if version == "" {
		version = hostDatabaseVersions(w.s)[r.URL.Path]
	}
	if version == "" && r.URL.Path == "/subscriptions/"+w.s.Worker.Subscription+"/providers/Microsoft.Network/privateDnsZones" {
		version = "2024-06-01"
	}
	if version == "" && strings.HasSuffix(r.URL.Path, "/virtualNetworkLinks") {
		zone := w.resources[strings.TrimSuffix(r.URL.Path, "/virtualNetworkLinks")]
		if zone != nil && zone["type"] == "Microsoft.Network/privateDnsZones" {
			version = "2024-06-01"
		}
	}
	method := "GET"
	if r.URL.Path == w.s.Host.virtualNetwork()+"/listDnsResolverPolicies" || r.URL.Path == w.s.Placement.virtualNetwork()+"/listDnsResolverPolicies" || r.URL.Path == w.s.Placement.Services.RelayNetwork.virtualNetwork(w.s)+"/listDnsResolverPolicies" {
		method = "POST"
	}
	if r.Method != method || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || version == "" || r.URL.RawQuery != "api-version="+version || r.Header.Get("Authorization") != "Bearer placement-metadata-canary" {
		w.t.Error("placement request escaped its selected identity or resource")
		return response(403, `{}`), nil
	}
	w.calls[r.URL.Path]++
	if w.status != 0 {
		return response(w.status, `{"error":{"code":"Unavailable","message":"sensitive-provider-canary"}}`), nil
	}
	if page, ok := w.pages[r.URL.Path]; ok {
		return jsonResponse(page), nil
	}
	return jsonResponse(w.resources[r.URL.Path]), nil
}

type placementProtection struct {
	denied   atomic.Bool
	creates  atomic.Int32
	verifies atomic.Int32
	imports  atomic.Int32
}

func (p *placementProtection) BeforeCreate(context.Context, Config) error {
	p.creates.Add(1)
	return nil
}
func (p *placementProtection) VerifyInstance(context.Context, Config, Snapshot) error {
	p.verifies.Add(1)
	if p.denied.Load() {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
func (p *placementProtection) BeforeImport(context.Context, Config, string) error {
	p.imports.Add(1)
	return nil
}
