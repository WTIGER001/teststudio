package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func directGrant(kind, principal, permission, object string, effect permissions.Effect) access.GrantDefinition {
	return access.GrantDefinition{PrincipalKind: permissions.PrincipalKind(kind), PrincipalID: principal, Permission: "teststudio." + permission, Object: object, Effect: effect, Reason: "Set explicit principal access"}
}

func TestDirectGrantAPIKeepsExactScopeExpiryAndStableRevisions(t *testing.T) {
	c := roleSetup(t)
	path := "/api/v1/projects/alpha/permissions"
	input := directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectAllow)
	row := accessCall(t, c, "POST", path, input, "", 200)
	id := row["id"].(string)
	detail := path + "/" + id
	if row["revision"] != float64(1) || row["data"].(map[string]any)["canUpdate"] != true {
		t.Fatal(row)
	}
	raw, _ := json.Marshal(row)
	if strings.Contains(string(raw), "nativeId") {
		t.Fatal("native identifier leaked", row)
	}
	accessCall(t, c, "POST", path, input, "", 409)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	roleAllowed(t, c, "member", "alpha", "suite/other", "catalog.read", false)
	roleAllowed(t, c, "member", "beta", "suite/health", "catalog.read", false)
	other := accessCall(t, c, "POST", path, directGrant("user", "member", "runs.read", "", permissions.EffectAllow), "", 200)
	native, err := c.api.Auth.Engine.GetStore().ListGrants(t.Context(), permissions.GrantQuery{Permissions: []string{"teststudio.runs.read"}, Principals: []permissions.PrincipalRef{{Kind: permissions.PrincipalUser, ID: "member"}}})
	if err != nil || len(native.Grants) != 1 {
		t.Fatal(native, err)
	}
	expiry := time.Date(2000, 2, 3, 4, 5, 6, 123456789, time.FixedZone("test", 3600))
	row = accessCall(t, c, "PUT", detail, access.GrantUpdate{ExpiresAt: &expiry, Reason: "Retain an expired grant"}, "1", 200)
	if row["data"].(map[string]any)["expiresAt"] != "2000-02-03T03:05:06.123456Z" || row["data"].(map[string]any)["expired"] != true {
		t.Fatal(row)
	}
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", false)
	accessCall(t, c, "PUT", detail, access.GrantUpdate{Reason: "Stale edit"}, "1", 409)
	accessCall(t, c, "PUT", detail, map[string]string{"expiresAt": "0000-01-01T00:00:00Z", "reason": "Reject unsupported expiry"}, assignmentRevision(row), 422)
	accessCall(t, c, "PUT", detail, map[string]any{"permission": "teststudio.runs.read", "reason": "Reject mutable identity"}, assignmentRevision(row), 422)
	row = accessCall(t, c, "PUT", detail, access.GrantUpdate{Reason: "Restore non-expiring access"}, assignmentRevision(row), 200)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	after, err := c.api.Auth.ReadPolicyGrant(t.Context(), native.Grants[0].ID)
	if err != nil || after.ID != native.Grants[0].ID {
		t.Fatal("unrelated grant replaced", after, err)
	}
	accessCall(t, c, "GET", "/api/v1/projects/beta/permissions/"+id, nil, "", 404)
	accessCall(t, c, "GET", "/api/v1/permissions/grants/"+id, nil, "", 404)
	accessCall(t, c, "DELETE", detail, map[string]string{"reason": "Remove this exact grant"}, assignmentRevision(row), 200)
	accessCall(t, c, "GET", detail, nil, "", 404)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", false)
	row2 := accessCall(t, c, "POST", path, input, "", 200)
	if row2["id"] != id || row2["revision"].(float64) <= row["revision"].(float64) {
		t.Fatal("recreation lost revisions", row, row2)
	}
	accessCall(t, c, "DELETE", detail, map[string]string{"reason": "Stale removal"}, assignmentRevision(row), 409)
	accessCall(t, c, "DELETE", detail, map[string]string{"reason": "Fresh removal"}, assignmentRevision(row2), 200)
	accessCall(t, c, "GET", path+"/"+other["id"].(string), nil, "", 200)
	rows, err := c.api.Store.All(t.Context(), "audit", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range rows {
		if strings.Contains(string(r.Data), "permission.grant.") {
			count++
			if !strings.Contains(string(r.Data), "reason") {
				t.Fatal("missing reason")
			}
		}
	}
	if count != 7 {
		t.Fatalf("grant audit count %d", count)
	}
}

