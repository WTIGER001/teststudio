package httpapi_test

import (
	"maps"
	"net/url"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestTrendAPIHandlesScopePaginationFiltersAndRevocation(t *testing.T) {
	c, id := preparedRun(t)
	var original runs.Run
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		var err error
		original, _, err = c.api.Runs.Get(t.Context(), "example", id)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(original.State) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if original.State != "completed" {
		t.Fatal("fixture did not finish")
	}
	for i, state := range []string{"failed", "passed", "timed-out"} {
		run := original
		run.ID = "trend-" + string(rune('a'+i))
		run.CreatedAt = original.CreatedAt.Add(time.Duration(i+1) * time.Minute)
		run.Results = []execution.Result{{TestID: "HEALTH-1", State: state, Attempt: 2}}
		run.Tests[0].Category = "Trend category"
		run.Counts = runs.Count(len(run.Tests), run.Results)
		change, err := storage.Put("runs", run.ID, "example", 0, run)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.api.Store.Commit(t.Context(), change); err != nil {
			t.Fatal(err)
		}
	}
	q := url.Values{"since": {original.CreatedAt.Add(-time.Hour).Format(time.RFC3339)}, "until": {original.CreatedAt.Add(time.Hour).Format(time.RFC3339)}, "limit": {"2"}}
	base := "/api/v1/projects/example/trends?"
	status, report := c.call("GET", base+q.Encode(), nil, nil)
	if status != 200 || len(report["points"].([]any)) != 2 || report["partial"] != true {
		t.Fatalf("first trend page: %d %v", status, report)
	}
	counts := report["totals"].(map[string]any)
	if counts["total"] != float64(2) || counts["timedOut"] != float64(1) || counts["passed"] != float64(1) {
		t.Fatalf("final counts: %v", counts)
	}
	cursor := report["cursor"].(string)
	q.Set("cursor", cursor)
	status, next := c.call("GET", base+q.Encode(), nil, nil)
	if status != 200 || len(next["points"].([]any)) != 2 || next["partial"] != false {
		t.Fatalf("older page: %d %v", status, next)
	}
	q.Set("category", "Trend category")
	if status, _ := c.call("GET", base+q.Encode(), nil, nil); status != 422 {
		t.Fatal("trend cursor crossed filter scope")
	}
	q.Del("cursor")
	q.Set("limit", "200")
	q.Set("suite", "health")
	q.Set("test", "HEALTH-1")
	status, filtered := c.call("GET", base+q.Encode(), nil, nil)
	if status != 200 || filtered["sampleCount"] != float64(3) {
		t.Fatalf("category and test scope: %d %v", status, filtered)
	}
	for key, value := range map[string]string{"since": "bad-date", "until": "2000-01-01T00:00:00Z", "limit": "501", "suite": ""} {
		bad := maps.Clone(q)
		bad.Set(key, value)
		if status, _ := c.call("GET", base+bad.Encode(), nil, nil); status != 422 {
			t.Fatalf("invalid %s was accepted", key)
		}
	}
	object := "example/run/trend-c"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", Effect: permissions.EffectDeny, TeamScope: "example", ObjectScope: &object, PermissionName: "teststudio.runs.read"}); err != nil {
		t.Fatal(err)
	}
	status, restricted := c.call("GET", base+q.Encode(), nil, nil)
	if status != 200 || restricted["sampleCount"] != float64(2) {
		t.Fatalf("object deny leaked a run: %d %v", status, restricted)
	}
	for _, point := range restricted["points"].([]any) {
		if point.(map[string]any)["runId"] == "trend-c" {
			t.Fatal("denied run reached trends")
		}
	}
	if status, _ := c.call("GET", strings.Replace(base, "example", "other", 1)+q.Encode(), nil, nil); status != 404 {
		t.Fatal("trend history crossed projects")
	}
	if err := c.api.Auth.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("GET", base+q.Encode(), nil, nil); status != 403 {
		t.Fatal("trend history ignored revoked permission")
	}
}
