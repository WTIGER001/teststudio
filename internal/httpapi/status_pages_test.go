package httpapi_test

import (
	"encoding/json"
	"fmt"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/statuspages"
)

func TestStatusPagesDraftPublicationPrivacyAndRevocation(t *testing.T) {
	c, run, binding := probeFixture(t)
	base := "/api/v1/projects/example/status-pages/public-health"
	definition := statuspages.Definition{ID: "public-health", Title: "Public service health", Background: "#ffffff", Window: "24h", MaxAge: 900, Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website", Description: "Public availability"}}}
	saved := accessCall(t, c, "PUT", base, definition, "0", 200)
	data := saved["data"].(map[string]any)
	token := data["publicToken"].(string)
	public := "/api/v1/public/status-pages/" + token
	anon := *c
	anon.http = &http.Client{}
	accessCall(t, &anon, "GET", public, nil, "", 404)
	accessCall(t, &anon, "GET", base, nil, "", 401)
	published := accessCall(t, c, "POST", base+"/publish", nil, "1", 200)
	report := accessCall(t, &anon, "GET", public, nil, "", 200)
	raw, _ := json.Marshal(report)
	for _, secret := range []string{binding.ID, run, "http://", "https://", "projectId", "environmentId", "userId", "runId", "HTTP", "publicToken"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("private data in public payload", secret, string(raw))
		}
	}
	if report["title"] != "Public service health" || report["probes"].([]any)[0].(map[string]any)["title"] != "Website" {
		t.Fatal(report)
	}
	for _, window := range []string{"24h", "7d", "1m", "1y", "all"} {
		accessCall(t, &anon, "GET", public+"?window="+window, nil, "", 200)
	}
	accessCall(t, &anon, "GET", public+"?window=bad", nil, "", 422)
	definition.Title = "Unpublished changes"
	definition.Background = "#001122"
	accessCall(t, c, "PUT", base, definition, fmt.Sprint(published["revision"]), 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["title"] != "Public service health" {
		t.Fatal("draft leaked")
	}
	accessCall(t, c, "POST", base+"/publish", nil, "1", 409)
	accessCall(t, c, "POST", base+"/unpublish", nil, "3", 200)
	accessCall(t, &anon, "GET", public, nil, "", 404)
	accessCall(t, c, "POST", base+"/publish", nil, "4", 200)
	if accessCall(t, &anon, "GET", public, nil, "", 200)["title"] != "Unpublished changes" {
		t.Fatal("republish missing")
	}
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "other", "name": "Other"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/projects/other/status-pages/public-health", definition, "0", 404)
	definition.Background = "url(javascript:alert(1))"
	accessCall(t, c, "PUT", base, definition, "5", 422)
	definition.Background = "#ffffff"
	definition.Probes = append(definition.Probes, definition.Probes[0])
	accessCall(t, c, "PUT", base, definition, "5", 422)
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "example", PermissionName: "teststudio.statuspages.publish", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, &anon, "GET", public, nil, "", 404)
}
func TestStatusPagesContributorManagementDoesNotGrantDeletion(t *testing.T) {
	c, _, binding := probeFixture(t)
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "status-manager", "name": "Status manager"}, "", 200)
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "status-manager"}, authorization.ContributorRole, map[string]any{"team": "example"}); err != nil {
		t.Fatal(err)
	}
	d := statuspages.Definition{ID: "contributor-page", Title: "Contributor page", Background: "#ffffff", Window: "24h", MaxAge: 900, Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website"}}}
	for _, action := range []string{"read", "manage", "publish"} {
		if err := c.api.Auth.Check(t.Context(), "status-manager", "example", "", "statuspages."+action); err != nil {
			t.Fatal(action, err)
		}
	}
	if err := c.api.Auth.Check(t.Context(), "status-manager", "example", "", "statuspages.delete"); err == nil {
		t.Fatal("Contributor may delete")
	}
	accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/contributor-page", d, "0", 200)
	accessCall(t, c, "DELETE", "/api/v1/projects/example/status-pages/contributor-page", nil, "0", 409)
	accessCall(t, c, "DELETE", "/api/v1/projects/example/status-pages/contributor-page", nil, "1", 200)
}

