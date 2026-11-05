package azureworker

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
)

type registryFixture struct {
	t           *testing.T
	mu          sync.Mutex
	settings    RegistrySettings
	registry    a.Registry
	connections []*a.PrivateEndpointConnection
	links       []*a.PrivateLinkResource
	replicas    []*a.Replication
	reads       int
	pages       map[string]int
	mode        string
}

func registryServiceTest(t *testing.T, relayName ...string) (Config, RegistrySettings, *registryFixture, *RegistryGuard) {
	t.Helper()
	c, endpoints, _, guard := endpointTest(t)
	s := RegistrySettings{Endpoints: endpoints, PreparationEndpoint: c.resourceID("Microsoft.Network", "privateEndpoints", "preparation-registry")}
	if len(relayName) > 1 {
		t.Fatal("multiple relay consumers supplied")
	}
	if len(relayName) == 1 {
		s.RelayEndpoint = c.resourceID("Microsoft.Network", "privateEndpoints", relayName[0])
	}
	id := s.registry()
	_, name := resourceParts(id)
	f := &registryFixture{t: t, settings: s, pages: map[string]int{}}
	f.registry = a.Registry{ID: to.Ptr(id), Name: to.Ptr(name), Type: to.Ptr("Microsoft.ContainerRegistry/registries"), Location: to.Ptr("westus2"), SKU: &a.SKU{Name: to.Ptr(a.SKUNamePremium), Tier: to.Ptr(a.SKUTierPremium)}, Properties: &a.RegistryProperties{ProvisioningState: to.Ptr(a.ProvisioningStateSucceeded), LoginServer: to.Ptr(c.Registry), DataEndpointHostNames: []*string{to.Ptr(name + ".westus2.data.azurecr.io")}, AdminUserEnabled: to.Ptr(false), AnonymousPullEnabled: to.Ptr(false), DataEndpointEnabled: to.Ptr(true), NetworkRuleBypassAllowedForTasks: to.Ptr(false), NetworkRuleBypassOptions: to.Ptr(a.NetworkRuleBypassOptionsNone), PublicNetworkAccess: to.Ptr(a.PublicNetworkAccessDisabled), RoleAssignmentMode: to.Ptr(a.RoleAssignmentModeLegacyRegistryPermissions), NetworkRuleSet: &a.NetworkRuleSet{DefaultAction: to.Ptr(a.DefaultActionDeny)}, Policies: &a.Policies{ExportPolicy: &a.ExportPolicy{Status: to.Ptr(a.ExportPolicyStatusDisabled)}, AzureADAuthenticationAsArmPolicy: &a.AzureADAuthenticationAsArmPolicy{Status: to.Ptr(a.AzureADAuthenticationAsArmPolicyStatusEnabled)}}}}
	for i, endpoint := range s.consumers() {
		name := []string{"worker.approval", "preparation.approval", "relay.approval"}[i]
		f.connections = append(f.connections, &a.PrivateEndpointConnection{ID: to.Ptr(s.connections() + "/" + name), Name: to.Ptr(name), Type: to.Ptr("Microsoft.ContainerRegistry/registries/privateEndpointConnections"), Properties: &a.PrivateEndpointConnectionProperties{ProvisioningState: to.Ptr(a.ProvisioningStateSucceeded), PrivateEndpoint: &a.PrivateEndpoint{ID: to.Ptr(endpoint)}, PrivateLinkServiceConnectionState: &a.PrivateLinkServiceConnectionState{Status: to.Ptr(a.ConnectionStatusApproved), ActionsRequired: to.Ptr(a.ActionsRequiredNone)}}})
	}
	f.registry.Properties.PrivateEndpointConnections = clone(f.connections)
	f.links = []*a.PrivateLinkResource{{ID: to.Ptr(s.links() + "/registry"), Name: to.Ptr("registry"), Type: to.Ptr("Microsoft.ContainerRegistry/registries/privateLinkResources"), Properties: &a.PrivateLinkResourceProperties{GroupID: to.Ptr("registry"), RequiredMembers: []*string{to.Ptr("registry"), to.Ptr("registry_data_westus2")}, RequiredZoneNames: []*string{to.Ptr("privatelink.azurecr.io")}}}}
	reader, err := NewSDKRegistryReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	result, err := NewRegistryGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, result
}

func (f *registryFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-11-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("registry SDK widened read authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	switch f.mode {
	case "unavailable":
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-registry-canary"}}), nil
	case "missing":
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	case "redirect":
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	path := r.URL.Path
	var body any
	switch path {
	case f.settings.registry():
		body = f.registry
	case f.settings.links():
		body = map[string]any{"value": f.links}
	case f.settings.replicas():
		body = map[string]any{"value": f.replicas}
	case f.settings.connections():
		f.pages[path]++
		page := map[string]any{"value": f.connections}
		switch f.mode {
		case "paged", "cycle":
			if r.URL.Query().Get("$skipToken") == "" {
				page["value"] = f.connections[:1]
				page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-11-01&%24skipToken=one"
			} else {
				page["value"] = f.connections[1:]
				if f.mode == "cycle" {
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-11-01&%24skipToken=one"
				}
			}
		case "foreign-page":
			page["nextLink"] = "https://foreign.invalid/private?api-version=2025-11-01&%24skipToken=one"
		case "foreign-list":
			page["nextLink"] = "https://management.azure.com" + f.settings.replicas() + "?api-version=2025-11-01&%24skipToken=one"
		case "endless":
			page["value"] = []any{}
			page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-11-01&%24skipToken=" + strconv.Itoa(f.pages[path])
		}
		body = page
	default:
		f.t.Error("unowned registry resource requested")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, body)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}
