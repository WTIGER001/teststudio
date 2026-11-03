package httpapi_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func assignmentSignInAs(t *testing.T, c *client, user string) {
	t.Helper()
	w := httptest.NewRecorder()
	session, err := c.api.Sessions.Create(t.Context(), w, user, "example", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	origin, err := url.Parse(c.origin)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Jar.SetCookies(origin, w.Result().Cookies())
	c.csrf = session.CSRF
}

func assignmentSetup(t *testing.T) *client {
	t.Helper()
	c := fixture(t)
	c.signIn()
	for _, id := range []string{"alpha", "beta"} {
		accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": id, "name": id}, "", 200)
	}
	for _, id := range []string{"manager", "member"} {
		accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": id, "name": id}, "", 200)
	}
	return c
}

func assignBody(kind, principal, role string) map[string]string {
	return map[string]string{"principalKind": kind, "principalId": principal, "roleId": "builtin.teststudio." + role, "reason": "Set explicit access"}
}

func assignmentRevision(row map[string]any) string {
	return strconv.FormatInt(int64(row["revision"].(float64)), 10)
}

func TestAccessAssignmentAPIKeepsExactProjectBindingsAndRecreationRevisions(t *testing.T) {
	c := assignmentSetup(t)
	a, err := authorization.New(t.Context(), c.api.Auth.Engine.GetStore(), c.api.Auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, project := range []string{"alpha", "beta"} {
		path := "/api/v1/projects/" + project + "/role-assignments"
		row := accessCall(t, c, "POST", path, assignBody("user", "member", "editor"), "", 200)
		if row["data"].(map[string]any)["canRevoke"] != true {
			t.Fatal("administrator cannot manage its assignment", row)
		}
		if row["revision"] != float64(2) || row["updatedAt"] == nil || row["data"].(map[string]any)["managed"] != true {
			t.Fatal(row)
		}
		bindings := row["data"].(map[string]any)["bindingValues"].(map[string]any)
		if len(bindings) != 1 || bindings["team"] != project {
			t.Fatal("server did not derive exact project bindings", row)
		}
		if err := a.Check(t.Context(), "member", project, "", "catalog.update"); err != nil {
			t.Fatal("existing engine missed assignment", err)
		}
		accessCall(t, c, "POST", path, assignBody("user", "member", "editor"), "", 409)
	}
	path := "/api/v1/projects/alpha/role-assignments/user/member/builtin.teststudio.editor"
	row := accessCall(t, c, "GET", path, nil, "", 200)
	revision := assignmentRevision(row)
	accessCall(t, c, "DELETE", path, map[string]string{"reason": "Remove alpha only"}, "0", 409)
	accessCall(t, c, "DELETE", path, map[string]string{"reason": " "}, revision, 422)
	accessCall(t, c, "DELETE", path, map[string]string{"reason": "Remove alpha only"}, revision, 200)
	accessCall(t, c, "GET", path, nil, "", 404)
	if err := a.Check(t.Context(), "member", "alpha", "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("revocation was cached", err)
	}
	if err := a.Check(t.Context(), "member", "beta", "", "catalog.update"); err != nil {
		t.Fatal("revocation crossed projects", err)
	}
	recreated := accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "member", "editor"), "", 200)
	if recreated["id"] != row["id"] || recreated["revision"].(float64) <= row["revision"].(float64) {
		t.Fatal("recreation reused an old revision", recreated, row)
	}
	accessCall(t, c, "DELETE", path, map[string]string{"reason": "Stale removal"}, revision, 409)
	accessCall(t, c, "DELETE", path, map[string]string{"reason": "Fresh removal"}, assignmentRevision(recreated), 200)
	audits, err := c.api.Store.All(t.Context(), "audit", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	assignments := 0
	for _, row := range audits {
		var event struct {
			Action string         `json:"action"`
			After  map[string]any `json:"after"`
		}
		if err := json.Unmarshal(row.Data, &event); err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(event.Action, "role.") {
			assignments++
			if event.After["reason"] == "" {
				t.Fatal("assignment audit lost reason", event)
			}
		}
	}
	if assignments != 4 {
		t.Fatal("assignment audit count", assignments)
	}
}

func TestAccessAssignmentAPILimitsDelegationAndSharedGroupChanges(t *testing.T) {
	c := assignmentSetup(t)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	for _, group := range []struct{ id, project string }{{"own", "alpha"}, {"foreign", "beta"}, {"shared", ""}} {
		accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": group.id, "name": group.id, "projectId": group.project}, "", 200)
	}
	assignmentSignInAs(t, c, "manager")
	if err := c.api.Auth.Check(t.Context(), "manager", "alpha", "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("project administrator unexpectedly has editor actions", err)
	}
	roles := accessCall(t, c, "GET", "/api/v1/roles?project=alpha", nil, "", 200)
	if len(roles["items"].([]any)) != 5 {
		t.Fatal(roles)
	}
	for _, item := range roles["items"].([]any) {
		role := item.(map[string]any)
		if role["scope"] != "project" || role["delegatable"] != true {
			t.Fatal(role)
		}
	}
	base := "/api/v1/projects/alpha/role-assignments"
	accessCall(t, c, "POST", base, assignBody("user", "member", "editor"), "", 200)
	accessCall(t, c, "POST", base, assignBody("group", "own", "operator"), "", 200)
	accessCall(t, c, "POST", base, assignBody("group", "shared", "operator"), "", 403)
	accessCall(t, c, "POST", base, assignBody("group", "foreign", "operator"), "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/beta/role-assignments", assignBody("user", "member", "operator"), "", 403)
	accessCall(t, c, "GET", "/api/v1/projects/beta/role-assignments", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/role-assignments", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/roles", nil, "", 403)
	accessCall(t, c, "POST", base, assignBody("user", "manager", "installation-admin"), "", 422)
	accessCall(t, c, "POST", base, assignBody("team", "member", "viewer"), "", 422)
	accessCall(t, c, "POST", base, assignBody("user", "member", "unknown"), "", 422)
	accessCall(t, c, "POST", base, map[string]any{"principalKind": "user", "principalId": "member", "roleId": "builtin.teststudio.operator", "bindingValues": map[string]any{"team": "*"}, "reason": "Widen scope"}, "", 422)
	object := "alpha/role/builtin.teststudio.viewer"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.permissions.grant", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", base, assignBody("user", "member", "viewer"), "", 403)
	accessCall(t, c, "POST", base, assignBody("group", "own", "viewer"), "", 403)
	if err := c.api.Auth.Check(t.Context(), "member", "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("project assignment granted installation access", err)
	}
	c.signIn()
	for _, project := range []string{"alpha", "beta"} {
		accessCall(t, c, "POST", "/api/v1/projects/"+project+"/role-assignments", assignBody("group", "shared", "publisher"), "", 200)
	}
	accessCall(t, c, "POST", "/api/v1/projects/beta/role-assignments", assignBody("group", "own", "publisher"), "", 404)
	assignmentSignInAs(t, c, "manager")
	sharedPath := "/api/v1/projects/alpha/role-assignments/group/shared/builtin.teststudio.publisher"
	shared := accessCall(t, c, "GET", sharedPath, nil, "", 200)
	if shared["data"].(map[string]any)["canRevoke"] != false {
		t.Fatal("shared group authority was inferred from project authority", shared)
	}
	accessCall(t, c, "DELETE", sharedPath, map[string]string{"reason": "Cannot manage a shared group"}, assignmentRevision(shared), 403)
	accessCall(t, c, "GET", "/api/v1/groups/shared", nil, "", 403)
	memberObject := "alpha/user/member"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", TeamScope: "alpha", ObjectScope: &memberObject, PermissionName: "teststudio.members.manage", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", base, assignBody("user", "member", "publisher"), "", 403)
	accessCall(t, c, "POST", base, assignBody("user", "manager", "publisher"), "", 200)

}

