package azureworker

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureReadIdentityBindsFreshPrincipalAndSubscriptionGrant(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	s.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if i == 2 {
			slices.Reverse(f.role.Properties.Permissions[0].Actions)
		}
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*4 || f.protection.reads != i*5 || f.protection.inventory.reads != i*4 {
			t.Fatal("observer grant rejected or parent repeated", err, f.reads, f.protection.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("selected grant became complete effective read authority")
	}
	f.assignment.Properties.PrincipalID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 12 {
		t.Fatal("changed principal reused cached grant")
	}
	f.protection.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 12 {
		t.Fatal("unverified protection reached observer grant reads")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
}

func TestAzureReadIdentityRejectsScopeConditionsAndAdditionalAuthority(t *testing.T) {
	for name, mutate := range map[string]func(*readIdentityFixture){
		"foreign-client": func(f *readIdentityFixture) {
			f.identity.Properties.ClientID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
		},
		"foreign-tenant": func(f *readIdentityFixture) {
			f.identity.Properties.TenantID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
		},
		"group-principal": func(f *readIdentityFixture) { f.assignment.Properties.PrincipalType = to.Ptr(a.PrincipalTypeGroup) },
		"resource-group-scope": func(f *readIdentityFixture) {
			f.assignment.Properties.Scope = to.Ptr(observerScope(f.protection.inventory.worker) + "/resourceGroups/studio")
		},
		"other-role":           func(f *readIdentityFixture) { f.assignment.Properties.RoleDefinitionID = to.Ptr("other-role-canary") },
		"assignment-condition": func(f *readIdentityFixture) { f.assignment.Properties.Condition = to.Ptr("private-condition-canary") },
		"delegated-identity": func(f *readIdentityFixture) {
			f.assignment.Properties.DelegatedManagedIdentityResourceID = to.Ptr(f.settings.Identity)
		},
		"builtin-role": func(f *readIdentityFixture) { f.role.Properties.RoleType = to.Ptr("BuiltInRole") },
		"broad-action": func(f *readIdentityFixture) { f.role.Properties.Permissions[0].Actions = []*string{to.Ptr("*")} },
		"extra-action": func(f *readIdentityFixture) {
			f.role.Properties.Permissions[0].Actions = append(f.role.Properties.Permissions[0].Actions, to.Ptr("Microsoft.Network/virtualNetworks/join/action"))
		},
		"read-exclusion": func(f *readIdentityFixture) {
			f.role.Properties.Permissions[0].NotActions = []*string{to.Ptr("Microsoft.Network/privateDnsZones/read")}
		},
		"data-access": func(f *readIdentityFixture) { f.role.Properties.Permissions[0].DataActions = []*string{to.Ptr("*")} },
		"role-condition": func(f *readIdentityFixture) {
			f.role.Properties.Permissions[0].Condition = to.Ptr("private-condition-canary")
		},
		"assignable-root": func(f *readIdentityFixture) { f.role.Properties.AssignableScopes = []*string{to.Ptr("/")} },
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := readIdentityTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureReadIdentityCredentialFactoryPinsVerifiedClientWithoutTokenRequest(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	var clients []string
	g.credential = func(id string) (azcore.TokenCredential, error) {
		clients = append(clients, id)
		return fixtureCredential{}, nil
	}
	s.ClientID = "aaaaaaaa-1111-2222-3333-444444444444"
	credential, err := g.Credential(t.Context(), c)
	if err != nil || credential == nil || len(clients) != 1 || clients[0] != f.settings.ClientID || f.reads != 4 {
		t.Fatal("credential did not bind verified identity in one inspection", err, clients, f.reads)
	}
	f.identity.Properties.PrincipalID = nil
	credential, err = g.Credential(t.Context(), c)
	requireDeadlineError(t, err, hostedexecution.ErrUnavailable)
	if credential != nil || len(clients) != 1 {
		t.Fatal("incomplete identity reached credential factory")
	}
	f.identity.Properties.PrincipalID = to.Ptr(f.settings.PrincipalID)
	g.credential = func(string) (azcore.TokenCredential, error) {
		return nil, errors.New("private-credential-factory-canary")
	}
	_, err = g.Credential(t.Context(), c)
	requireDeadlineError(t, err, hostedexecution.ErrUnavailable)
}

func TestAzureReadIdentitySDKPreservesUnknownMetadataAndExactResourceScope(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	reader := g.reader.(*SDKReadIdentityReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.URL.Path = s.Role },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$filter=principalId" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Identity+"?api-version="+readIdentityVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Identity(t.Context(), c.PullIdentity)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	_, err = reader.Assignment(t.Context(), s.Role)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected observer request reached transport")
	}
	for _, mode := range []string{"unavailable", "redirect", "empty-role"} {
		f.mode = mode
		requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	}
}

func TestAzureReadIdentitySettingsAndDefinitionsFreezeGrantBindings(t *testing.T) {
	c, s, _, g := readIdentityTest(t)
	for _, mutate := range []func(*ReadIdentitySettings){
		func(s *ReadIdentitySettings) { s.Identity = c.PullIdentity },
		func(s *ReadIdentitySettings) {
			s.Identity = s.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.RuntimeIdentity
		},
		func(s *ReadIdentitySettings) { s.ClientID = s.PrincipalID },
		func(s *ReadIdentitySettings) { s.TenantID = allDenyPrincipals },
		func(s *ReadIdentitySettings) {
			s.Assignment = observerScope(c) + "/resourceGroups/studio/providers/Microsoft.Authorization/roleAssignments/44444444-dddd-4444-dddd-444444444444"
		},
		func(s *ReadIdentitySettings) {
			s.Role = strings.Replace(s.Role, c.Subscription, "99999999-2222-3333-4444-555555555555", 1)
		},
	} {
		changed := s
		mutate(&changed)
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.Protection.WorkerAssignment = strings.Replace(s.Protection.WorkerAssignment, "aaaaaaaa-1111-2222-3333-444444444444", "ffffffff-1111-2222-3333-444444444444", 1)
	_, err := NewReadIdentityGuard(c, changed, g.protection, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKReadIdentityReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	first, _ := s.RoleDefinition(c)
	first.Properties.Permissions[0].Actions[0] = to.Ptr("*")
	second, err := s.RoleDefinition(c)
	if err != nil || !networkDenyActions(second.Properties.Permissions[0].Actions, "*/read", networkReadAssociationAction) {
		t.Fatal("fresh read role retained caller mutation", err)
	}
}
