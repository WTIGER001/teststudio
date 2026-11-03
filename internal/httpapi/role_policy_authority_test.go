package httpapi_test

import (
	"reflect"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
)

func TestCustomRoleAPIDelegationCannotWidenScopeOrInheritedAuthority(t *testing.T) {
	c := roleSetup(t)
	for _, group := range []struct{ id, project string }{{"shared-policy-group", ""}, {"owned-policy-group", "alpha"}, {"foreign-policy-group", "beta"}} {
		accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": group.id, "name": group.id, "projectId": group.project}, "", 200)
	}
	privileged := roleInput("privileged-editor", "alpha", roleRule("catalog.update", permissions.EffectAllow, ""))
	accessCall(t, c, "POST", "/api/v1/roles", privileged, "", 200)
	foreign := roleInput("foreign-policy", "beta", roleRule("catalog.read", permissions.EffectAllow, ""))
	accessCall(t, c, "POST", "/api/v1/roles", foreign, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	reader := roleInput("reader", "alpha", roleRule("catalog.read", permissions.EffectAllow, ""))
	accessCall(t, c, "POST", "/api/v1/roles", reader, "", 403)
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", PermissionName: "teststudio.roles.manage", TeamScope: "alpha", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	stored := accessCall(t, c, "POST", "/api/v1/roles", reader, "", 200)
	for _, group := range []struct {
		id     string
		status int
	}{{"owned-policy-group", 200}, {"shared-policy-group", 403}, {"foreign-policy-group", 403}} {
		accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "group", "principalId": group.id, "roleId": reader.ID, "reason": "Enforce stored group ownership for custom roles"}, "", group.status)
	}
	widen := roleEdit(t, stored)
	widen.Project = "beta"
	accessCall(t, c, "PUT", "/api/v1/roles/reader", widen, assignmentRevision(stored), 422)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("installation-policy", "", roleRule("system.manage", permissions.EffectAllow, "")), "", 403)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("another-project", "beta", roleRule("catalog.read", permissions.EffectAllow, "")), "", 403)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("unsafe-action", "alpha", roleRule("system.manage", permissions.EffectAllow, "")), "", 422)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("cannot-edit", "alpha", roleRule("catalog.update", permissions.EffectAllow, "")), "", 403)
	inherited := roleInput("inherit-privilege", "alpha")
	inherited.Inherits = []string{privileged.ID}
	accessCall(t, c, "POST", "/api/v1/roles", inherited, "", 403)
	inherited.Inherits = []string{foreign.ID}
	accessCall(t, c, "POST", "/api/v1/roles", inherited, "", 422)
	accessCall(t, c, "GET", "/api/v1/roles/"+foreign.ID, nil, "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": reader.ID, "reason": "Delegate an owned readable action"}, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": privileged.ID, "reason": "Cannot delegate an unheld action"}, "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": foreign.ID, "reason": "Cannot move a custom role"}, "", 404)
	parent := roleInput("parent", "alpha")
	parent.Inherits = []string{reader.ID}
	accessCall(t, c, "POST", "/api/v1/roles", parent, "", 200)
	object := "alpha/role/parent"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", PermissionName: "teststudio.permissions.grant", TeamScope: "alpha", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	edit := roleEdit(t, stored)
	edit.Name = "Indirect edit must fail"
	accessCall(t, c, "PUT", "/api/v1/roles/reader", edit, assignmentRevision(stored), 403)
	current := accessCall(t, c, "GET", "/api/v1/roles/reader", nil, "", 200)
	if current["data"].(map[string]any)["name"] == edit.Name {
		t.Fatal("ancestor deny bypassed through a child update", current)
	}
}