func TestAccessAssignmentAPIProtectsLastUsableAdministratorAndAdoptsLegacyRecords(t *testing.T) {
	c := assignmentSetup(t)
	root := "/api/v1/role-assignments/user/admin/builtin.teststudio.installation-admin"
	row := accessCall(t, c, "GET", root, nil, "", 200)
	if row["revision"] != float64(1) || row["updatedAt"] != nil || row["data"].(map[string]any)["managed"] != false {
		t.Fatal("legacy bootstrap assignment hidden or rewritten", row)
	}
	version, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	before, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	denied := accessCall(t, c, "DELETE", root, map[string]string{"reason": "Try removing final administrator"}, "1", 422)
	if !strings.Contains(denied["message"].(string), "last usable") {
		t.Fatal(denied)
	}
	after, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || version.Revision != after.Revision {
		t.Fatal("rejected removal advanced directory revision", after, err)
	}
	audits, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil || len(audits) != len(before) {
		t.Fatal("rejected removal left an audit change", audits, err)
	}
	if _, err := c.api.Store.Get(t.Context(), "access-assignments", row["id"].(string)); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("rejected removal left metadata", err)
	}
	accessCall(t, c, "POST", "/api/v1/role-assignments", assignBody("user", "member", "installation-admin"), "", 200)
	accessCall(t, c, "DELETE", root, map[string]string{"reason": "Unlinked accounts cannot provide recovery"}, "1", 422)
	provider, _ := storage.Put("providers", "example", "", 0, map[string]any{"issuer": "https://identity.example.test", "clientId": "teststudio", "disabled": false})
	if err := c.api.Store.Commit(t.Context(), provider); err != nil {
		t.Fatal(err)
	}
	backup, err := c.api.Auth.Directory.ResolveExternal(t.Context(), "https://identity.example.test", "backup", "Backup", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "recovery", "name": "Recovery administrators"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/recovery/members/"+backup.ID, map[string]any{}, "0", 200)
	group := accessCall(t, c, "POST", "/api/v1/role-assignments", assignBody("group", "recovery", "installation-admin"), "", 200)
	accessCall(t, c, "DELETE", root, map[string]string{"reason": "Use the configured group administrator"}, "1", 200)
	if err := c.api.Auth.Check(t.Context(), "admin", "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("revocation did not reach current service", err)
	}
	accessCall(t, c, "GET", "/api/v1/role-assignments", nil, "", 403)
	assignmentSignInAs(t, c, backup.ID)
	accessCall(t, c, "DELETE", "/api/v1/role-assignments/group/recovery/builtin.teststudio.installation-admin", map[string]string{"reason": "Cannot remove final login path"}, assignmentRevision(group), 422)
	accessCall(t, c, "GET", "/api/v1/role-assignments/group/recovery/builtin.teststudio.installation-admin", nil, "", 200)
}

