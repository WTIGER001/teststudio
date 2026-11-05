package azureworker

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
)

type readIdentityFixture struct {
	t                  *testing.T
	mu                 sync.Mutex
	settings           ReadIdentitySettings
	protection         *networkDenyFixture
	identity           m.Identity
	assignment         a.RoleAssignment
	role               a.RoleDefinition
	identityProtection a.DenyAssignment
	stacks             map[string]d.DeploymentStack
	reads              int
	mode               string
	federation         string
}

func readIdentityTest(t *testing.T) (Config, ReadIdentitySettings, *readIdentityFixture, *ReadIdentityGuard) {
	t.Helper()
	c, protection, previous, prior := networkDenyTest(t)
	s := ReadIdentitySettings{Protection: protection, Identity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "worker-observer"), ClientID: "11111111-aaaa-1111-aaaa-111111111111", PrincipalID: "22222222-bbbb-2222-bbbb-222222222222", TenantID: "33333333-cccc-3333-cccc-333333333333", Assignment: observerScope(c) + "/providers/Microsoft.Authorization/roleAssignments/44444444-dddd-4444-dddd-444444444444", Role: observerScope(c) + "/providers/Microsoft.Authorization/roleDefinitions/55555555-eeee-5555-eeee-555555555555"}
	s.ProtectionAssignment = s.Identity + "/providers/Microsoft.Authorization/denyAssignments/66666666-ffff-6666-ffff-666666666666"
	s.NetworkProtectionStack = observerScope(c) + "/providers/Microsoft.Resources/deploymentStacks/studio-network-protection"
	s.IdentityProtectionStack = c.resourceID("Microsoft.Resources", "deploymentStacks", "studio-observer-protection")
	f := &readIdentityFixture{t: t, settings: s, protection: previous}
	f.identityProtection = clone(previous.assignments[protection.WorkerAssignment])
	f.identityProtection.ID = to.Ptr(s.ProtectionAssignment)
	f.identityProtection.Name = to.Ptr("66666666-ffff-6666-ffff-666666666666")
	f.identityProtection.Properties.Scope = to.Ptr(s.Identity)
	f.identityProtection.Properties.Permissions[0].NotActions = []*string{to.Ptr("*/read"), to.Ptr("Microsoft.Authorization/locks/delete")}
	f.stacks = protectionStackFixtures(c, s)
	f.identity = m.Identity{ID: to.Ptr(s.Identity), Name: to.Ptr("worker-observer"), Type: to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities"), Location: to.Ptr(c.Region), Properties: &m.UserAssignedIdentityProperties{ClientID: to.Ptr(s.ClientID), PrincipalID: to.Ptr(s.PrincipalID), TenantID: to.Ptr(s.TenantID)}}
	f.assignment, _ = s.AssignmentDefinition(c)
	f.assignment.ID, f.assignment.Name, f.assignment.Type = to.Ptr(s.Assignment), to.Ptr("44444444-dddd-4444-dddd-444444444444"), to.Ptr("Microsoft.Authorization/roleAssignments")
	f.assignment.Properties.Scope = to.Ptr(observerScope(c))
	f.role, _ = s.RoleDefinition(c)
	f.role.ID, f.role.Name, f.role.Type = to.Ptr(s.Role), to.Ptr("55555555-eeee-5555-eeee-555555555555"), to.Ptr("Microsoft.Authorization/roleDefinitions")
	reader, err := NewSDKReadIdentityReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewReadIdentityGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *readIdentityFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	version := readAssignmentVersion
	var body any
	switch r.URL.Path {
	case f.settings.Identity:
		version, body = readIdentityVersion, f.identity
	case f.settings.Assignment:
		body = f.assignment
	case f.settings.Role:
		version, body = readRoleVersion, f.role
	case f.settings.Identity + "/federatedIdentityCredentials":
		version, body = readIdentityVersion, map[string]any{"value": []any{}}
	case f.settings.ProtectionAssignment:
		version, body = networkDenyVersion, f.identityProtection
	case f.settings.NetworkProtectionStack, f.settings.IdentityProtectionStack:
		version, body = protectionStackVersion, f.stacks[r.URL.Path]
	default:
		f.t.Error("unselected observer identity resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version="+version || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("observer identity reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-observer-identity-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	if f.mode == "empty-role" && strings.EqualFold(r.URL.Path, f.settings.Role) {
		body = a.RoleDefinition{ID: f.role.ID, Name: f.role.Name, Type: f.role.Type}
	}
	if r.URL.Path == f.settings.Identity+"/federatedIdentityCredentials" {
		switch f.federation {
		case "present":
			body = map[string]any{"value": []any{map[string]any{"name": "private-federation-name", "properties": map[string]any{"issuer": "https://private-issuer.invalid", "subject": "private-subject-canary", "audiences": []string{"api://AzureADTokenExchange"}}}}}
		case "missing-value":
			body = map[string]any{}
		case "null-value":
			body = map[string]any{"value": nil}
		case "null-item":
			body = map[string]any{"value": []any{nil}}
		case "incomplete":
			body = map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + r.URL.Path + "?api-version=" + readIdentityVersion + "&%24skiptoken=private-next-canary"}
		case "foreign-page":
			body = map[string]any{"value": []any{}, "nextLink": "https://foreign.invalid/private-next-canary"}
		case "forbidden":
			return deadlineResponse(r, 403, map[string]any{"error": map[string]any{"code": "AuthorizationFailed", "message": "private-federation-canary"}}), nil
		}
	}
	return deadlineResponse(r, 200, body), nil
}
