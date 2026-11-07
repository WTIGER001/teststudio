package httpapi_test

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestStandardRoleAPIDelegationReplacementAndOwnerGroupGuards(t *testing.T) {
	c := assignmentSetup(t)
	base := "/api/v1/projects/alpha/role-assignments"
	owner := accessCall(t, c, "POST", base, assignBody("user", "manager", "owner"), "", 200)
	accessCall(t, c, "POST", base, assignBody("user", "member", "contributor"), "", 200)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "owners", "name": "Owners", "projectId": "alpha"}, "", 200)
	accessCall(t, c, "POST", base, assignBody("group", "owners", "owner"), "", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/owners/members/manager", map[string]any{}, "0", 200)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "legacy-admins", "name": "Legacy administrators", "projectId": "alpha"}, "", 200)
	accessCall(t, c, "POST", base, assignBody("group", "legacy-admins", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "member")
	accessCall(t, c, "POST", base, assignBody("user", "member", "owner"), "", 403)
	accessCall(t, c, "DELETE", base+"/user/manager/"+authorization.OwnerRole, map[string]string{"reason": "Cannot remove Owner"}, assignmentRevision(owner), 403)
	accessCall(t, c, "PUT", base+"/user/manager", map[string]string{"previousRoleId": authorization.OwnerRole, "roleId": authorization.ReaderRole, "reason": "Cannot demote Owner"}, assignmentRevision(owner), 403)
	accessCall(t, c, "PUT", "/api/v1/groups/owners/members/member", map[string]any{}, "0", 403)
	accessCall(t, c, "POST", "/api/v1/groups/owners/archive", map[string]string{"reason": "Cannot remove group Owners"}, "1", 403)
	accessCall(t, c, "POST", base, assignBody("user", "manager", "operator"), "", 403)
	accessCall(t, c, "POST", base, assignBody("user", "member", "project-admin"), "", 403)
	accessCall(t, c, "PUT", "/api/v1/groups/legacy-admins/members/member", map[string]any{}, "0", 403)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "member", "permissions.grant", "", "allow"), "", 403)
	assignmentSignInAs(t, c, "manager")
	row := accessCall(t, c, "GET", base+"/user/member/"+authorization.ContributorRole, nil, "", 200)
	replaced := accessCall(t, c, "PUT", base+"/user/member", map[string]string{"previousRoleId": authorization.ContributorRole, "roleId": authorization.MemberRole, "reason": "Change execution role"}, assignmentRevision(row), 200)
	if replaced["data"].(map[string]any)["roleId"] != authorization.MemberRole {
		t.Fatal(replaced)
	}
	unchanged := accessCall(t, c, "PUT", base+"/user/member", map[string]string{"previousRoleId": authorization.MemberRole, "roleId": authorization.MemberRole, "reason": "Keep current role"}, assignmentRevision(replaced), 200)
	if unchanged["data"].(map[string]any)["canRevoke"] != true {
		t.Fatal("unchanged role lost management capability")
	}
	accessCall(t, c, "PUT", base+"/user/member", map[string]string{"previousRoleId": authorization.ContributorRole, "roleId": authorization.ReaderRole, "reason": "Reject stale replacement"}, assignmentRevision(row), 409)
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "runs.create"); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "projects.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
}