func TestDirectGrantDenyOverridesInheritedRolesAndExpiredDeniesStayVisible(t *testing.T) {
	c := roleSetup(t)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "operators", "name": "Operators", "projectId": "alpha"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/operators/members/member", map[string]any{}, "0", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("group", "operators", "viewer"), "", 200)
	path := "/api/v1/projects/alpha/permissions"
	allow := accessCall(t, c, "POST", path, directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
	deny := accessCall(t, c, "POST", path, directGrant("group", "operators", "catalog.read", "suite/health", permissions.EffectDeny), "", 200)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", false)
	roleAllowed(t, c, "member", "alpha", "suite/other", "catalog.read", true)
	expiry := time.Now().Add(-time.Hour)
	deny = accessCall(t, c, "PUT", path+"/"+deny["id"].(string), access.GrantUpdate{ExpiresAt: &expiry, Reason: "Expire the explicit deny"}, assignmentRevision(deny), 200)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	page := accessCall(t, c, "GET", path, nil, "", 200)
	if len(page["items"].([]any)) != 2 {
		t.Fatal("expired deny disappeared", page)
	}
	accessCall(t, c, "DELETE", path+"/"+allow["id"].(string), map[string]string{"reason": "Keep role access"}, assignmentRevision(allow), 200)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", true)
	accessCall(t, c, "DELETE", "/api/v1/groups/operators/members/member", map[string]string{"reason": "Revoke inherited group access"}, "1", 200)
	roleAllowed(t, c, "member", "alpha", "suite/health", "catalog.read", false)
}

func TestDirectGrantPaginationBindsActorScopeAndCollection(t *testing.T) {
	c := roleSetup(t)
	path := "/api/v1/projects/alpha/permissions"
	for _, permission := range []string{"catalog.read", "runs.read", "evidence.read"} {
		accessCall(t, c, "POST", path, directGrant("user", "member", permission, "", permissions.EffectAllow), "", 200)
	}
	seen := map[string]bool{}
	cursor := ""
	for {
		page := accessCall(t, c, "GET", path+"?limit=1&cursor="+url.QueryEscape(cursor), nil, "", 200)
		for _, v := range page["items"].([]any) {
			id := v.(map[string]any)["id"].(string)
			if seen[id] {
				t.Fatal("duplicate", id)
			}
			seen[id] = true
		}
		next, _ := page["cursor"].(string)
		if next == "" {
			break
		}
		if cursor == "" {
			accessCall(t, c, "GET", "/api/v1/permissions/grants?cursor="+url.QueryEscape(next), nil, "", 422)
			accessCall(t, c, "GET", "/api/v1/projects/beta/permissions?cursor="+url.QueryEscape(next), nil, "", 422)
			accessCall(t, c, "GET", "/api/v1/roles/custom?project=alpha&cursor="+url.QueryEscape(next), nil, "", 422)
		}
		cursor = next
	}
	if len(seen) != 3 {
		t.Fatal(seen)
	}
	accessCall(t, c, "GET", path+"?limit=0", nil, "", 422)
}

func TestDirectGrantNativeMismatchAndUnmanagedDuplicatesFailClosed(t *testing.T) {
	c := roleSetup(t)
	path := "/api/v1/projects/alpha/permissions"
	input := directGrant("user", "member", "runs.read", "", permissions.EffectAllow)
	native := permissions.Grant{OwnerKind: input.PrincipalKind, OwnerID: input.PrincipalID, PermissionName: input.Permission, TeamScope: "alpha", Effect: input.Effect}
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), native); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", path, input, "", 409)
	input.Permission = "teststudio.catalog.read"
	row := accessCall(t, c, "POST", path, input, "", 200)
	stored, err := c.api.Store.Get(t.Context(), "access-grants/alpha", row["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[map[string]any](stored)
	if err != nil {
		t.Fatal(err)
	}
	id := int64(value["nativeId"].(float64))
	if err := c.api.Auth.WithPolicyTransaction(t.Context(), func(auth *authorization.Service) error { return auth.RemovePolicyGrant(t.Context(), id) }); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "PUT", path+"/"+row["id"].(string), access.GrantUpdate{Reason: "Reject missing native state"}, assignmentRevision(row), 409)
	accessCall(t, c, "DELETE", path+"/"+row["id"].(string), map[string]string{"reason": "Reject missing native state"}, assignmentRevision(row), 409)
	if _, err := c.api.Auth.ReadPolicyGrant(t.Context(), id); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("exact native missing result", err)
	}
}

