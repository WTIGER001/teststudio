package httpapi_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/wtiger001/teststudio/internal/authorization"
)

func conversionActions(raw any) []string {
	result := []string{}
	for _, value := range raw.([]any) {
		result = append(result, value.(string))
	}
	return result
}

func TestRoleConversionAPIRequiresCurrentReviewedStateAndSupportsRollback(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/projects/alpha/role-conversions"
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "member", "operator"), "", 200)
	preview := accessCall(t, c, "POST", base+"/preview", assignBody("user", "member", "member"), "", 200)
	data := preview["data"].(map[string]any)
	if !slices.Contains(conversionActions(data["removedPermissions"]), "teststudio.schedules.delete") {
		t.Fatal("deletion loss not reviewed", preview)
	}
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "schedules.delete"); err != nil {
		t.Fatal("preview changed access", err)
	}
	id := preview["id"].(string)
	applied := accessCall(t, c, "POST", base+"/"+id+"/apply", nil, assignmentRevision(preview), 200)
	if applied["data"].(map[string]any)["applied"] != true {
		t.Fatal(applied)
	}
	accessCall(t, c, "POST", base+"/"+id+"/apply", nil, assignmentRevision(preview), 200)
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "schedules.delete"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("old deletion access retained", err)
	}
	rollback := accessCall(t, c, "POST", base+"/"+id+"/rollback-preview", map[string]string{"reason": "Restore reviewed original policy"}, "", 200)
	accessCall(t, c, "POST", base+"/"+rollback["id"].(string)+"/apply", nil, assignmentRevision(rollback), 200)
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "schedules.delete"); err != nil {
		t.Fatal("rollback lost original action", err)
	}
	stale := accessCall(t, c, "POST", base+"/preview", assignBody("user", "member", "reader"), "", 200)
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "new-user", "name": "Change directory revision"}, "", 200)
	accessCall(t, c, "POST", base+"/"+stale["id"].(string)+"/apply", nil, assignmentRevision(stale), 409)
	accessCall(t, c, "GET", "/api/v1/projects/beta/role-conversions/"+id, nil, "", 404)
}

func TestStandardOwnerAPIPreservesLastUsableOwner(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/projects/alpha/role-assignments"
	owner := accessCall(t, c, "POST", base, assignBody("user", "admin", "owner"), "", 200)
	status := accessCall(t, c, "GET", "/api/v1/projects/alpha/ownership", nil, "", 200)
	if status["state"] != "owned" {
		t.Fatal(status)
	}
	accessCall(t, c, "DELETE", base+"/user/admin/"+authorization.OwnerRole, map[string]string{"reason": "Cannot remove final Owner"}, assignmentRevision(owner), 422)
	accessCall(t, c, "PUT", base+"/user/admin", map[string]string{"previousRoleId": authorization.OwnerRole, "roleId": authorization.ContributorRole, "reason": "Cannot demote final Owner"}, assignmentRevision(owner), 422)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "admin", "members.owners", "", "deny"), "", 422)
	if err := c.api.Auth.Check(t.Context(), "admin", "alpha", "", "members.owners"); err != nil {
		t.Fatal("rejected change survived", err)
	}
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "owner-team", "name": "Owner team", "projectId": "alpha"}, "", 200)
	accessCall(t, c, "POST", base, assignBody("group", "owner-team", "owner"), "", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/owner-team/members/admin", map[string]any{}, "0", 200)
	accessCall(t, c, "DELETE", base+"/user/admin/"+authorization.OwnerRole, map[string]string{"reason": "Retain group Owner access"}, assignmentRevision(owner), 200)
	accessCall(t, c, "POST", "/api/v1/groups/owner-team/archive", map[string]string{"reason": "Cannot archive final Owner path"}, "1", 422)
	accessCall(t, c, "DELETE", "/api/v1/groups/owner-team/members/admin", map[string]string{"reason": "Cannot remove final Owner member"}, "1", 422)
}
