package httpapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/httpapi"
)

func providerInput(id string, disabled bool) access.ProviderInput {
	return access.ProviderInput{ID: id, Name: "Example provider", Issuer: "https://identity.example.test", ClientID: "teststudio", Disabled: &disabled, AutoProvision: true, Scopes: []string{"openid", "profile", "email"}, GroupMappings: map[string]string{}, MembershipTTLSeconds: 1800, Reason: "Configure an independent identity provider"}
}

func TestProviderAdministrationKeepsOwnershipRevisionsAndCredentialValuesPrivate(t *testing.T) {
	c := fixture(t, func(s *httpapi.Server) { s.OIDC.SecretRefs = []string{"env:TESTSTUDIO_OIDC_SECRET"} })
	base := "/api/v1/identity-providers"
	input := providerInput("example", true)
	accessCall(t, c, "POST", base, input, "", 401)
	c.signIn()
	t.Setenv("TESTSTUDIO_OIDC_SECRET", "private-credential-must-never-appear")
	input.SecretRef = "env:TESTSTUDIO_OIDC_SECRET"
	created := accessCall(t, c, "POST", base, input, "", 200)
	if created["revision"] != float64(1) || created["data"].(map[string]any)["canManage"] != true {
		t.Fatal(created)
	}
	if mappings, ok := created["data"].(map[string]any)["groupMappings"].(map[string]any); !ok || len(mappings) != 0 {
		t.Fatal("empty mappings did not match the typed response contract", created)
	}
	if created["data"].(map[string]any)["callbackUrl"] != c.origin+"/auth/callback/example" {
		t.Fatal("callback did not use the installation origin", created)
	}
	accessCall(t, c, "POST", base, input, "", 409)
	accessCall(t, c, "PUT", base+"/example", input, "0", 422)
	accessCall(t, c, "PUT", base+"/other", input, "1", 422)
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "alpha", "name": "Alpha"}, "", 200)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "mapped-reviewers", "name": "Mapped reviewers", "projectId": "alpha", "managedBy": "oidc:example"}, "", 200)
	group := accessCall(t, c, "GET", "/api/v1/groups/mapped-reviewers", nil, "", 200)
	if group["data"].(map[string]any)["managedBy"] != "oidc:example" {
		t.Fatal(group)
	}
	accessCall(t, c, "PUT", "/api/v1/groups/mapped-reviewers", map[string]string{"id": "mapped-reviewers", "name": "Changed reviewers", "projectId": "alpha", "managedBy": "oidc:other"}, "1", 422)
	accessCall(t, c, "PUT", "/api/v1/groups/mapped-reviewers/members/admin", map[string]any{}, "0", 422)
	input.GroupClaim = "groups"
	input.GroupMappings = map[string]string{"provider-reviewers": "mapped-reviewers"}
	accessCall(t, c, "PUT", base+"/example", input, "1", 200)
	disabled := false
	input.Disabled = &disabled
	accessCall(t, c, "PUT", base+"/example", input, "2", 200)
	accessCall(t, c, "PUT", base+"/example", input, "2", 409)
	listed := accessCall(t, c, "GET", base, nil, "", 200)
	if listed["canCreate"] != true || len(listed["credentialRefs"].([]any)) != 1 {
		t.Fatal(listed)
	}
	public := accessCall(t, c, "GET", "/auth/providers", nil, "", 200)
	encoded, _ := json.Marshal(public)
	for _, private := range []string{"issuer", "clientId", "secretRef", "groupMappings", "TESTSTUDIO_OIDC_SECRET"} {
		if strings.Contains(string(encoded), private) {
			t.Fatal("public provider choices exposed private configuration", private)
		}
	}
	audit, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(map[string]any{"listed": listed, "audit": audit})
	if strings.Contains(string(encoded), "private-credential-must-never-appear") {
		t.Fatal("provider administration read a credential value")
	}
}

func TestProviderAdministrationRejectsInvalidMappingsAndUnapprovedCredentialsWithoutWrites(t *testing.T) {
	c := fixture(t)
	c.signIn()
	base := "/api/v1/identity-providers"
	input := providerInput("example", true)
	accessCall(t, c, "POST", base, input, "", 200)
	defaultLifetime := providerInput("default-lifetime", true)
	defaultLifetime.MembershipTTLSeconds = 0
	withDefault := accessCall(t, c, "POST", base, defaultLifetime, "", 200)
	if withDefault["data"].(map[string]any)["membershipTtlSeconds"] != float64(0) {
		t.Fatal("default lifetime was omitted from the typed response", withDefault)
	}
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "local", "name": "Local group"}, "", 200)
	for _, mode := range []string{"state", "credential", "claim", "scope", "foreign-group", "source", "blank-reason", "logout-credentials", "logout-scheme", "logout-fragment"} {
		t.Run(mode, func(t *testing.T) {
			value := providerInput("example", true)
			switch mode {
			case "state":
				value.Disabled = nil
			case "credential":
				value.SecretRef = "env:UNAPPROVED_PROCESS_SECRET"
			case "claim":
				value.GroupMappings = map[string]string{"external": "local"}
			case "scope":
				value.Scopes = []string{"openid", "openid"}
			case "foreign-group":
				value.GroupClaim = "groups"
				value.GroupMappings = map[string]string{"external": "local"}
			case "source":
				value.SecretRef = "inline:secret-value"
			case "logout-credentials":
				value.LogoutEndpoint = "https://identity.example.test/logout?id_token_hint=private"
			case "logout-scheme":
				value.LogoutEndpoint = "http://identity.example.test/logout"
			case "logout-fragment":
				value.LogoutEndpoint = "https://identity.example.test/logout#fragment"
			case "blank-reason":
				value.Reason = " "
			}
			before, err := c.api.Auth.Directory.Revision(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			accessCall(t, c, "PUT", base+"/example", value, "1", 422)
			after, err := c.api.Auth.Directory.Revision(t.Context())
			if err != nil || after.Revision != before.Revision {
				t.Fatal("rejected configuration changed directory state", err)
			}
			stored := accessCall(t, c, "GET", base+"/example", nil, "", 200)
			if stored["revision"] != float64(1) {
				t.Fatal("rejected configuration changed revision", stored)
			}
		})
	}
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "unknown", "name": "Unknown", "managedBy": "oidc:missing"}, "", 404)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "bad-source", "name": "Bad", "managedBy": "arbitrary:source"}, "", 422)
}

