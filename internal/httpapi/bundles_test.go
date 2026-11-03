package httpapi_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/sources"
)

func TestBundleAPIRequiresCSRFAndRetainsExactPrivateBytes(t *testing.T) {
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"go"}}
	})
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", map[string]string{"id": "bundle", "name": "Bundle"}, nil); status != 200 {
		t.Fatal(data)
	}
	base := "/api/v1/projects/bundle/suites/totals"
	if status, data := c.call("PUT", base, map[string]any{"id": "totals", "name": "Totals", "requiresTarget": false, "applications": []string{}}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	files := map[string]string{
		"teststudio-suite.json": `{"schemaVersion":1,"suiteId":"totals","type":"unit","runner":"go","applications":[],"moduleRoot":".","tests":{"packages":["./checks"]},"catalog":"catalog.json","buildProfile":"go-module"}`,
		"catalog.json":          `{"schemaVersion":"teststudio.catalog/v1","tests":[{"id":"BUNDLE-API-001","title":"Compute a total","category":"Unit","type":"unit","runner":"go","state":"enabled","safety":"safe","rationale":"Computes in memory","timeoutSeconds":5,"source":"checks/total_test.go","selector":"./checks::TestTotal"}]}`,
		"go.mod":                "module bundle.local/totals\n\ngo 1.24\n",
		"checks/total_test.go":  "package checks\nimport \"testing\"\nfunc TestTotal(t *testing.T) {}\n",
	}
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	upload := func(csrf string) (int, map[string]any) {
		r, err := http.NewRequest("POST", c.origin+base+"/source-bundles?name=source.zip&manifest=teststudio-suite.json", bytes.NewReader(buffer.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", "application/zip")
		r.Header.Set("Origin", c.origin)
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Idempotency-Key", "upload-once")
		response, err := c.http.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var row map[string]any
		if err := json.NewDecoder(response.Body).Decode(&row); err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, row
	}
	if status, _ := upload(""); status != 403 {
		t.Fatal("ZIP upload accepted missing CSRF")
	}
	status, row := upload(c.csrf)
	if status != 200 {
		t.Fatal(row)
	}
	if row["data"].(map[string]any)["state"] != "received" {
		t.Fatal("upload ran a build in the request")
	}
	status, replay := upload(c.csrf)
	if status != 200 || replay["id"] != row["id"] {
		t.Fatal("ZIP retry created another job")
	}
	response, err := c.http.Get(c.origin + base + "/source-preparations/" + row["id"].(string) + "/bundle")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Content-Type") != "application/zip" || !bytes.Equal(content, buffer.Bytes()) {
		t.Fatal("download changed bytes or allowed public caching")
	}
	anonymous, err := http.Get(c.origin + base + "/source-preparations/" + row["id"].(string) + "/bundle")
	if err != nil {
		t.Fatal(err)
	}
	anonymous.Body.Close()
	if anonymous.StatusCode != 401 {
		t.Fatal("anonymous bundle download allowed")
	}
}
