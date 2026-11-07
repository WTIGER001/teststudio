package httpapi_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func builtinChange(disabled bool) map[string]any {
	return map[string]any{"disabled": disabled, "reason": "Review the installation-wide predefined role state"}
}

func TestBuiltinRoleAPIWithdrawsAndRestoresAccessAcrossProjectsAndInstances(t *testing.T) {
	c := assignmentSetup(t)
	id := "builtin.teststudio.editor"
	path := "/api/v1/predefined-roles/" + id
	for _, project := range []string{"alpha", "beta"} {
		accessCall(t, c, "POST", "/api/v1/projects/"+project+"/role-assignments", assignBody("user", "member", "editor"), "", 200)
	}
	second, err := authorization.New(t.Context(), c.api.Auth.Engine.GetStore(), &identity.Directory{Store: c.api.Store})
	if err != nil {
		t.Fatal(err)
	}
	for _, auth := range []*authorization.Service{c.api.Auth, second} {
		if err := auth.Check(t.Context(), "member", "alpha", "", "catalog.update"); err != nil {
			t.Fatal(err)
		}
	}
	before := accessCall(t, c, "GET", path, nil, "", 200)
	if before["revision"] != float64(1) || before["data"].(map[string]any)["canManage"] != true || before["updatedAt"] != nil {
		t.Fatal(before)
	}
	catalog := accessCall(t, c, "GET", "/api/v1/predefined-roles", nil, "", 200)
	if len(catalog["items"].([]any)) != 11 {
		t.Fatal(catalog)
	}
	disabled := accessCall(t, c, "PUT", path, builtinChange(true), assignmentRevision(before), 200)
	if disabled["revision"].(float64) <= before["revision"].(float64) || disabled["data"].(map[string]any)["disabled"] != true {
		t.Fatal(disabled)
	}
	for _, auth := range []*authorization.Service{c.api.Auth, second} {
		for _, project := range []string{"alpha", "beta"} {
			if err := auth.Check(t.Context(), "member", project, "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
				t.Fatal("disabled role retained access", project, err)
			}
		}
	}
	roles := accessCall(t, c, "GET", "/api/v1/roles?project=alpha", nil, "", 200)
	for _, r := range roles["items"].([]any) {
		role := r.(map[string]any)
		if role["id"] == id && (role["disabled"] != true || role["delegatable"] != false) {
			t.Fatal("stale role choice", role)
		}
	}
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "editor"), "", 422)
	accessCall(t, c, "PUT", path, builtinChange(false), assignmentRevision(before), 409)
	accessCall(t, c, "PUT", path, builtinChange(false), "0", 422)
	accessCall(t, c, "PUT", path, map[string]string{"reason": "Reject omitted state"}, assignmentRevision(disabled), 422)
	accessCall(t, c, "PUT", path, map[string]any{"disabled": false, "permissions": []string{}, "reason": "Reject code definition edits"}, assignmentRevision(disabled), 422)
	enabled := accessCall(t, c, "PUT", path, builtinChange(false), assignmentRevision(disabled), 200)
	if enabled["revision"].(float64) <= disabled["revision"].(float64) || enabled["updatedAt"] == nil {
		t.Fatal(enabled)
	}
	for _, auth := range []*authorization.Service{c.api.Auth, second} {
		if err := auth.Check(t.Context(), "member", "alpha", "", "catalog.update"); err != nil {
			t.Fatal("enabled role did not restore access", err)
		}
	}
	for _, project := range []string{"alpha", "beta"} {
		page := accessCall(t, c, "GET", "/api/v1/projects/"+project+"/role-assignments", nil, "", 200)
		if len(page["items"].([]any)) != 1 {
			t.Fatal("state change removed an assignment", page)
		}
	}
	after := accessCall(t, c, "GET", path, nil, "", 200)
	if !reflect.DeepEqual(before["data"].(map[string]any)["permissions"], after["data"].(map[string]any)["permissions"]) {
		t.Fatal("state change edited code permissions")
	}
	rows, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range rows {
		if strings.Contains(string(r.Data), "role.builtin.state") {
			count++
			if !strings.Contains(string(r.Data), "Review the installation-wide") {
				t.Fatal("missing override reason")
			}
		}
	}
	if count != 2 {
		t.Fatal("unexpected state audit count", count)
	}
}

