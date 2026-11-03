package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func roleRule(name string, effect permissions.Effect, object string) access.RoleRule {
	return access.RoleRule{Permission: "teststudio." + name, Effect: effect, Object: object}
}

func roleInput(id, project string, rules ...access.RoleRule) access.RolePolicyMutation {
	return access.RolePolicyMutation{RolePolicy: access.RolePolicy{ID: id, Name: id, Project: project, Rules: rules, Inherits: []string{}}, Reason: "Set explicit scoped policy"}
}

func roleEdit(t *testing.T, row map[string]any) access.RolePolicyMutation {
	t.Helper()
	raw, err := json.Marshal(row["data"])
	if err != nil {
		t.Fatal(err)
	}
	var value access.RolePolicy
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return access.RolePolicyMutation{RolePolicy: value, Reason: "Review and update this policy"}
}

func roleSetup(t *testing.T) *client {
	t.Helper()
	c := assignmentSetup(t)
	for _, project := range []string{"alpha", "beta"} {
		accessCall(t, c, "PUT", "/api/v1/projects/"+project+"/suites/health", map[string]any{"id": "health", "name": "Health", "applications": []string{}, "requiresTarget": false}, "0", 200)
	}
	return c
}

func roleAllowed(t *testing.T, c *client, user, project, object, permission string, want bool) {
	t.Helper()
	err := c.api.Auth.Check(t.Context(), user, project, object, permission)
	if want && err != nil {
		t.Fatal("expected role access", permission, err)
	}
	if !want && !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("expected role denial", permission, err)
	}
}

