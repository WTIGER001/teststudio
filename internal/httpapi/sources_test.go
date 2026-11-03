package httpapi_test

import (
	"encoding/json"
	"testing"
)

func TestNativeSourceUploadPublicationAndRetainedProvenanceThroughAPI(t *testing.T) {
	c, runID := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	status, document := c.call("GET", base+"/catalog-exports", nil, nil)
	if status != 200 {
		t.Fatal(document)
	}
	first := document["tests"].([]any)[0].(map[string]any)
	first["id"], first["title"] = "SOURCE-CHECK", "Native source health check"
	document["tests"] = []any{first}
	content, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	status, preview := c.call("POST", base+"/source-uploads", map[string]any{"name": "health.json", "format": "json", "content": string(content)}, nil)
	if status != 200 {
		t.Fatalf("stage: %d %v", status, preview)
	}
	upload := preview["upload"].(map[string]any)
	id := upload["id"].(string)
	if preview["plan"].(map[string]any)["changes"].([]any)[0].(map[string]any)["action"] != "create" {
		t.Fatal("missing discovery")
	}
	if status, _ := c.call("GET", base+"/tests/SOURCE-CHECK", nil, nil); status != 404 {
		t.Fatal("staging applied catalog")
	}
	if status, _ := c.call("POST", base+"/catalog-imports/"+preview["plan"].(map[string]any)["id"].(string)+"/apply", map[string]any{}, map[string]string{"If-Match": "1"}); status != 403 {
		t.Fatal("source plan bypassed atomic publication")
	}
	endpoint := base + "/source-uploads/" + id + "/publish"
	if status, _ := c.call("POST", endpoint, map[string]any{}, map[string]string{"If-Match": "1"}); status != 422 {
		t.Fatal("implicit activation choice accepted")
	}
	status, release := c.call("POST", endpoint, map[string]any{"activate": false}, map[string]string{"If-Match": "1"})
	if status != 200 {
		t.Fatalf("publish: %d %v", status, release)
	}
	artifact := release["catalog"].(map[string]any)["artifact"].(map[string]any)
	if artifact["sourceSnapshotId"] != id || artifact["sourceRevision"] != upload["data"].(map[string]any)["digest"] {
		t.Fatal("missing immutable provenance")
	}
	status, again := c.call("POST", endpoint, map[string]any{"activate": false}, map[string]string{"If-Match": "1"})
	if status != 200 || again["id"] != release["id"] {
		t.Fatal("publication retry changed release")
	}
	status, stored := c.call("GET", base+"/source-uploads/"+id, nil, nil)
	if status != 200 || stored["upload"].(map[string]any)["data"].(map[string]any)["state"] != "published" {
		t.Fatal("published upload missing")
	}
	status, source := c.call("GET", base+"/source-uploads/"+id+"/content", nil, nil)
	if status != 200 || source["content"] != string(content) {
		t.Fatal("retained bytes differ")
	}
	status, history := c.call("GET", base+"/source-uploads?limit=1", nil, nil)
	if status != 200 || len(history["items"].([]any)) != 1 {
		t.Fatal("source history missing")
	}
	status, run := c.call("GET", "/api/v1/projects/example/runs/"+runID, nil, nil)
	if status != 200 || len(run["data"].(map[string]any)["tests"].([]any)) != 1 {
		t.Fatal("publication changed historical run")
	}
	if status, _ := c.call("GET", "/api/v1/projects/example/suites/unrelated/source-uploads/"+id+"/content", nil, nil); status != 404 {
		t.Fatal("source content escaped suite")
	}
}

func TestSourceUploadRejectsMalformedBodiesAndForgedProvenance(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	for _, input := range []map[string]any{
		{"name": "../escape.yaml", "format": "yaml", "content": "tests: []"},
		{"name": "health.yaml", "format": "zip", "content": "tests: []"},
		{"name": "health.yaml", "format": "yaml", "content": "tests: []", "unexpected": true},
	} {
		if status, _ := c.call("POST", base+"/source-uploads", input, nil); status != 422 {
			t.Fatalf("malformed upload accepted: %d", status)
		}
	}
	if status, _ := c.call("GET", base+"/source-uploads?limit=1001", nil, nil); status != 422 {
		t.Fatal("unbounded source scan")
	}
	if status, _ := c.call("POST", base+"/catalog-drafts", map[string]any{"sourceSnapshotId": "missing", "sourceRevision": "forged"}, nil); status != 404 {
		t.Fatal("forged provenance accepted")
	}
}