func TestApplicationRoleAPIEnforcesResourcesRunsAndRevocation(t *testing.T) {
	c, existingRun := preparedRun(t)
	accessCall(t, c, "PUT", "/api/v1/projects/example/applications/other", map[string]string{"id": "other", "name": "Other"}, "0", 200)
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "scoped", "name": "Scoped"}, "", 200)
	base := "/api/v1/projects/example/applications/app/role-assignments"
	assignment := accessCall(t, c, "POST", base, assignBody("user", "scoped", "member"), "", 200)
	assignmentSignInAs(t, c, "scoped")
	apps := accessCall(t, c, "GET", "/api/v1/projects/example/applications", nil, "", 200)
	if len(apps["items"].([]any)) != 1 {
		t.Fatal("sibling leaked", apps)
	}
	parent := accessCall(t, c, "GET", "/api/v1/projects/example", nil, "", 200)
	if len(parent["data"].(map[string]any)) != 2 {
		t.Fatal("project settings leaked", parent)
	}
	accessCall(t, c, "GET", "/api/v1/projects/example/role-assignments", nil, "", 403)
	caps := accessCall(t, c, "GET", "/api/v1/me/capabilities?project=example&application=app", nil, "", 200)
	if caps["teststudio.runs.create"] != true || caps["teststudio.catalog.publish"] != false {
		t.Fatal(caps)
	}
	explained := accessCall(t, c, "GET", "/api/v1/me/effective-permissions?project=example&application=app&permission=teststudio.runs.create", nil, "", 200)
	if explained["allowed"] != true || explained["applicationId"] != "app" || len(explained["items"].([]any)) == 0 {
		t.Fatal("scoped access explanation lost its source", explained)
	}
	accessCall(t, c, "GET", "/api/v1/me/capabilities?project=example&application=other", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/projects/example/runs/"+existingRun, nil, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/example/suites/health/tests", catalog.Test{ID: "DENIED"}, "", 403)
	status, created := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local", Application: "app"}, map[string]string{"Idempotency-Key": "scoped-run"})
	if status != 200 {
		t.Fatalf("scoped run: %d %v", status, created)
	}
	id := created["id"].(string)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		row := accessCall(t, c, "GET", "/api/v1/projects/example/runs/"+id, nil, "", 200)
		data := row["data"].(map[string]any)
		if runs.Terminal(data["state"].(string)) {
			if data["outcome"] != "passed" {
				t.Fatal(data)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	observed := accessCall(t, c, "GET", "/api/v1/projects/example/runs/"+id, nil, "", 200)
	if observed["data"].(map[string]any)["outcome"] != "passed" {
		t.Fatal("scoped execution did not finish")
	}
	artifacts := accessCall(t, c, "GET", "/api/v1/projects/example/runs/"+id+"/artifacts", nil, "", 200)
	if len(artifacts["items"].([]any)) == 0 {
		t.Fatal("no scoped evidence")
	}
	artifactID := artifacts["items"].([]any)[0].(map[string]any)["id"].(string)
	grant := accessCall(t, c, "POST", "/api/v1/projects/example/runs/"+id+"/artifacts/"+artifactID+"/access", map[string]string{"action": "download"}, "", 200)
	privateURL := grant["url"].(string)
	response, err := http.Get(privateURL)
	if err != nil {
		t.Fatal("scoped evidence request failed")
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("scoped evidence denied", response.StatusCode)
	}
	clock := time.Now().UTC()
	c.api.Schedules.Now = func() time.Time { return clock }
	schedule := accessCall(t, c, "POST", "/api/v1/projects/example/schedules", schedules.Definition{ID: "member-schedule", Name: "Member schedule", Enabled: true, Kind: "interval", IntervalSeconds: 5, Request: runs.Request{Suite: "health", Environment: "local", Application: "app"}}, "", 200)
	clock, _ = time.Parse(time.RFC3339Nano, schedule["data"].(map[string]any)["nextAt"].(string))
	if _, err := c.api.Schedules.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	fired := accessCall(t, c, "GET", "/api/v1/projects/example/schedules/member-schedule/firings", nil, "", 200)
	if len(fired["items"].([]any)) != 1 {
		t.Fatal("scoped schedule did not fire")
	}
	active := accessCall(t, c, "GET", "/api/v1/projects/example/schedules/member-schedule", nil, "", 200)
	clock, _ = time.Parse(time.RFC3339Nano, active["data"].(map[string]any)["nextAt"].(string))
	var mixed runs.Run
	row, err := c.api.Store.Get(t.Context(), "runs", existingRun)
	if err != nil {
		t.Fatal(err)
	}
	mixed, err = storage.Decode[runs.Run](row)
	if err != nil {
		t.Fatal(err)
	}
	mixed.ID = "mixed"
	mixed.Tests = append(mixed.Tests, catalog.Test{ID: "OTHER", Application: "other"})
	change, err := storage.Put("runs", mixed.ID, "example", 0, mixed)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "/logs", "/artifacts", "/attempts"} {
		accessCall(t, c, "GET", "/api/v1/projects/example/runs/mixed"+suffix, nil, "", 403)
	}
	listing := accessCall(t, c, "GET", "/api/v1/projects/example/runs", nil, "", 200)
	for _, raw := range listing["items"].([]any) {
		if raw.(map[string]any)["id"] == "mixed" {
			t.Fatal("mixed run leaked", listing)
		}
	}
	admin := *c
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	admin.http = &http.Client{Jar: jar}
	assignmentSignInAs(t, &admin, "admin")
	accessCall(t, &admin, "DELETE", base+"/user/scoped/"+authorization.MemberRole, map[string]string{"reason": "Revoke scoped execution"}, assignmentRevision(assignment), 200)
	accessCall(t, c, "GET", "/api/v1/me", nil, "", 200)
	accessCall(t, c, "GET", "/api/v1/projects/example/runs/"+id, nil, "", 403)
	response, err = http.Get(privateURL)
	if err != nil {
		t.Fatal("revoked evidence request failed")
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("old evidence access survived revocation", response.StatusCode)
	}
	status, _ = c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local"}, map[string]string{"Idempotency-Key": "revoked-run"})
	if status != 403 {
		t.Fatalf("revoked execution: %d", status)
	}
	if _, err := c.api.Schedules.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	fired = accessCall(t, &admin, "GET", "/api/v1/projects/example/schedules/member-schedule/firings", nil, "", 200)
	blocked := false
	for _, item := range fired["items"].([]any) {
		if item.(map[string]any)["data"].(map[string]any)["state"] == "blocked" {
			blocked = true
		}
	}
	if !blocked {
		t.Fatal("revoked schedule did not block")
	}
}

