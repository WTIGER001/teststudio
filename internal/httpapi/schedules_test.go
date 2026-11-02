package httpapi_test

import (
	"strconv"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
)

func TestScheduleAPIValidatesRevisionsAndKeepsRunsAndHistoryScoped(t *testing.T) {
	c, _ := preparedRun(t)
	clock := time.Now().UTC()
	c.api.Schedules.Now = func() time.Time { return clock }
	base := "/api/v1/projects/example/schedules"
	d := schedules.Definition{ID: "health-probe", Name: "Health probe", Kind: "interval", IntervalSeconds: 5, Enabled: true, Request: runs.Request{Suite: "health", Environment: "local"}}
	status, preview := c.call("POST", base+"/preview", d, nil)
	if status != 200 || len(preview["next"].([]any)) != 5 {
		t.Fatalf("schedule preview: %d %v", status, preview)
	}
	status, row := c.call("POST", base, d, nil)
	if status != 200 {
		t.Fatalf("schedule creation: %d %v", status, row)
	}
	if status, _ := c.call("PUT", base+"/"+d.ID, d, nil); status != 422 {
		t.Fatal("schedule update omitted If-Match")
	}
	if status, _ := c.call("PUT", base+"/"+d.ID, d, map[string]string{"If-Match": "9"}); status != 409 {
		t.Fatal("stale schedule revision did not conflict")
	}
	if status, _ := c.call("GET", "/api/v1/projects/other/schedules/"+d.ID, nil, nil); status != 404 {
		t.Fatal("another project exposed a schedule")
	}
	clock, _ = time.Parse(time.RFC3339Nano, row["data"].(map[string]any)["nextAt"].(string))
	if _, err := c.api.Schedules.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	status, history := c.call("GET", base+"/"+d.ID+"/firings", nil, nil)
	if status != 200 || len(history["items"].([]any)) != 1 {
		t.Fatalf("firing history: %d %v", status, history)
	}
	firing := history["items"].([]any)[0].(map[string]any)["data"].(map[string]any)
	status, run := c.call("GET", "/api/v1/projects/example/runs/"+firing["runId"].(string), nil, nil)
	if status != 200 || run["data"].(map[string]any)["trigger"] != "schedule" {
		t.Fatalf("scheduled run missing: %d %v", status, run)
	}
	status, row = c.call("GET", base+"/"+d.ID, nil, nil)
	if status != 200 {
		t.Fatal(row)
	}
	revision := strconv.Itoa(int(row["revision"].(float64)))
	if status, data := c.call("POST", base+"/"+d.ID+"/archive", map[string]any{}, map[string]string{"If-Match": revision}); status != 200 {
		t.Fatal(data)
	}
	status, listing := c.call("GET", base, nil, nil)
	if status != 200 || len(listing["items"].([]any)) != 0 {
		t.Fatal("archived schedule remained in active list")
	}
	status, listing = c.call("GET", base+"?archived=true", nil, nil)
	if status != 200 || len(listing["items"].([]any)) != 1 {
		t.Fatal("archived schedule lost its identity")
	}
	if status, history = c.call("GET", base+"/"+d.ID+"/firings", nil, nil); status != 200 || len(history["items"].([]any)) != 1 {
		t.Fatal("archive erased firing history")
	}
	object := "example/schedule/" + d.ID
	grant := permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.schedules.read", TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("GET", base+"/"+d.ID+"/firings", nil, nil); status != 403 {
		t.Fatal("schedule history ignored object deny")
	}
	status, listing = c.call("GET", base+"?archived=true", nil, nil)
	if status != 200 || len(listing["items"].([]any)) != 0 {
		t.Fatal("schedule list exposed denied metadata")
	}
}
