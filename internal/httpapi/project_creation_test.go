package httpapi_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestProjectCreatorAPIKeepsInitialOwnershipAndForeignResourcesScoped(t *testing.T) {
	c, foreignRun := preparedRun(t)
	artifact, err := c.api.Evidence.SaveBytes(t.Context(), evidence.Artifact{Project: "example", Run: foreignRun, Kind: "log", Filename: "foreign.log", ContentType: "text/plain"}, []byte("Private foreign evidence"), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"creator", "member"} {
		accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": user, "name": user}, "", 200)
	}
	for _, group := range []map[string]string{{"id": "foreign", "name": "Foreign group", "projectId": "example"}, {"id": "shared", "name": "Installation group"}} {
		accessCall(t, c, "POST", "/api/v1/groups", group, "", 200)
	}
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "creator", PermissionName: "teststudio.projects.create", TeamScope: "*", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	assignmentSignInAs(t, c, "creator")
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "owned", "name": "Owned"}, "", 200)
	capabilities := accessCall(t, c, "GET", "/api/v1/me/capabilities?project=owned", nil, "", 200)
	for _, permission := range []string{"projects.manage", "environments.manage", "suites.manage", "sources.manage", "catalog.create", "catalog.publish", "runs.create", "members.manage"} {
		if capabilities["teststudio."+permission] != true {
			t.Fatal("creator cannot finish setup", permission, capabilities)
		}
	}
	accessCall(t, c, "PUT", "/api/v1/projects/owned/applications/app", map[string]string{"id": "app", "name": "App"}, "0", 200)
	accessCall(t, c, "PUT", "/api/v1/projects/owned/suites/health", map[string]any{"id": "health", "name": "Health", "applications": []string{"app"}, "requiresTarget": true}, "0", 200)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "own", "name": "Owned group", "projectId": "owned"}, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/owned/role-assignments", assignBody("user", "member", "viewer"), "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/owned/role-assignments", assignBody("group", "own", "operator"), "", 200)
	for _, group := range []string{"shared", "foreign"} {
		accessCall(t, c, "POST", "/api/v1/projects/owned/role-assignments", assignBody("group", group, "operator"), "", 403)
	}
	accessCall(t, c, "POST", "/api/v1/projects/owned/role-assignments", assignBody("user", "creator", "installation-admin"), "", 422)
	accessCall(t, c, "POST", "/api/v1/role-assignments", assignBody("user", "creator", "installation-admin"), "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/example/role-assignments", assignBody("user", "creator", "project-admin"), "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/owned/permissions", directGrant("user", "creator", "system.manage", "", permissions.EffectAllow), "", 422)
	accessCall(t, c, "POST", "/api/v1/projects/example/permissions", directGrant("user", "creator", "runs.read", "", permissions.EffectAllow), "", 403)
	for _, group := range []string{"shared", "foreign"} {
		accessCall(t, c, "PUT", "/api/v1/groups/"+group, map[string]string{"id": group, "name": "Changed"}, "1", 403)
		accessCall(t, c, "PUT", "/api/v1/groups/"+group+"/members/member", map[string]any{}, "0", 403)
	}
	for _, suffix := range []string{"", "/attempts", "/events", "/logs", "/artifacts"} {
		accessCall(t, c, "GET", "/api/v1/projects/example/runs/"+foreignRun+suffix, nil, "", 403)
	}
	accessCall(t, c, "POST", "/api/v1/projects/example/runs/"+foreignRun+"/cancel", map[string]any{}, "", 403)
	denied := accessCall(t, c, "POST", "/api/v1/projects/example/runs/"+foreignRun+"/artifacts/"+artifact.ID+"/access", map[string]string{"action": "download"}, "", 403)
	if denied["url"] != nil {
		t.Fatal("foreign evidence grant leaked", denied)
	}
	accessCall(t, c, "GET", "/api/v1/projects/owned/runs/"+foreignRun, nil, "", 404)
	accessCall(t, c, "POST", "/api/v1/projects/owned/runs/"+foreignRun+"/artifacts/"+artifact.ID+"/access", map[string]string{"action": "view"}, "", 404)
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "example", "name": "Take over"}, "", 409)
	for _, permission := range []string{"system.manage", "users.manage", "providers.manage"} {
		if err := c.api.Auth.Check(t.Context(), "creator", "", "", permission); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("installation privilege widened", permission, err)
		}
	}
	rows, err := c.api.Store.All(t.Context(), "access-assignments", "example")
	if err != nil || len(rows) != 0 {
		t.Fatal("foreign assignments changed", rows, err)
	}
}

func TestProjectCreatorAPIRejectsUnusableBootstrapWithoutPartialRecords(t *testing.T) {
	c := fixture(t)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "creator", "name": "Creator"}, "", 200)
	assignmentSignInAs(t, c, "creator")
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "forbidden", "name": "Forbidden"}, "", 403)
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "creator", PermissionName: "teststudio.projects.create", TeamScope: "*", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Engine.DisableBuiltInRole(t.Context(), authorization.OwnerRole); err != nil {
		t.Fatal(err)
	}
	failure := accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "unusable", "name": "Unusable"}, "", 422)
	if !strings.Contains(failure["message"].(string), "no project was created") {
		t.Fatal(failure)
	}
	for _, project := range []string{"forbidden", "unusable"} {
		if _, err := c.api.Store.Get(t.Context(), "projects", project); !errors.Is(err, storage.ErrNotFound) {
			t.Fatal("partial project survived", project, err)
		}
		for _, bucket := range []string{"access-assignments", "audit"} {
			rows, err := c.api.Store.All(t.Context(), bucket, project)
			if err != nil || len(rows) != 0 {
				t.Fatal(bucket, rows, err)
			}
		}
	}
	if err := c.api.Auth.Engine.EnableBuiltInRole(t.Context(), authorization.OwnerRole); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "unusable", "name": "Usable after policy repair"}, "", http.StatusOK)
}
