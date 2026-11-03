package httpapi_test

import (
	"reflect"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
)

func comparison(t *testing.T, result map[string]any, name string) (bool, bool) {
	t.Helper()
	for _, raw := range result["decisions"].([]any) {
		row := raw.(map[string]any)
		if row["permission"] == "teststudio."+name {
			return row["before"].(bool), row["after"].(bool)
		}
	}
	t.Fatal("missing permission comparison", name)
	return false, false
}

func TestCustomRolePreviewRollsBackPolicyAssignmentAuditAndDirectory(t *testing.T) {
	c := roleSetup(t)
	input := access.RolePreviewInput{Policy: roleInput("preview-reader", "alpha", roleRule("catalog.read", permissions.EffectAllow, "suite/health")), User: "member", Object: "suite/health"}
	version, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	audit, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	result := accessCall(t, c, "POST", "/api/v1/roles/preview", input, "", 200)
	before, after := comparison(t, result, "catalog.read")
	if before || !after {
		t.Fatal("new policy not evaluated", result)
	}
	accessCall(t, c, "GET", "/api/v1/roles/preview-reader", nil, "", 404)
	if _, err := c.api.Auth.Engine.RoleDefinition(t.Context(), input.Policy.ID); err == nil {
		t.Fatal("preview committed native role")
	}
	assignments, err := c.api.Auth.Engine.RoleAssignmentsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: input.User})
	if err != nil || len(assignments) != 0 {
		t.Fatal("preview committed assignment", assignments, err)
	}
	latest, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || version.Revision != latest.Revision {
		t.Fatal("preview committed directory clock", latest, version, err)
	}
	currentAudit, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil || !reflect.DeepEqual(audit, currentAudit) {
		t.Fatal("preview committed audit", err)
	}
	stored := accessCall(t, c, "POST", "/api/v1/roles", input.Policy, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": input.Policy.ID, "reason": "Verify existing policy preview"}, "", 200)
	input.Policy = roleEdit(t, stored)
	input.Policy.Rules = append(input.Policy.Rules, roleRule("catalog.read", permissions.EffectDeny, "suite/health"))
	input.Revision = 1
	result = accessCall(t, c, "POST", "/api/v1/roles/preview", input, "", 200)
	before, after = comparison(t, result, "catalog.read")
	if !before || after {
		t.Fatal("proposed deny not evaluated", result)
	}
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	input.Revision = 0
	accessCall(t, c, "POST", "/api/v1/roles/preview", input, "", 409)
	input.Revision = 1
	input.Object = "suite/missing"
	accessCall(t, c, "POST", "/api/v1/roles/preview", input, "", 404)
	input.Object = "suite/health"
	object := "alpha/user/member"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.permissions.grant", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", "/api/v1/roles/preview", input, "", 403)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	if grants, err := c.api.Auth.Engine.GrantsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: input.Policy.ID}); err != nil || len(grants) != 1 {
		t.Fatal("failed preview changed grants", grants, err)
	}
}

func TestCustomRoleCapabilitiesUseActualRoleAuthority(t *testing.T) {
	c := roleSetup(t)
	stored := accessCall(t, c, "POST", "/api/v1/roles", roleInput("reader", "alpha", roleRule("catalog.read", permissions.EffectAllow, "")), "", 200)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("choices", "alpha"), "", 422)
	root := accessCall(t, c, "GET", "/api/v1/roles/reader", nil, "", 200)
	if root["canManage"] != true || root["delegatable"] != true {
		t.Fatal(root)
	}
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	view := accessCall(t, c, "GET", "/api/v1/roles/reader", nil, "", 200)
	if view["canManage"] != false || view["delegatable"] != true {
		t.Fatal("role flags inferred from project admin", view)
	}
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", TeamScope: "alpha", PermissionName: "teststudio.roles.manage", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	view = accessCall(t, c, "GET", "/api/v1/roles/reader", nil, "", 200)
	if view["canManage"] != true {
		t.Fatal(view)
	}
	object := "alpha/role/reader"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.permissions.grant", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	view = accessCall(t, c, "GET", "/api/v1/roles/reader", nil, "", 200)
	if view["canManage"] != false || view["delegatable"] != false {
		t.Fatal("object deny not reflected", view)
	}
	choices := accessCall(t, c, "GET", "/api/v1/roles/choices?project=alpha", nil, "", 200)
	if len(choices["items"].([]any)) != 1 || choices["items"].([]any)[0].(map[string]any)["delegatable"] != false {
		t.Fatal(choices)
	}
	if !reflect.DeepEqual(view["data"], stored["data"]) {
		t.Fatal("capability read rewrote metadata")
	}
}