func TestProviderAdministrationRespectsInstallationAndObjectAuthority(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/identity-providers"
	for _, id := range []string{"a-private", "z-visible"} {
		accessCall(t, c, "POST", base, providerInput(id, true), "", 200)
	}
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "GET", base, nil, "", 403)
	accessCall(t, c, "POST", base, providerInput("forbidden", true), "", 403)
	c.signIn()
	for _, grant := range []permissions.Grant{
		{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio.providers.read", TeamScope: "*", Effect: permissions.EffectAllow},
	} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), grant); err != nil {
			t.Fatal(err)
		}
	}
	deny := "/api/v1/permissions/grants"
	accessCall(t, c, "POST", deny, directGrant("user", "member", "providers.read", "provider/a-private", permissions.EffectDeny), "", 200)
	assignmentSignInAs(t, c, "member")
	page := accessCall(t, c, "GET", base+"?limit=1", nil, "", 200)
	if len(page["items"].([]any)) != 0 || page["partial"] != true || page["canCreate"] != false {
		t.Fatal(page)
	}
	accessCall(t, c, "GET", base+"/a-private", nil, "", 403)
	row := accessCall(t, c, "GET", base+"/z-visible", nil, "", 200)
	if row["data"].(map[string]any)["canManage"] != false {
		t.Fatal(row)
	}
	accessCall(t, c, "PUT", base+"/z-visible", providerInput("z-visible", true), "1", 403)
}

func TestProviderMappingChangesRequireActualGroupMembershipAuthority(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/identity-providers/example"
	value := providerInput("example", true)
	accessCall(t, c, "POST", "/api/v1/identity-providers", value, "", 200)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "owned", "name": "Owned", "projectId": "alpha", "managedBy": "oidc:example"}, "", 200)
	value.GroupClaim = "groups"
	value.GroupMappings = map[string]string{"external": "owned"}
	accessCall(t, c, "PUT", base, value, "1", 200)
	for _, action := range []string{"providers.read", "providers.manage", "groups.read"} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio." + action, TeamScope: "*", Effect: permissions.EffectAllow}); err != nil {
			t.Fatal(err)
		}
	}
	assignmentSignInAs(t, c, "member")
	read := accessCall(t, c, "GET", base, nil, "", 200)
	if read["data"].(map[string]any)["canManage"] != false {
		t.Fatal("provider authority bypassed group management", read)
	}
	accessCall(t, c, "PUT", base, value, "2", 403)
	c.signIn()
	object := "alpha/group/owned"
	for _, action := range []string{"groups.manage", "members.manage"} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio." + action, TeamScope: "alpha", ObjectScope: &object, Effect: permissions.EffectAllow}); err != nil {
			t.Fatal(err)
		}
	}
	assignmentSignInAs(t, c, "member")
	read = accessCall(t, c, "GET", base, nil, "", 200)
	if read["data"].(map[string]any)["canManage"] != true {
		t.Fatal("exact destination group authority was not honored", read)
	}
	accessCall(t, c, "PUT", base, value, "2", 200)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "member", "members.manage", "group/owned", permissions.EffectDeny), "", 200)
	assignmentSignInAs(t, c, "member")
	read = accessCall(t, c, "GET", base, nil, "", 200)
	if read["data"].(map[string]any)["canManage"] != false {
		t.Fatal("an exact group membership deny was ignored", read)
	}
	accessCall(t, c, "PUT", base, value, "3", 403)
}

func TestProviderEnablingRequiresActiveMappedGroupsAndProjects(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/identity-providers/example"
	value := providerInput("example", true)
	accessCall(t, c, "POST", "/api/v1/identity-providers", value, "", 200)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "owned", "name": "Owned", "projectId": "alpha", "managedBy": "oidc:example"}, "", 200)
	value.GroupClaim = "groups"
	value.GroupMappings = map[string]string{"external": "owned"}
	accessCall(t, c, "PUT", base, value, "1", 200)
	accessCall(t, c, "POST", "/api/v1/groups/owned/archive", map[string]string{"reason": "Verify disabled mappings remain inspectable"}, "1", 200)
	disabled := false
	value.Disabled = &disabled
	accessCall(t, c, "PUT", base, value, "2", 422)
	disabled = true
	accessCall(t, c, "PUT", base, value, "2", 200)
	accessCall(t, c, "POST", "/api/v1/groups/owned/restore", map[string]string{"reason": "Restore the mapped destination"}, "2", 200)
	disabled = false
	accessCall(t, c, "PUT", base, value, "3", 200)
	project := accessCall(t, c, "GET", "/api/v1/projects/alpha", nil, "", 200)
	accessCall(t, c, "PUT", "/api/v1/projects/alpha", map[string]any{"id": "alpha", "name": "Alpha", "archived": true}, assignmentRevision(project), 200)
	accessCall(t, c, "PUT", base, value, "4", 422)
}
