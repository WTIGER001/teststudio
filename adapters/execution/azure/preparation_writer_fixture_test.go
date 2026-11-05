package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

type preparationWriterFixture struct {
	t          *testing.T
	mu         sync.Mutex
	settings   PreparationWriterSettings
	identity   m.Identity
	assignment a.RoleAssignment
	role       a.RoleDefinition
	reads      int
	mode       string
}

func preparationWriterTest(t *testing.T) (Config, PreparationWriterSettings, *preparationWriterFixture, *preparationRegistryFixture, *PreparationWriterGuard) {
	t.Helper()
	c, registry, prior, parent := preparationRegistryTest(t)
	s := PreparationWriterSettings{Registry: registry, Identity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "source-writer"), ClientID: "11111111-aaaa-1111-aaaa-111111111111", PrincipalID: "22222222-bbbb-2222-bbbb-222222222222", TenantID: "33333333-cccc-3333-cccc-333333333333", Assignment: registry.Registry.registry() + "/providers/Microsoft.Authorization/roleAssignments/44444444-dddd-4444-dddd-444444444444"}
	f := &preparationWriterFixture{t: t, settings: s}
	f.identity = m.Identity{ID: to.Ptr(s.Identity), Name: to.Ptr("source-writer"), Type: to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities"), Location: to.Ptr(c.Region), Properties: &m.UserAssignedIdentityProperties{ClientID: to.Ptr(s.ClientID), PrincipalID: to.Ptr(s.PrincipalID), TenantID: to.Ptr(s.TenantID)}}
	f.assignment, _ = s.AssignmentDefinition(c)
	f.assignment.ID, f.assignment.Name, f.assignment.Type = to.Ptr(s.Assignment), to.Ptr("44444444-dddd-4444-dddd-444444444444"), to.Ptr("Microsoft.Authorization/roleAssignments")
	f.assignment.Properties.Scope = to.Ptr(registry.Registry.registry())
	f.role = a.RoleDefinition{ID: to.Ptr(observerScope(c) + preparationPushRole), Name: to.Ptr("8311e382-0749-4cb8-b61a-304f252e45ec"), Type: to.Ptr("Microsoft.Authorization/roleDefinitions"), Properties: &a.RoleDefinitionProperties{RoleName: to.Ptr("AcrPush"), RoleType: to.Ptr("BuiltInRole"), AssignableScopes: []*string{to.Ptr("/")}, Permissions: []*a.Permission{{Actions: []*string{to.Ptr("Microsoft.ContainerRegistry/registries/pull/read"), to.Ptr("Microsoft.ContainerRegistry/registries/push/write")}}}}}
	reader, err := NewSDKPreparationWriterReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewPreparationWriterGuard(c, s, parent, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, prior, g
}

func (f *preparationWriterFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	var body any
	version := readAssignmentVersion
	switch r.URL.Path {
	case f.settings.Identity:
		body, version = f.identity, readIdentityVersion
	case f.settings.Assignment:
		body = f.assignment
	default:
		if r.URL.Path != strings.TrimSuffix(strings.Split(f.settings.Assignment, "/resourceGroups/")[0], "/")+preparationPushRole {
			f.t.Error("unselected source writer resource reached transport")
		}
		body, version = f.role, readRoleVersion
	}
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version="+version || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("source writer reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"message": "private-writer-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private-writer-canary")
		return response, nil
	}
	return deadlineResponse(r, 200, body), nil
}