func TestStatusPagesHistoryIsCompleteAndTargetSpecific(t *testing.T) {
	c, id, binding := probeFixture(t)
	original, _, err := c.api.Runs.Get(t.Context(), "example", id)
	if err != nil {
		t.Fatal(err)
	}
	first := time.Now().UTC().Add(-500 * 24 * time.Hour)
	changes := []storage.Change{}
	for i := 0; i < 1005; i++ {
		run := original
		run.ID = fmt.Sprintf("history-%04d", i)
		run.CreatedAt = first.Add(time.Duration(i) * time.Minute)
		at := run.CreatedAt.Add(time.Second)
		run.FinishedAt = &at
		run.Results = append([]execution.Result{}, original.Results...)
		run.Results[0].FinishedAt = at
		change, err := storage.Put("runs", run.ID, "example", 0, run)
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, change)
	}
	if err = c.api.Store.CommitAll(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	d := statuspages.Definition{ID: "history-page", Title: "History", Background: "#ffffff", Window: "all", MaxAge: 900, Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website"}}}
	base := "/api/v1/projects/example/status-pages/history-page"
	accessCall(t, c, "PUT", base, d, "0", 200)
	published := accessCall(t, c, "POST", base+"/publish", nil, "1", 200)
	token := published["data"].(map[string]any)["publicToken"].(string)
	public := "/api/v1/public/status-pages/" + token
	report := accessCall(t, c, "GET", public, nil, "", 200)
	row := report["probes"].([]any)[0].(map[string]any)
	availability := row["availability"].(map[string]any)
	since, err := time.Parse(time.RFC3339Nano, availability["since"].(string))
	if err != nil || !since.Equal(first.Add(time.Second)) {
		t.Fatal("old history was truncated", since, err)
	}
	if len(row["history"].([]any)) != 30 || availability["upSeconds"].(float64) < 1005*60 {
		t.Fatal(row)
	}
	restricted := permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "example", PermissionName: "teststudio.runs.read", Effect: permissions.EffectDeny}
	object := "example/run/" + id
	restricted.ObjectScope = &object
	if err = c.api.Auth.Engine.CreateGrant(t.Context(), restricted); err != nil {
		t.Fatal(err)
	}
	recent := accessCall(t, c, "GET", public+"?window=24h", nil, "", 200)["probes"].([]any)[0].(map[string]any)
	if len(recent["history"].([]any)) != 0 || recent["availability"].(map[string]any)["percent"] != nil {
		t.Fatal("restricted observation leaked", recent)
	}
	envRow, err := c.api.Store.Get(t.Context(), projects.Bucket("environments", "example"), "local")
	if err != nil {
		t.Fatal(err)
	}
	env, _ := storage.Decode[projects.Environment](envRow)
	env.Endpoints["api"] = "https://different.example.test"
	if err = c.api.Projects.SaveEnvironment(t.Context(), "admin", "example", env, envRow.Revision); err != nil {
		t.Fatal(err)
	}
	changed := accessCall(t, c, "GET", public, nil, "", 200)["probes"].([]any)[0].(map[string]any)
	if changed["availability"].(map[string]any)["percent"] != nil {
		t.Fatal("previous target counted as current availability", changed)
	}
	for _, path := range []string{"/status/" + token, "/projects"} {
		response, err := c.http.Get(c.origin + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		policy := response.Header.Get("Content-Security-Policy")
		if strings.HasPrefix(path, "/status/") {
			if !strings.Contains(policy, "frame-ancestors *") || !strings.Contains(policy, "img-src 'self' data: https:") {
				t.Fatal(policy)
			}
		} else if !strings.Contains(policy, "frame-ancestors 'none'") || !strings.Contains(policy, "img-src 'self' data: blob: https:") {
			t.Fatal(policy)
		}
	}
}

func TestStatusPagesEnforceContributorAndApplicationBoundaries(t *testing.T) {
	c, _, binding := probeFixture(t)
	for _, user := range []string{"scoped-manager", "app-manager"} {
		accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": user, "name": user}, "", 200)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "scoped-manager"}, authorization.ContributorRole, map[string]any{"team": "example"}); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "app-manager"}, authorization.ContributorRole, map[string]any{"team": "example", "application": "app"}); err != nil {
		t.Fatal(err)
	}
	sessionClient := func(user string) *client {
		recorder := httptest.NewRecorder()
		session, err := c.api.Sessions.Create(t.Context(), recorder, user, "isolated-status-check", time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		jar, _ := cookiejar.New(nil)
		origin, _ := url.Parse(c.origin)
		jar.SetCookies(origin, recorder.Result().Cookies())
		copy := *c
		copy.http = &http.Client{Jar: jar}
		copy.csrf = session.CSRF
		return &copy
	}
	manager := sessionClient("scoped-manager")
	app := sessionClient("app-manager")
	d := statuspages.Definition{ID: "managed-page", Title: "Managed", Background: "#ffffff", Window: "24h", MaxAge: 900, Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website"}}}
	base := "/api/v1/projects/example/status-pages/managed-page"
	accessCall(t, manager, "PUT", base, d, "0", 200)
	published := accessCall(t, manager, "POST", base+"/publish", nil, "1", 200)
	token := published["data"].(map[string]any)["publicToken"].(string)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/"+token, nil, "", 200)
	accessCall(t, manager, "DELETE", base, nil, "2", 403)
	accessCall(t, manager, "POST", base+"/unpublish", nil, "2", 200)
	accessCall(t, app, "GET", base, nil, "", 403)
	accessCall(t, app, "PUT", base, d, "3", 403)
	accessCall(t, manager, "GET", "/api/v1/projects/foreign/status-pages/managed-page", nil, "", 404)
}
