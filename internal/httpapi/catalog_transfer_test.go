package httpapi_test

import (
	"testing"
)

func TestCatalogTransferBulkAndDuplicateThroughAPI(t *testing.T) {
	c, runID := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	status, document := c.call("GET", base+"/catalog-exports", nil, nil)
	if status != 200 || document["schemaVersion"] != "teststudio.catalog/v1" {
		t.Fatalf("export: %d %v", status, document)
	}
	first := document["tests"].([]any)[0].(map[string]any)
	first["title"] = "Imported title"
	second := map[string]any{}
	for key, value := range first {
		second[key] = value
	}
	second["id"] = "HEALTH-2"
	document["tests"] = []any{first, second}
	status, preview := c.call("POST", base+"/catalog-imports", document, nil)
	if status != 200 {
		t.Fatal(preview)
	}
	plan := preview["data"].(map[string]any)
	if len(plan["issues"].([]any)) != 0 {
		t.Fatal(plan)
	}
	if status, _ := c.call("GET", base+"/tests/HEALTH-2", nil, nil); status != 404 {
		t.Fatalf("preview changes live data: %d", status)
	}
	apply := base + "/catalog-imports/" + preview["id"].(string) + "/apply"
	for range 2 {
		if status, data := c.call("POST", apply, map[string]any{}, map[string]string{"If-Match": "1"}); status != 200 {
			t.Fatalf("apply: %d %v", status, data)
		}
	}
	if status, data := c.call("POST", base+"/tests/HEALTH-1/actions/duplicate", map[string]string{"id": "COPY-1"}, map[string]string{"If-Match": "2"}); status != 200 || data["data"].(map[string]any)["state"] != "disabled" {
		t.Fatalf("duplicate: %d %v", status, data)
	}
	status, preview = c.call("POST", base+"/tests/bulk-preview", map[string]any{"action": "archive", "tests": []any{map[string]any{"id": "HEALTH-1", "revision": 2}, map[string]any{"id": "HEALTH-2", "revision": 1}}}, nil)
	if status != 200 {
		t.Fatal(preview)
	}
	if status, data := c.call("POST", base+"/catalog-imports/"+preview["id"].(string)+"/apply", map[string]any{}, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatalf("bulk apply: %d %v", status, data)
	}
	status, data := c.call("GET", "/api/v1/projects/example/runs/"+runID, nil, nil)
	if status != 200 {
		t.Fatal(data)
	}
	snapshot := data["data"].(map[string]any)["tests"].([]any)
	if len(snapshot) != 1 || snapshot[0].(map[string]any)["title"] != "Availability" {
		t.Fatal("import changed an existing run snapshot")
	}
}
