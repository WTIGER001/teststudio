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
	return w
}

func (w *placementWorld) Do(r *http.Request) (*http.Response, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	version := placementVersions(w.s)[r.URL.Path]
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || version == "" || r.URL.RawQuery != "api-version="+version || r.Header.Get("Authorization") != "Bearer placement-metadata-canary" {
		w.t.Error("placement request escaped its selected identity or resource")
		return response(403, `{}`), nil
	}
	w.calls[r.URL.Path]++
	if w.status != 0 {
		return response(w.status, `{"error":{"code":"Unavailable","message":"sensitive-provider-canary"}}`), nil
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