func TestCustomRoleAPIKeepsObjectRulesInheritanceAndDisabledState(t *testing.T) {
	c := roleSetup(t)
	input := roleInput("object-reader", "alpha", roleRule("catalog.read", permissions.EffectAllow, "suite/health"), roleRule("runs.read", permissions.EffectAllow, ""))
	expiry := time.Date(2000, 1, 2, 3, 4, 5, 123456789, time.UTC)
	expired := roleRule("catalog.read", permissions.EffectDeny, "suite/health")
	expired.ExpiresAt = &expiry
	input.Rules = append(input.Rules, expired)
	base := accessCall(t, c, "POST", "/api/v1/roles", input, "", 200)
	accessCall(t, c, "POST", "/api/v1/roles", input, "", 409)
	parent := roleInput("composite", "alpha", roleRule("runs.read", permissions.EffectDeny, ""))
	parent.Inherits = []string{input.ID}
	composite := accessCall(t, c, "POST", "/api/v1/roles", parent, "", 200)
	assignment := accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": parent.ID, "reason": "Assign the inherited role"}, "", 200)
	if assignment["data"].(map[string]any)["roleName"] != parent.Name {
		t.Fatal(assignment)
	}
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	roleAllowed(t, c, "member", "alpha", "suite/other", "catalog.read", false)
	roleAllowed(t, c, "member", "alpha", "", "catalog.read", false)
	roleAllowed(t, c, "member", "beta", "suite/health", "catalog.read", false)
	roleAllowed(t, c, "member", "alpha", "", "runs.read", false)
	list := accessCall(t, c, "GET", "/api/v1/projects/alpha/role-assignments", nil, "", 200)
	if len(list["items"].([]any)) != 1 {
		t.Fatal("custom assignment not listed", list)
	}
	grants, err := c.api.Auth.Engine.GetStore().ListGrants(t.Context(), permissions.GrantQuery{Principals: []permissions.PrincipalRef{{Kind: permissions.PrincipalRole, ID: input.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	edit := roleEdit(t, base)
	edit.Name = "Updated object reader"
	base = accessCall(t, c, "PUT", "/api/v1/roles/"+edit.ID, edit, assignmentRevision(base), 200)
	after, err := c.api.Auth.Engine.GetStore().ListGrants(t.Context(), permissions.GrantQuery{Principals: []permissions.PrincipalRef{{Kind: permissions.PrincipalRole, ID: input.ID}}})
	if err != nil || !reflect.DeepEqual(grants.Grants, after.Grants) {
		t.Fatal("metadata update replaced grant identifiers", grants, after, err)
	}
	for _, id := range []string{parent.ID, input.ID} {
		row := composite
		if id == input.ID {
			row = base
		}
		value := roleEdit(t, row)
		value.Disabled = true
		disabled := accessCall(t, c, "PUT", "/api/v1/roles/"+id, value, assignmentRevision(row), 200)
		roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", false)
		accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "user", "principalId": "manager", "roleId": id, "reason": "Reject a disabled role"}, "", 422)
		value.Disabled = false
		enabled := accessCall(t, c, "PUT", "/api/v1/roles/"+id, value, assignmentRevision(disabled), 200)
		roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
		if id == input.ID {
			base = enabled
		} else {
			composite = enabled
		}
	}
	value := roleEdit(t, base)
	value.Inherits = []string{parent.ID}
	version, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "PUT", "/api/v1/roles/"+value.ID, value, assignmentRevision(base), 422)
	accessCall(t, c, "PUT", "/api/v1/roles/"+value.ID, roleEdit(t, base), "1", 409)
	current, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || current.Revision != version.Revision {
		t.Fatal("rejected role update advanced the directory", version, current, err)
	}
	row := accessCall(t, c, "GET", "/api/v1/roles/"+value.ID, nil, "", 200)
	if assignmentRevision(row) != assignmentRevision(base) {
		t.Fatal("rejected update changed policy", row)
	}
	accessCall(t, c, "DELETE", "/api/v1/projects/alpha/role-assignments/user/member/"+parent.ID, map[string]string{"reason": "Remove exactly this scoped role"}, assignmentRevision(assignment), 200)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", false)
	rows, err := c.api.Store.All(t.Context(), "audit", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range rows {
		var event struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(row.Data, &event); err != nil {
			t.Fatal(err)
		}
		if event.Action == "role.policy.save" {
			count++
		}
	}
	if count != 7 {
		t.Fatal("policy audit does not match committed saves", count)
	}
}

func TestCustomRoleAPIPagesBindActorScopeAndObjectPermission(t *testing.T) {
	c := roleSetup(t)
	for _, entry := range [][2]string{{"a-reader", "alpha"}, {"b-reader", "alpha"}, {"other-reader", "beta"}, {"installation-reader", ""}} {
		accessCall(t, c, "POST", "/api/v1/roles", roleInput(entry[0], entry[1]), "", 200)
	}
	page := accessCall(t, c, "GET", "/api/v1/roles/custom?project=alpha&limit=1", nil, "", 200)
	if len(page["items"].([]any)) != 1 || page["partial"] != true {
		t.Fatal(page)
	}
	cursor := page["cursor"].(string)
	accessCall(t, c, "GET", "/api/v1/roles/custom?project=beta&limit=1&cursor="+url.QueryEscape(cursor), nil, "", 422)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "GET", "/api/v1/roles/custom?project=alpha&limit=1&cursor="+url.QueryEscape(cursor), nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/roles/other-reader", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/roles/custom", nil, "", 403)
	c.signIn()
	object := "alpha/role/a-reader"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.roles.read", TeamScope: "alpha", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	filtered := accessCall(t, c, "GET", "/api/v1/roles/custom?project=alpha&limit=1", nil, "", 200)
	if len(filtered["items"].([]any)) != 0 || filtered["cursor"] == nil {
		t.Fatal("filtered page lost continuation", filtered)
	}
	next := accessCall(t, c, "GET", "/api/v1/roles/custom?project=alpha&limit=1&cursor="+url.QueryEscape(filtered["cursor"].(string)), nil, "", 200)
	if len(next["items"].([]any)) != 1 || next["items"].([]any)[0].(map[string]any)["id"] != "b-reader" {
		t.Fatal(next)
	}
	accessCall(t, c, "GET", "/api/v1/roles/a-reader", nil, "", 403)
	installation := accessCall(t, c, "GET", "/api/v1/roles/custom", nil, "", 200)
	if len(installation["items"].([]any)) != 1 || installation["items"].([]any)[0].(map[string]any)["id"] != "installation-reader" {
		t.Fatal("installation listing leaked project roles", installation)
	}
}
