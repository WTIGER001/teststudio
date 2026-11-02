package httpapi_test

import (
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestLegacyTimeoutViewsAndFailureRerunsStayConsistent(t *testing.T) {
	c, id := preparedRun(t)
	base := "/api/v1/projects/example/runs"
	var run runs.Run
	var row storage.Record
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		var err error
		run, row, err = c.api.Runs.Get(t.Context(), "example", id)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(run.State) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.State != "completed" || len(run.Results) != 1 {
		t.Fatalf("fixture run: %+v", run)
	}
	run.Results[0].State, run.Results[0].ObservedState, run.Results[0].FailureKind = "failed", "failed", "timeout"
	run.Outcome = "failed"
	run.Counts = runs.Counts{Total: 1, Failed: 1, Completed: 1}
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, run)
	if err != nil {
		t.Fatal(err)
	}
	attempt := run.Results[0]
	attemptRow, err := c.api.Store.Get(t.Context(), "attempts/"+id, "HEALTH-1.0001")
	if err != nil {
		t.Fatal(err)
	}
	attemptChange, err := storage.Put(attemptRow.Bucket, attemptRow.ID, attemptRow.Project, attemptRow.Revision, attempt)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.api.Store.Commit(t.Context(), change, attemptChange); err != nil {
		t.Fatal(err)
	}
	status, detail := c.call("GET", base+"/"+id, nil, nil)
	if status != 200 {
		t.Fatal(detail)
	}
	data := detail["data"].(map[string]any)
	counts := data["counts"].(map[string]any)
	result := data["results"].([]any)[0].(map[string]any)
	if data["outcome"] != "timed-out" || counts["timedOut"] != float64(1) || counts["failed"] != float64(0) || result["state"] != "timed-out" || result["observedState"] != "timed-out" {
		t.Fatalf("detail: %v", data)
	}
	status, list := c.call("GET", base, nil, nil)
	if status != 200 {
		t.Fatal(list)
	}
	summary := list["items"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if summary["outcome"] != "timed-out" || summary["counts"].(map[string]any)["timedOut"] != float64(1) {
		t.Fatalf("summary: %v", summary)
	}
	status, attempts := c.call("GET", base+"/"+id+"/attempts", nil, nil)
	if status != 200 || attempts["items"].([]any)[0].(map[string]any)["data"].(map[string]any)["state"] != "timed-out" {
		t.Fatalf("attempts: %v", attempts)
	}
	stored, err := c.api.Store.Get(t.Context(), row.Bucket, id)
	if err != nil {
		t.Fatal(err)
	}
	original, err := storage.Decode[runs.Run](stored)
	if err != nil || original.Outcome != "failed" || original.Results[0].State != "failed" {
		t.Fatal("reading history rewrote stored results")
	}
	status, child := c.call("POST", base+"/"+id+"/rerun-failures", map[string]any{}, map[string]string{"Idempotency-Key": "legacy-timeout-rerun"})
	if status != 200 {
		t.Fatalf("rerun: %d %v", status, child)
	}
	request := child["data"].(map[string]any)["request"].(map[string]any)
	if request["parentRunId"] != id || len(request["testIds"].([]any)) != 1 || request["testIds"].([]any)[0] != "HEALTH-1" {
		t.Fatalf("rerun selection: %v", request)
	}
	if status, _ := c.call("POST", "/api/v1/projects/other/runs/"+id+"/rerun-failures", map[string]any{}, map[string]string{"Idempotency-Key": "foreign-timeout-rerun"}); status != 404 {
		t.Fatal("timeout rerun crossed projects")
	}
}