func TestApplicationDeveloperAPIProtectsSharedSuitesAndRepositorySources(t *testing.T) {
	provider := &apiRepositoryProvider{}
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"go"}, Providers: map[string]sourceprovider.Provider{"github": provider}}
	})
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "scopes", "name": "Scopes"}, "", 200)
	for _, app := range []string{"app", "other"} {
		accessCall(t, c, "PUT", "/api/v1/projects/scopes/applications/"+app, map[string]string{"id": app, "name": app}, "0", 200)
	}
	for _, suite := range []struct {
		id   string
		apps []string
	}{{"unit", []string{"app"}}, {"foreign", []string{"other"}}, {"shared", []string{"app", "other"}}} {
		accessCall(t, c, "PUT", "/api/v1/projects/scopes/suites/"+suite.id, map[string]any{"id": suite.id, "name": suite.id, "applications": suite.apps, "requiresTarget": false}, "0", 200)
	}
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "author", "name": "Author"}, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/scopes/applications/app/role-assignments", assignBody("user", "author", "developer"), "", 200)
	definition := catalog.Test{ID: "TEST-ONE", Title: "One", Application: "app", Category: "Unit", Type: "unit", Runner: "go", State: "enabled", Safety: "safe", Rationale: "Local unit example", TimeoutSeconds: 10, Source: "./..."}
	accessCall(t, c, "POST", "/api/v1/projects/scopes/suites/shared/tests", definition, "", 200)
	definition.ID = "TEST-TWO"
	definition.Application = "other"
	accessCall(t, c, "POST", "/api/v1/projects/scopes/suites/shared/tests", definition, "", 200)
	assignmentSignInAs(t, c, "author")
	definition.Application = "app"
	definition.ID = "OWN-TEST"
	accessCall(t, c, "POST", "/api/v1/projects/scopes/suites/unit/tests", definition, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/scopes/suites/shared/tests", definition, "", 403)
	accessCall(t, c, "POST", "/api/v1/projects/scopes/suites/foreign/tests", definition, "", 403)
	listed := accessCall(t, c, "GET", "/api/v1/projects/scopes/suites/shared/tests", nil, "", 200)
	if len(listed["items"].([]any)) != 1 {
		t.Fatal("shared catalog leaked sibling definition", listed)
	}
	accessCall(t, c, "PUT", "/api/v1/projects/scopes/suites/unit", map[string]any{"id": "unit", "name": "Unit", "applications": []string{"app", "other"}}, "1", 403)
	config := sources.RepositoryConfig{Name: "Scoped repository", Provider: "github", Repository: "owner/example", Reference: "heads/main", Enabled: true, Bindings: []sources.RepositoryBinding{{Suite: "unit", Manifest: "suites/unit.json", Runner: "go", Profile: "go-module"}}}
	source := accessCall(t, c, "PUT", "/api/v1/projects/scopes/repository-sources/scoped", config, "0", 200)
	accessCall(t, c, "GET", "/api/v1/projects/scopes/repository-sources/scoped", nil, "", 200)
	config.Bindings = append(config.Bindings, sources.RepositoryBinding{Suite: "foreign", Manifest: "suites/foreign.json", Runner: "go", Profile: "go-module"})
	accessCall(t, c, "PUT", "/api/v1/projects/scopes/repository-sources/scoped", config, assignmentRevision(source), 403)
	if provider.calls != 0 {
		t.Fatal("authorization check contacted source provider")
	}
}

func TestStandardRoleAPIConcurrentReplacementKeepsOneWinner(t *testing.T) {
	c := assignmentSetup(t)
	row := accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "member", "member"), "", 200)
	service := &access.Service{Store: c.api.Store, Auth: c.api.Auth, LocalUser: "admin"}
	var start sync.WaitGroup
	start.Add(1)
	results := make(chan error, 2)
	for _, role := range []string{authorization.DeveloperRole, authorization.ContributorRole} {
		go func() {
			start.Wait()
			_, err := service.ReplaceStandardAssignment(t.Context(), "admin", "alpha", access.AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "member", RoleID: role, PreviousRole: authorization.MemberRole, Reason: "Competing role change"}, int64(row["revision"].(float64)))
			results <- err
		}()
	}
	start.Done()
	wins := 0
	conflicts := 0
	for range 2 {
		err := <-results
		if err == nil {
			wins++
		} else if errors.Is(err, storage.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("role replacement: %d winners, %d conflicts", wins, conflicts)
	}
	assignments, err := c.api.Auth.Engine.RoleAssignmentsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "member"})
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 {
		t.Fatal("concurrent replacement combined roles", assignments)
	}
}
