package httpapi_test

import (
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/storage"
	"strconv"
	"testing"
	"time"
)

func TestRetainedHistoryAPIWorksAfterSuiteArchiveAndWorkingRemoval(t *testing.T) {
	c, id := preparedRun(t)
	project := "/api/v1/projects/example"
	suite := project + "/suites/health"
	base := suite + "/tests/HEALTH-1"
	deadline := time.Now().Add(5 * time.Second)
	for {
		status, run := c.call("GET", project+"/runs/"+id, nil, nil)
		if status != 200 {
			t.Fatal(status, run)
		}
		if run["data"].(map[string]any)["state"] == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture run did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status, _ := c.call("POST", base+"/actions/archive", map[string]any{}, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status)
	}
	status, current := c.call("GET", suite+"/lifecycle", nil, nil)
	if status != 200 {
		t.Fatal(status, current)
	}
	if status, value := c.call("POST", suite+"/actions/archive", nil, map[string]string{"If-Match": strconv.FormatInt(int64(current["revision"].(float64)), 10)}); status != 200 {
		t.Fatal(status, value)
	}
	status, retained := c.call("GET", base+"/retained", nil, nil)
	if status != 200 || retained["suiteArchived"] != true || retained["lifecycle"] != "archived" {
		t.Fatal(status, retained)
	}
	status, history := c.call("GET", base+"/run-history?limit=1", nil, nil)
	if status != 200 || len(history["items"].([]any)) != 1 || history["items"].([]any)[0].(map[string]any)["runId"] != id {
		t.Fatal(status, history)
	}
	row, err := c.api.Store.Get(t.Context(), catalog.Bucket("tests", "example", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.api.Store.Commit(t.Context(), storage.Change{Record: row, Delete: true}); err != nil {
		t.Fatal(err)
	}
	status, retained = c.call("GET", base+"/retained", nil, nil)
	if status != 200 || retained["current"] != false || retained["lifecycle"] != "removed" {
		t.Fatal(status, retained)
	}
	if status, _ := c.call("GET", base+"/run-history?limit=101", nil, nil); status != 422 {
		t.Fatal(status)
	}
	if status, _ := c.call("GET", "/api/v1/projects/foreign/suites/health/tests/HEALTH-1/retained", nil, nil); status != 404 {
		t.Fatal(status)
	}
}
