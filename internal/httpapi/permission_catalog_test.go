package httpapi_test

import (
	"net/url"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func permissionChoice(t *testing.T, page map[string]any, id string) map[string]any {
	t.Helper()
	for _, item := range page["items"].([]any) {
		value := item.(map[string]any)
		if value["id"] == "teststudio."+id {
			return value
		}
	}
	t.Fatal("missing permission choice", id)
	return nil
}

func TestCustomRolePermissionCatalogDescribesExactServerGrantability(t *testing.T) {
	c := roleSetup(t)
	installation := accessCall(t, c, "GET", "/api/v1/permissions", nil, "", 200)
	if len(installation["items"].([]any)) != len(authorization.PermissionNames()) {
		t.Fatal("permission registration and description differ", installation)
	}
	for _, raw := range installation["items"].([]any) {
		item := raw.(map[string]any)
		description := item["description"].(string)
		if len(description) < 6 || strings.Contains(description, "%!") || item["canGrant"] != true {
			t.Fatal("invalid registered description or installer grantability", item)
		}
	}
	project := accessCall(t, c, "GET", "/api/v1/permissions?project=alpha", nil, "", 200)
	for _, raw := range project["items"].([]any) {
		item := raw.(map[string]any)
		if item["installationOnly"] != false {
			t.Fatal("installation action exposed as project choice", item)
		}
	}
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	page := accessCall(t, c, "GET", "/api/v1/permissions?project=alpha", nil, "", 200)
	if permissionChoice(t, page, "catalog.read")["canGrant"] != true || permissionChoice(t, page, "catalog.update")["canGrant"] != false {
		t.Fatal("grantability inferred from administration instead of current actions", page)
	}
	accessCall(t, c, "GET", "/api/v1/permissions", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/permissions?project=beta", nil, "", 403)
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", TeamScope: "alpha", PermissionName: "teststudio.roles.manage", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	object := "alpha/suite/health"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.permissions.grant", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	page = accessCall(t, c, "GET", "/api/v1/permissions?project=alpha&object="+url.QueryEscape("suite/health"), nil, "", 200)
	for _, raw := range page["items"].([]any) {
		if raw.(map[string]any)["canGrant"] != false {
			t.Fatal("object delegation deny not reflected", raw)
		}
	}
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("blocked-object", "alpha", roleRule("catalog.read", permissions.EffectAllow, "suite/health")), "", 403)
	accessCall(t, c, "GET", "/api/v1/permissions?project=alpha&object=suite/*", nil, "", 422)
}
