package httpapi_test

import "testing"

func TestCatalogListThroughAPIValidatesQueryAndCursorBindings(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	status, test := c.call("GET", base+"/tests/HEALTH-1", nil, nil)
	if status != 200 {
		t.Fatal(status, test)
	}
	definition := test["data"].(map[string]any)
	for _, id := range []string{"HEALTH-2", "HEALTH-3"} {
		copy := map[string]any{}
		for k, v := range definition {
			copy[k] = v
		}
		copy["id"] = id
		if status, _ := c.call("POST", base+"/tests", copy, nil); status != 200 {
			t.Fatal(status)
		}
	}
	list := "/api/v1/projects/example/tests?limit=1&state=active&sort=id&runner=http"
	status, page := c.call("GET", list, nil, nil)
	if status != 200 || page["matched"].(float64) != 3 || page["cursor"] == nil || page["options"] == nil || page["categories"] == nil {
		t.Fatal(status, page)
	}
	cursor := page["cursor"].(string)
	status, next := c.call("GET", list+"&cursor="+cursor, nil, nil)
	if status != 200 || next["items"].([]any)[0].(map[string]any)["id"] != "HEALTH-2" {
		t.Fatal(status, next)
	}
	if status, _ := c.call("GET", list+"&query=changed&cursor="+cursor, nil, nil); status != 422 {
		t.Fatal("changed-query cursor", status)
	}
	for _, query := range []string{"limit=0", "limit=101", "limit=bad", "descendants=bad", "sort=unknown", "state=unknown"} {
		if status, _ := c.call("GET", "/api/v1/projects/example/tests?"+query, nil, nil); status != 422 {
			t.Fatal(query, status)
		}
	}
	status, suite := c.call("GET", base+"/tests?limit=1&category=Health", nil, nil)
	if status != 200 || suite["matched"].(float64) != 3 {
		t.Fatal(status, suite)
	}
	if status, _ := c.call("GET", base+"/tests?suite=foreign", nil, nil); status != 422 {
		t.Fatal(status)
	}
}