func TestDirectGrantPreservesUsableAdministrator(t *testing.T) {
	for _, permission := range []string{"system.manage", "users.manage", "permissions.grant", "permissions.read", "roles.manage", "roles.read", "members.manage", "members.read"} {
		t.Run(permission, func(t *testing.T) {
			c := assignmentSetup(t)
			version, err := c.api.Auth.Directory.Revision(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			result := accessCall(t, c, "POST", "/api/v1/permissions/grants", directGrant("user", "admin", permission, "", permissions.EffectDeny), "", 422)
			if !strings.Contains(result["message"].(string), "last usable") {
				t.Fatal(result)
			}
			after, err := c.api.Auth.Directory.Revision(t.Context())
			if err != nil || version.Revision != after.Revision {
				t.Fatal("rejected deny changed revision", err)
			}
			grants, err := c.api.Store.All(t.Context(), "access-grants/", "")
			if err != nil || len(grants) != 0 {
				t.Fatal("rejected deny retained metadata", grants, err)
			}
			roleAllowed(t, c, "admin", "", "", permission, true)
		})
	}
}

func TestDirectGrantAuthorityAndInactivePrincipals(t *testing.T) {
	c := roleSetup(t)
	for _, g := range []map[string]string{{"id": "own-group", "name": "Own", "projectId": "alpha"}, {"id": "foreign-group", "name": "Foreign", "projectId": "beta"}, {"id": "shared-group", "name": "Shared"}} {
		accessCall(t, c, "POST", "/api/v1/groups", g, "", 200)
	}
	path := "/api/v1/projects/alpha/permissions"
	row := accessCall(t, c, "POST", path, directGrant("user", "member", "runs.read", "", permissions.EffectAllow), "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "POST", path, directGrant("group", "own-group", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
	accessCall(t, c, "POST", path, directGrant("group", "shared-group", "catalog.read", "suite/health", permissions.EffectAllow), "", 403)
	accessCall(t, c, "POST", path, directGrant("group", "foreign-group", "catalog.read", "suite/health", permissions.EffectAllow), "", 403)
	accessCall(t, c, "POST", path, directGrant("user", "member", "system.manage", "", permissions.EffectAllow), "", 422)
	accessCall(t, c, "POST", path, directGrant("user", "member", "roles.manage", "", permissions.EffectAllow), "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/beta/permissions", directGrant("user", "member", "catalog.read", "", permissions.EffectAllow), "", 403)
	accessCall(t, c, "POST", "/api/v1/permissions/grants", directGrant("user", "member", "catalog.read", "", permissions.EffectAllow), "", 403)
	for _, object := range []string{"suite/*", "alpha/suite/health", "suite/missing", "user/member"} {
		want := 422
		if object == "suite/missing" {
			want = 404
		}
		accessCall(t, c, "POST", path, directGrant("user", "member", "catalog.read", object, permissions.EffectAllow), "", want)
	}
	c.signIn()
	user := accessCall(t, c, "GET", "/api/v1/users/member", nil, "", 200)
	accessCall(t, c, "POST", "/api/v1/users/member/disable", map[string]string{"reason": "Suspend principal"}, assignmentRevision(user), 200)
	detail := path + "/" + row["id"].(string)
	view := accessCall(t, c, "GET", detail, nil, "", 200)
	if view["data"].(map[string]any)["canUpdate"] != false || view["data"].(map[string]any)["canRemove"] != true {
		t.Fatal(view)
	}
	accessCall(t, c, "PUT", detail, access.GrantUpdate{Reason: "Cannot renew disabled user"}, assignmentRevision(view), 422)
	accessCall(t, c, "DELETE", detail, map[string]string{"reason": "Clean disabled user's grant"}, assignmentRevision(view), 200)
	if _, err := c.api.Auth.ReadPolicyGrant(t.Context(), -1); err == nil {
		t.Fatal("invalid grant accepted")
	}
}
