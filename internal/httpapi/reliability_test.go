package httpapi_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestReliabilityReportRequiresSourceAuthorityAndExplicitMonitoring(t *testing.T) {
	c, run := preparedRun(t)
	for range 100 {
		v, _, err := c.api.Runs.Get(t.Context(), "example", run)
		if err != nil {
			t.Fatal(err)
		}
		if v.State == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	base := "/api/v1/projects/example/suites/health/tests/HEALTH-1/reliability"
	checks := base + "/checks?environment=local&since=" + url.QueryEscape(time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)) + "&until=" + url.QueryEscape(time.Now().UTC().Format(time.RFC3339Nano))
	individual := accessCall(t, c, "GET", checks, nil, "", 200)["items"].([]any)
	if len(individual) != 1 || individual[0].(map[string]any)["runId"] != run || individual[0].(map[string]any)["state"] != "passed" {
		t.Fatal("missing individual execution", individual)
	}
	accessCall(t, c, "GET", base+"/checks", nil, "", 422)
	accessCall(t, c, "GET", checks+"&limit=101", nil, "", 422)
	report := accessCall(t, c, "GET", base+"?environment=local&window=1m", nil, "", 200)
	row := report["probes"].([]any)[0].(map[string]any)
	counts := row["totals"].(map[string]any)["counts"].(map[string]any)
	if row["metric"] != "outcomes" || counts["passed"] != float64(1) || row["passRate"] != float64(100) || row["availability"].(map[string]any)["percent"] != nil {
		t.Fatal(row)
	}
	monitor := accessCall(t, c, "GET", base+"?environment=local&maxAgeSeconds=900", nil, "", 200)["probes"].([]any)[0].(map[string]any)
	if monitor["metric"] != "availability" || monitor["availability"].(map[string]any)["upSeconds"].(float64) <= 0 {
		t.Fatal(monitor)
	}
	accessCall(t, c, "GET", base+"?maxAgeSeconds=1", nil, "", 422)
	accessCall(t, c, "GET", base+"?since=2026-01-01T00:00:00Z", nil, "", 422)
	accessCall(t, c, "GET", base+"?environment=missing", nil, "", 404)
	anon := *c
	anon.http = &http.Client{}
	accessCall(t, &anon, "GET", base, nil, "", 401)
	accessCall(t, &anon, "GET", checks, nil, "", 401)
	envRow, err := c.api.Store.Get(t.Context(), projects.Bucket("environments", "example"), "local")
	if err != nil {
		t.Fatal(err)
	}
	env, err := storage.Decode[projects.Environment](envRow)
	if err != nil {
		t.Fatal(err)
	}
	env.Endpoints["api"] = "https://changed.example.test"
	if err = c.api.Projects.SaveEnvironment(t.Context(), "admin", "example", env, envRow.Revision); err != nil {
		t.Fatal(err)
	}
	changed := accessCall(t, c, "GET", base+"?environment=local&maxAgeSeconds=900", nil, "", 200)["probes"].([]any)[0].(map[string]any)
	if changed["availability"].(map[string]any)["percent"] != nil {
		t.Fatal("old target became current availability", changed)
	}
	object := "example/run/" + run
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "example", ObjectScope: &object, PermissionName: "teststudio.runs.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if items := accessCall(t, c, "GET", checks, nil, "", 200)["items"].([]any); len(items) != 0 {
		t.Fatal("denied individual results leaked", len(items))
	}
	denied := accessCall(t, c, "GET", base+"?environment=local&window=1m", nil, "", 200)["probes"].([]any)[0].(map[string]any)
	if denied["totals"].(map[string]any)["counts"].(map[string]any)["passed"] != float64(0) || denied["passRate"] != nil || len(denied["history"].([]any)) != 0 {
		t.Fatal("denied results leaked through summaries", denied)
	}
}
