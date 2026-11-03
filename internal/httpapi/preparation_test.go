package httpapi_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type apiInspector struct{ calls int }

func (p *apiInspector) Reference(ref execution.Reference) execution.Reference {
	ref.Profile = "go-local-isolated-v1"
	ref.Container = "inspect-" + ref.ID
	ref.Volume = "inspect-output-" + ref.ID
	return ref
}
func (p *apiInspector) CleanupInspection(context.Context, preparation.InspectionRequest) error {
	return nil
}
func (p *apiInspector) Inspect(_ context.Context, input preparation.InspectionRequest) (preparation.Inspection, error) {
	p.calls++
	test := catalog.Test{ID: "API-CODE-1", Title: "API code check", Category: "Unit", Runner: "go", Type: "unit", State: "enabled", Safety: "safe", Rationale: "Computes in memory", TimeoutSeconds: 5, Source: "checks/api_test.go", Selector: "./checks::TestApi"}
	return preparation.Inspection{Image: input.Image, Project: input.Reference.Project, Discovery: suitespec.Discovery{Schema: suitespec.DiscoverySchema, Manifest: suitespec.Manifest{Schema: 1, Suite: input.Suite, Runner: "go", Type: "unit", ModuleRoot: ".", Catalog: "catalog.json", Tests: suitespec.Tests{Packages: []string{"./checks"}}, BuildProfile: "go-module"}, Origin: suitespec.Origin{SnapshotDigest: "sha256:" + strings.Repeat("a", 64)}, Catalog: catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{test}}, Tests: []suitespec.DiscoveredTest{{ID: test.ID, Selector: test.Selector, File: test.Source, Line: 1}}}}, nil
}
func TestPrebuiltPreparationAPIQueuesAndPublishes(t *testing.T) {
	probe := &apiInspector{}
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", Inspectors: map[string]preparation.ArtifactInspector{"go": probe}}
	})
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", map[string]string{"id": "example", "name": "Example"}, nil); status != 200 {
		t.Fatal(data)
	}
	base := "/api/v1/projects/example/suites/code"
	if status, data := c.call("PUT", base, map[string]any{"id": "code", "name": "Code", "requiresTarget": false, "applications": []string{}}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	status, options := c.call("GET", base+"/source-preparation-options", nil, nil)
	if status != 200 || len(options["runners"].([]any)) != 1 {
		t.Fatal("preparation options unavailable")
	}
	input := map[string]string{"runner": "go", "image": "sha256:" + strings.Repeat("b", 64)}
	if status, _ := c.call("POST", base+"/source-preparations", input, map[string]string{"Idempotency-Key": "image-v1", "X-CSRF-Token": ""}); status != 403 {
		t.Fatal("preparation accepted a missing request token")
	}
	status, queued := c.call("POST", base+"/source-preparations", input, map[string]string{"Idempotency-Key": "image-v1"})
	if status != 200 {
		t.Fatal(queued)
	}
	status, replayed := c.call("POST", base+"/source-preparations", input, map[string]string{"Idempotency-Key": "image-v1"})
	if status != 200 || replayed["id"] != queued["id"] || probe.calls != 0 {
		t.Fatal("enqueue executed or duplicated preparation")
	}
	if processed, err := c.api.Sources.ProcessNext(t.Context(), "worker"); err != nil || !processed {
		t.Fatal("preparation worker failed", err)
	}
	status, ready := c.call("GET", base+"/source-preparations/"+queued["id"].(string), nil, nil)
	if status != 200 {
		t.Fatal(ready)
	}
	job := ready["data"].(map[string]any)
	if job["state"] != "ready" {
		t.Fatal(job)
	}
	upload := job["uploadId"].(string)
	status, release := c.call("POST", base+"/source-uploads/"+upload+"/publish", map[string]bool{"activate": true}, map[string]string{"If-Match": "1"})
	if status != 200 {
		t.Fatal(release)
	}
	status, stored := c.call("GET", base+"/source-preparations/"+queued["id"].(string), nil, nil)
	if status != 200 || stored["data"].(map[string]any)["state"] != "published" {
		t.Fatal("published job state missing")
	}
	if status, _ := c.call("POST", base+"/source-preparations/"+queued["id"].(string)+"/cancel", map[string]any{}, map[string]string{"If-Match": "4"}); status != 422 {
		t.Fatal("published preparation was cancelled")
	}
	status, runs := c.call("GET", "/api/v1/projects/example/runs", nil, nil)
	if status != 200 || len(runs["items"].([]any)) != 0 {
		t.Fatal("publication created a test run")
	}
}