func TestBuiltinRoleAPICannotWidenProjectOrRestrictedAuthority(t *testing.T) {
	c := assignmentSetup(t)
	path := "/api/v1/predefined-roles/builtin.teststudio.editor"
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	for _, route := range []string{"/api/v1/predefined-roles", path} {
		accessCall(t, c, "GET", route, nil, "", 403)
	}
	accessCall(t, c, "PUT", path, builtinChange(true), "1", 403)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/permissions/grants", directGrant("user", "admin", "roles.manage", "role/builtin.teststudio.editor", permissions.EffectDeny), "", 200)
	row := accessCall(t, c, "GET", path, nil, "", 200)
	if row["data"].(map[string]any)["canManage"] != false {
		t.Fatal(row)
	}
	accessCall(t, c, "PUT", path, builtinChange(true), assignmentRevision(row), 403)
	accessCall(t, c, "POST", "/api/v1/permissions/grants", directGrant("user", "admin", "roles.read", "role/builtin.teststudio.viewer", permissions.EffectDeny), "", 200)
	accessCall(t, c, "GET", "/api/v1/predefined-roles/builtin.teststudio.viewer", nil, "", 403)
	page := accessCall(t, c, "GET", "/api/v1/predefined-roles", nil, "", 200)
	if len(page["items"].([]any)) != 10 {
		t.Fatal("hidden predefined role leaked", page)
	}
	accessCall(t, c, "POST", "/api/v1/permissions/grants", directGrant("user", "admin", "catalog.publish", "", permissions.EffectDeny), "", 200)
	accessCall(t, c, "PUT", "/api/v1/predefined-roles/builtin.teststudio.publisher", builtinChange(true), "1", 403)
	accessCall(t, c, "GET", "/api/v1/predefined-roles/custom-role", nil, "", 404)
	accessCall(t, c, "PUT", "/api/v1/predefined-roles/builtin.unknown", builtinChange(true), "1", 404)
}

func TestBuiltinRoleAPIKeepsTheFinalAdministratorAndAllowsCompleteCustomRecovery(t *testing.T) {
	c := assignmentSetup(t)
	id := "builtin.teststudio.installation-admin"
	path := "/api/v1/predefined-roles/" + id
	before := accessCall(t, c, "GET", path, nil, "", 200)
	version, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	blocked := accessCall(t, c, "PUT", path, builtinChange(true), assignmentRevision(before), 422)
	if !strings.Contains(blocked["message"].(string), "last usable") {
		t.Fatal(blocked)
	}
	current, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || current.Revision != version.Revision {
		t.Fatal("rejected override changed directory", err)
	}
	if _, err := c.api.Store.Get(t.Context(), "access-builtin-roles", id); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("rejected override retained metadata", err)
	}
	custom := roleInput("complete-administrator", "")
	for _, name := range authorization.PermissionNames() {
		custom.Rules = append(custom.Rules, roleRule(strings.TrimPrefix(name, "teststudio."), permissions.EffectAllow, ""))
	}
	accessCall(t, c, "POST", "/api/v1/roles", custom, "", 200)
	accessCall(t, c, "POST", "/api/v1/role-assignments", map[string]string{"principalKind": "user", "principalId": "admin", "roleId": custom.ID, "reason": "Keep complete alternate administrator access"}, "", 200)
	disabled := accessCall(t, c, "PUT", path, builtinChange(true), assignmentRevision(before), 200)
	if disabled["data"].(map[string]any)["canManage"] != true {
		t.Fatal(disabled)
	}
	roleAllowed(t, c, "admin", "", "", "system.manage", true)
	accessCall(t, c, "PUT", path, builtinChange(false), assignmentRevision(disabled), 200)
	row := accessCall(t, c, "GET", path, nil, "", 200)
	if row["data"].(map[string]any)["disabled"] != false {
		t.Fatal(row)
	}
}

func TestBuiltinRoleAPIRejectsStaleUnmanagedStateAndInconsistentMetadata(t *testing.T) {
	c := assignmentSetup(t)
	id := "builtin.teststudio.editor"
	path := "/api/v1/predefined-roles/" + id
	before := accessCall(t, c, "GET", path, nil, "", 200)
	if err := c.api.Auth.Engine.DisableBuiltInRole(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	current := accessCall(t, c, "GET", path, nil, "", 200)
	if current["revision"] == before["revision"] {
		t.Fatal("native state change reused revision")
	}
	accessCall(t, c, "PUT", path, builtinChange(false), assignmentRevision(before), 409)
	accessCall(t, c, "PUT", path, builtinChange(false), assignmentRevision(current), 200)
	if err := c.api.Auth.Engine.DisableBuiltInRole(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "GET", path, nil, "", 409)
	accessCall(t, c, "PUT", path, builtinChange(false), "3", 409)
}