func TestAccessAssignmentAPIPagesAreBoundedFilteredAndScopeBound(t *testing.T) {
	c := assignmentSetup(t)
	for _, project := range []string{"alpha", "beta"} {
		for _, role := range []string{"editor", "viewer", "operator"} {
			accessCall(t, c, "POST", "/api/v1/projects/"+project+"/role-assignments", assignBody("user", "member", role), "", 200)
		}
	}
	object := "alpha/role/builtin.teststudio.viewer"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.permissions.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	path, cursor, count, scans := "/api/v1/projects/alpha/role-assignments?limit=1", "", 0, 0
	var initialCursor string
	for {
		page := accessCall(t, c, "GET", path+"&cursor="+url.QueryEscape(cursor), nil, "", 200)
		items := page["items"].([]any)
		if len(items) > 1 {
			t.Fatal("unbounded assignment page", page)
		}
		for _, item := range items {
			row := item.(map[string]any)
			if row["projectId"] != "alpha" || row["data"].(map[string]any)["roleId"] == "builtin.teststudio.viewer" {
				t.Fatal("private assignment crossed scope or deny", row)
			}
		}
		count += len(items)
		scans++
		cursor, _ = page["cursor"].(string)
		if cursor == "" {
			if page["partial"] != false {
				t.Fatal(page)
			}
			break
		}
		if initialCursor == "" {
			initialCursor = cursor
		}
		if page["partial"] != true || scans > 10 {
			t.Fatal("invalid bounded continuation", page)
		}
	}
	if count != 2 || scans != 7 {
		t.Fatal("filtered assignment coverage", count, scans)
	}
	accessCall(t, c, "GET", "/api/v1/projects/beta/role-assignments?cursor="+url.QueryEscape(initialCursor), nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/users?cursor="+url.QueryEscape(initialCursor), nil, "", 422)
	accessCall(t, c, "GET", path+"&cursor=corrupt", nil, "", 422)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "GET", path+"&cursor="+url.QueryEscape(initialCursor), nil, "", 422)
}

func TestAccessAssignmentAPIRejectsInactiveTargetsButAllowsScopedCleanup(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/projects/alpha/role-assignments"
	row := accessCall(t, c, "POST", base, assignBody("user", "member", "editor"), "", 200)
	accessCall(t, c, "POST", "/api/v1/users/member/disable", map[string]string{"reason": "Inactive account"}, "1", 200)
	accessCall(t, c, "POST", base, assignBody("user", "member", "viewer"), "", 422)
	accessCall(t, c, "DELETE", base+"/user/member/builtin.teststudio.editor", map[string]string{"reason": "Remove inactive account access"}, assignmentRevision(row), 200)
	if err := c.api.Auth.Engine.DisableBuiltInRole(t.Context(), "builtin.teststudio.operator"); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", base, assignBody("user", "manager", "operator"), "", 422)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "inactive", "name": "Inactive", "projectId": "alpha"}, "", 200)
	accessCall(t, c, "POST", "/api/v1/groups/inactive/archive", map[string]string{"reason": "Retired group"}, "1", 200)
	accessCall(t, c, "POST", base, assignBody("group", "inactive", "viewer"), "", 422)
	project := accessCall(t, c, "GET", "/api/v1/projects/alpha", nil, "", 200)
	value := project["data"].(map[string]any)
	value["archived"] = true
	accessCall(t, c, "PUT", "/api/v1/projects/alpha", value, fmt.Sprint(project["revision"]), 200)
	accessCall(t, c, "POST", base, assignBody("user", "manager", "viewer"), "", 422)
}
