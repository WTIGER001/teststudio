package httpapi_test

import (
	"fmt"
	"net/url"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
)

func TestDirectGrantReadOnlyAndObjectDeniesControlCapabilities(t *testing.T) {
	c := roleSetup(t)
	path := "/api/v1/projects/alpha/permissions"
	first := accessCall(t, c, "POST", path, directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
	second := accessCall(t, c, "POST", path, directGrant("user", "member", "runs.read", "", permissions.EffectAllow), "", 200)
	page := accessCall(t, c, "GET", path+"?limit=1", nil, "", 200)
	cursor := page["cursor"].(string)
	for _, action := range []string{"members.read", "permissions.read"} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio." + action, Effect: permissions.EffectAllow, TeamScope: "alpha"}); err != nil {
			t.Fatal(err)
		}
	}
	assignmentSignInAs(t, c, "member")
	read := accessCall(t, c, "GET", path+"/"+first["id"].(string), nil, "", 200)
	if read["data"].(map[string]any)["canUpdate"] != false || read["data"].(map[string]any)["canRemove"] != false {
		t.Fatal(read)
	}
	if accessCall(t, c, "GET", path, nil, "", 200)["canCreate"] != false {
		t.Fatal("read-only caller can create")
	}
	accessCall(t, c, "GET", path+"?cursor="+url.QueryEscape(cursor), nil, "", 422)
	accessCall(t, c, "POST", path, directGrant("user", "member", "evidence.read", "", permissions.EffectAllow), "", 403)
	accessCall(t, c, "DELETE", path+"/"+first["id"].(string), map[string]string{"reason": "Cannot remove read-only grant"}, assignmentRevision(first), 403)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	object := "alpha/grant/" + first["id"].(string)
	for _, action := range []string{"permissions.read", "permissions.grant"} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", PermissionName: "teststudio." + action, Effect: permissions.EffectDeny, TeamScope: "alpha", ObjectScope: &object}); err != nil {
			t.Fatal(err)
		}
	}
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "GET", path+"/"+first["id"].(string), nil, "", 403)
	accessCall(t, c, "PUT", path+"/"+first["id"].(string), access.GrantUpdate{Reason: "Cannot bypass object denial"}, assignmentRevision(first), 403)
	filtered := accessCall(t, c, "GET", path, nil, "", 200)
	if len(filtered["items"].([]any)) != 1 || filtered["items"].([]any)[0].(map[string]any)["id"] != second["id"] {
		t.Fatal("filtered grant leaked", filtered)
	}
	accessCall(t, c, "POST", path, directGrant("user", "member", "evidence.read", "grant/"+first["id"].(string), permissions.EffectAllow), "", 403)
	c.signIn()
	accessCall(t, c, "POST", path, directGrant("user", "member", "evidence.read", "grant/"+first["id"].(string), permissions.EffectAllow), "", 200)
}

func TestDirectGrantArchivedScopesPermitRemovalAndRejectRenewal(t *testing.T) {
	c := roleSetup(t)
	path := "/api/v1/projects/alpha/permissions"
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "operators", "name": "Operators", "projectId": "alpha"}, "", 200)
	row := accessCall(t, c, "POST", path, directGrant("group", "operators", "catalog.read", "", permissions.EffectAllow), "", 200)
	detail := path + "/" + row["id"].(string)
	accessCall(t, c, "POST", "/api/v1/groups/operators/archive", map[string]string{"reason": "Archive the grant principal"}, "1", 200)
	read := accessCall(t, c, "GET", detail, nil, "", 200)
	if read["data"].(map[string]any)["canUpdate"] != false || read["data"].(map[string]any)["canRemove"] != true {
		t.Fatal(read)
	}
	accessCall(t, c, "PUT", detail, access.GrantUpdate{Reason: "Reject renewal for archived group"}, assignmentRevision(row), 422)
	accessCall(t, c, "DELETE", detail, map[string]string{"reason": "Clean the archived grant"}, assignmentRevision(row), 200)
	row = accessCall(t, c, "POST", path, directGrant("user", "member", "runs.read", "", permissions.EffectAllow), "", 200)
	detail = path + "/" + row["id"].(string)
	accessCall(t, c, "PUT", "/api/v1/projects/alpha", map[string]any{"id": "alpha", "name": "Alpha", "archived": true}, "1", 200)
	page := accessCall(t, c, "GET", path, nil, "", 200)
	if page["canCreate"] != false {
		t.Fatal(page)
	}
	read = accessCall(t, c, "GET", detail, nil, "", 200)
	if read["data"].(map[string]any)["canUpdate"] != false || read["data"].(map[string]any)["canRemove"] != true {
		t.Fatal(read)
	}
	accessCall(t, c, "PUT", detail, access.GrantUpdate{Reason: "Reject renewal for archived project"}, assignmentRevision(row), 422)
	accessCall(t, c, "POST", path, directGrant("user", "member", "catalog.read", "", permissions.EffectAllow), "", 422)
	accessCall(t, c, "DELETE", detail, map[string]string{"reason": "Clean the archived project's grant"}, assignmentRevision(row), 200)
}

func TestDirectGrantCreationScansNativePagesWithoutTotalCeiling(t *testing.T) {
	c := roleSetup(t)
	for start := 0; start < 1101; start += 100 {
		grants := []permissions.Grant{}
		for index := start; index < start+100 && index < 1101; index++ {
			object := fmt.Sprintf("alpha/suite/unmanaged-%04d", index)
			grants = append(grants, permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio.catalog.read", TeamScope: "alpha", Effect: permissions.EffectAllow, ObjectScope: &object})
		}
		if err := c.api.Auth.Engine.CreateGrants(t.Context(), grants); err != nil {
			t.Fatal(err)
		}
	}
	path := "/api/v1/projects/alpha/permissions"
	input := directGrant("user", "member", "catalog.read", "", permissions.EffectAllow)
	row := accessCall(t, c, "POST", path, input, "", 200)
	accessCall(t, c, "POST", path, input, "", 409)
	accessCall(t, c, "GET", path+"/"+row["id"].(string), nil, "", 200)
}