func TestCustomRoleAPIRejectsInvalidObjectsAndServerOwnedFields(t *testing.T) {
	c := roleSetup(t)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "foreign", "name": "Foreign", "projectId": "beta"}, "", 200)
	for _, object := range []string{"*", "suite/*", "alpha/suite/health", "suite/../health", "user/admin"} {
		accessCall(t, c, "POST", "/api/v1/roles", roleInput("invalid-object", "alpha", roleRule("catalog.read", permissions.EffectAllow, object)), "", 422)
	}
	for _, object := range []string{"suite/missing", "group/foreign"} {
		accessCall(t, c, "POST", "/api/v1/roles", roleInput("foreign-object", "alpha", roleRule("catalog.read", permissions.EffectAllow, object)), "", 404)
	}
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("builtin.teststudio.replacement", "alpha"), "", 422)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("custom", "alpha"), "", 422)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("unregistered", "alpha", roleRule("catalog.unknown", permissions.EffectAllow, "")), "", 422)
	accessCall(t, c, "POST", "/api/v1/roles", map[string]any{"id": "raw-scope", "name": "Raw scope", "projectId": "alpha", "rules": []any{}, "inherits": []string{}, "reason": "Reject raw grant scope", "teamScope": "*"}, "", 422)
	input := roleInput("duplicate-rules", "alpha", roleRule("catalog.read", permissions.EffectAllow, ""), roleRule("catalog.read", permissions.EffectAllow, ""))
	accessCall(t, c, "POST", "/api/v1/roles", input, "", 422)
	input = roleInput("duplicate-inheritance", "alpha")
	input.Inherits = []string{"one", "one"}
	accessCall(t, c, "POST", "/api/v1/roles", input, "", 422)
	stored := accessCall(t, c, "POST", "/api/v1/roles", roleInput("permanent", "alpha"), "", 200)
	accessCall(t, c, "PUT", "/api/v1/roles/permanent", roleEdit(t, stored), "0", 422)
	accessCall(t, c, "PUT", "/api/v1/roles/other", roleEdit(t, stored), "1", 422)
	edit := roleEdit(t, stored)
	edit.Reason = " "
	accessCall(t, c, "PUT", "/api/v1/roles/permanent", edit, "1", 422)
	accessCall(t, c, "PUT", "/api/v1/projects/alpha", map[string]any{"id": "alpha", "name": "Archived", "archived": true}, "1", 200)
	accessCall(t, c, "PUT", "/api/v1/roles/permanent", roleEdit(t, stored), "1", 422)
}

func TestCustomRoleAPIProtectsFinalAdministratorAcrossInheritedPolicies(t *testing.T) {
	c := roleSetup(t)
	child := roleInput("administration-policy", "")
	for _, action := range []string{"system.manage", "users.manage", "permissions.grant", "permissions.read", "roles.manage", "roles.read", "members.manage", "members.read"} {
		child.Rules = append(child.Rules, roleRule(action, permissions.EffectAllow, ""))
	}
	stored := accessCall(t, c, "POST", "/api/v1/roles", child, "", 200)
	parent := roleInput("administrator", "")
	parent.Inherits = []string{child.ID}
	accessCall(t, c, "POST", "/api/v1/roles", parent, "", 200)
	accessCall(t, c, "POST", "/api/v1/role-assignments", map[string]string{"principalKind": "user", "principalId": "admin", "roleId": parent.ID, "reason": "Retain an explicit custom administrator"}, "", 200)
	root := "/api/v1/role-assignments/user/admin/builtin.teststudio.installation-admin"
	assignment := accessCall(t, c, "GET", root, nil, "", 200)
	accessCall(t, c, "DELETE", root, map[string]string{"reason": "Use the explicit custom role"}, assignmentRevision(assignment), 200)
	roleAllowed(t, c, "admin", "", "", "system.manage", true)
	grants, err := c.api.Auth.Engine.GetStore().ListGrants(t.Context(), permissions.GrantQuery{Principals: []permissions.PrincipalRef{{Kind: permissions.PrincipalRole, ID: child.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	version, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	edit := roleEdit(t, stored)
	edit.Disabled = true
	denied := accessCall(t, c, "PUT", "/api/v1/roles/"+child.ID, edit, assignmentRevision(stored), 422)
	if !strings.Contains(denied["message"].(string), "last usable installation administrator") {
		t.Fatal(denied)
	}
	edit = roleEdit(t, stored)
	edit.Rules = append(edit.Rules, roleRule("system.manage", permissions.EffectDeny, ""))
	accessCall(t, c, "PUT", "/api/v1/roles/"+child.ID, edit, assignmentRevision(stored), 422)
	current := accessCall(t, c, "GET", "/api/v1/roles/"+child.ID, nil, "", 200)
	if current["revision"] != stored["revision"] || current["data"].(map[string]any)["disabled"] != false {
		t.Fatal("guard did not roll back managed metadata", current)
	}
	after, err := c.api.Auth.Engine.GetStore().ListGrants(t.Context(), permissions.GrantQuery{Principals: []permissions.PrincipalRef{{Kind: permissions.PrincipalRole, ID: child.ID}}})
	if err != nil || !reflect.DeepEqual(grants.Grants, after.Grants) {
		t.Fatal("guard did not restore native grant identifiers", grants, after, err)
	}
	latest, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || latest.Revision != version.Revision {
		t.Fatal("guard advanced directory revision", latest, version, err)
	}
	roleAllowed(t, c, "admin", "", "", "system.manage", true)
}
