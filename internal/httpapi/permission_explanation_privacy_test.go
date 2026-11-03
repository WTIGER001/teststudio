package httpapi_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
)

func TestPermissionExplanationRespectsGrantRoleAndMembershipObjectDenies(t *testing.T) {
	for _, kind := range []string{"grant", "role", "group"} {
		t.Run(kind, func(t *testing.T) {
			c := roleSetup(t)
			accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "policy-reviewers", "name": "Policy reviewers", "projectId": "alpha"}, "", 200)
			accessCall(t, c, "PUT", "/api/v1/groups/policy-reviewers/members/member", map[string]any{}, "0", 200)
			accessCall(t, c, "POST", "/api/v1/roles", roleInput("policy-reader", "alpha", roleRule("catalog.read", permissions.EffectAllow, "suite/health")), "", 200)
			accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "group", "principalId": "policy-reviewers", "roleId": "policy-reader", "reason": "Inspect permitted role paths"}, "", 200)
			grant := accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
			accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("group", "policy-reviewers", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
			accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
			object, action := "grant/"+grant["id"].(string), "permissions.read"
			if kind == "role" {
				object = "role/policy-reader"
			}
			if kind == "group" {
				object, action = "group/policy-reviewers", "members.read"
			}
			accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "manager", action, object, permissions.EffectDeny), "", 200)
			assignmentSignInAs(t, c, "manager")
			result := accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/health", "catalog.read"), nil, "", 200)
			restricted := 0
			for _, source := range explanationItems(result) {
				if source["restricted"] == true {
					restricted++
					if source["principalId"] != nil || source["roleId"] != nil || source["grantId"] != nil || source["expiresAt"] != nil {
						t.Fatal("source object deny leaked details", source)
					}
				}
			}
			want := 1
			if kind == "group" {
				want = 2
			}
			if restricted != want {
				t.Fatal(kind, "unexpected restricted path count", result)
			}
		})
	}
}

type changingExplanationStore struct {
	permissions.PermissionStore
	change func() error
}

func (s *changingExplanationStore) ReadGrant(ctx context.Context, id int64) (permissions.Grant, error) {
	return s.PermissionStore.(authorization.GrantReader).ReadGrant(ctx, id)
}

func (s *changingExplanationStore) ListGrants(ctx context.Context, q permissions.GrantQuery) (permissions.GrantQueryResult, error) {
	if s.change != nil {
		change := s.change
		s.change = nil
		if err := change(); err != nil {
			return permissions.GrantQueryResult{}, err
		}
	}
	return s.PermissionStore.ListGrants(ctx, q)
}

func TestPermissionExplanationRejectsDirectoryChangeDuringSourceCollection(t *testing.T) {
	c := roleSetup(t)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
	store := &changingExplanationStore{PermissionStore: c.api.Auth.Engine.GetStore(), change: func() error {
		return c.api.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "member", Name: "Changed during source lookup"}, 1)
	}}
	engine, err := authorization.NewEngine(t.Context(), store, c.api.Auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	c.api.Auth.Engine = engine
	path := explanationURL("member", "alpha", "suite/health", "catalog.read")
	accessCall(t, c, "GET", path, nil, "", 409)
	result := accessCall(t, c, "GET", path, nil, "", 200)
	if result["userName"] != "Changed during source lookup" || result["allowed"] != true {
		t.Fatal(result)
	}
}

func TestPermissionExplanationErrorsDoNotRevealAnUnreadableRole(t *testing.T) {
	c := roleSetup(t)
	private := "private-beta-binding-reader"
	accessCall(t, c, "POST", "/api/v1/roles", roleInput(private, "beta"), "", 200)
	parent := roleInput("private-beta-binding-parent", "beta")
	parent.Inherits = []string{private}
	accessCall(t, c, "POST", "/api/v1/roles", parent, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/beta/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": parent.ID, "reason": "Verify unreadable source error handling"}, "", 200)
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: private, PermissionName: "teststudio.catalog.read", TeamScope: "?team", VariableSpec: map[string]any{"team": "required"}, Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectDeny), "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	result := accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/health", "catalog.read"), nil, "", 422)
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), private) || strings.Contains(string(raw), parent.ID) {
		t.Fatal("failed source lookup leaked a private role", result)
	}
}
