package httpapi_test

import (
	"fmt"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"testing"
	"time"
)

func TestTimingEstimateBaselineReadAndReasonedResetThroughAPI(t *testing.T) {
	c, id := preparedRun(t)
	var run runs.Run
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		row, err := c.api.Store.Get(t.Context(), "runs", id)
		if err != nil {
			t.Fatal(err)
		}
		run, err = storage.Decode[runs.Run](row)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(run.State) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Outcome != "passed" {
		t.Fatal("initial run did not pass")
	}
	before, err := c.api.Store.All(t.Context(), "runs", "example")
	if err != nil {
		t.Fatal(err)
	}
	status, data := c.call("POST", "/api/v1/projects/example/runs/estimate", runs.Request{Suite: "health", Environment: "local"}, nil)
	if status != 200 || data["measuredTests"] != float64(1) || data["readyTests"] != float64(0) || data["expectedExecutionMs"] == nil {
		t.Fatalf("estimate: %d %v", status, data)
	}
	after, err := c.api.Store.All(t.Context(), "runs", "example")
	if err != nil || len(before) != len(after) {
		t.Fatal("estimate created a run")
	}
	p := run.Timing.Tests["HEALTH-1"]
	base := "/api/v1/projects/example/timing-baselines/" + p.Key
	status, data = c.call("GET", base, nil, nil)
	if status != 200 {
		t.Fatal(data)
	}
	rev := int64(data["revision"].(float64))
	input := map[string]string{"reason": "Recalibrate the local target"}
	if status, _ := c.call("POST", base+"/reset", input, nil); status != 422 {
		t.Fatal("reset accepted missing revision")
	}
	if status, _ := c.call("POST", base+"/reset", map[string]string{"reason": " "}, map[string]string{"If-Match": fmt.Sprint(rev)}); status != 422 {
		t.Fatal("reset accepted empty reason")
	}
	status, data = c.call("POST", base+"/reset", input, map[string]string{"If-Match": fmt.Sprint(rev)})
	if status != 200 {
		t.Fatal(data)
	}
	window := data["data"].(map[string]any)
	if window["generation"] != float64(2) || len(window["samples"].([]any)) != 0 {
		t.Fatal("baseline not reset")
	}
	if status, _ := c.call("POST", base+"/reset", input, map[string]string{"If-Match": fmt.Sprint(rev)}); status != 409 {
		t.Fatal("stale reset accepted")
	}
	if status, _ := c.call("GET", "/api/v1/projects/other/timing-baselines/"+p.Key, nil, nil); status != 404 {
		t.Fatal("baseline crossed project")
	}
	if err := c.api.Auth.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("POST", "/api/v1/projects/example/runs/estimate", runs.Request{Suite: "health", Environment: "local"}, nil); status != 403 {
		t.Fatal("estimate ignored revocation")
	}
	if status, _ := c.call("POST", base+"/reset", input, map[string]string{"If-Match": fmt.Sprint(rev + 1)}); status != 403 {
		t.Fatal("reset ignored revocation")
	}
	if status, _ := c.call("GET", base, nil, nil); status != 403 {
		t.Fatal("baseline read ignored revocation")
	}
}
