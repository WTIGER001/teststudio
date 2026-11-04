package httpapi_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestRunEvidenceFiltersBindTestAttemptAndBoundedCursor(t *testing.T) {
	c, id := preparedRun(t)
	changes := []storage.Change{}
	for i := 0; i < 225; i++ {
		value := execution.Result{TestID: "OTHER", Attempt: 1, State: "passed"}
		if i >= 210 {
			value.TestID = "SELECTED"
			value.Attempt = 1 + i%2
		}
		for _, kind := range []string{"attempts", "artifacts", "events"} {
			change, err := storage.Put(kind+"/"+id, fmt.Sprintf("filter-%03d", i), "example", 0, value)
			if err != nil {
				t.Fatal(err)
			}
			changes = append(changes, change)
		}
	}
	for start := 0; start < len(changes); start += 100 {
		if err := c.api.Store.Commit(t.Context(), changes[start:min(start+100, len(changes))]...); err != nil {
			t.Fatal(err)
		}
	}
	base := "/api/v1/projects/example/runs/" + id
	for _, kind := range []string{"attempts", "artifacts", "events"} {
		path := base + "/" + kind + "?test=SELECTED&attempt=2&limit=3"
		status, page := c.call("GET", path, nil, nil)
		if status != 200 || len(page["items"].([]any)) != 0 || page["cursor"] == nil {
			t.Fatalf("bounded filtered scan %s: %d %v", kind, status, page)
		}
		cursor := page["cursor"].(string)
		if status, data := c.call("GET", base+"/"+kind+"?test=OTHER&attempt=2&limit=3&cursor="+url.QueryEscape(cursor), nil, nil); status != 422 {
			t.Fatal("cursor accepted another filter", status, data)
		}
		status, next := c.call("GET", path+"&cursor="+url.QueryEscape(cursor), nil, nil)
		if status != 200 || len(next["items"].([]any)) != 3 {
			t.Fatal("filtered continuation", kind, status, next)
		}
		for _, entry := range next["items"].([]any) {
			value := entry.(map[string]any)["data"].(map[string]any)
			if value["testId"] != "SELECTED" || value["attempt"] != float64(2) {
				t.Fatal("unrelated attempt leaked into filtered page", value)
			}
		}
	}
	for _, query := range []string{"attempt=1", "test=SELECTED&attempt=0", "test=SELECTED&attempt=4", "test=../foreign", "test=SELECTED&attempt=x"} {
		for _, kind := range []string{"attempts", "artifacts", "logs"} {
			if status, data := c.call("GET", base+"/"+kind+"?"+query, nil, nil); status != 422 {
				t.Fatal("invalid filter accepted", query, status, data)
			}
		}
	}
}
