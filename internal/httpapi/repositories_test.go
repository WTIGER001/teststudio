package httpapi_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
)

type apiRepositoryProvider struct{ calls int }

func (p *apiRepositoryProvider) Validate(sourceprovider.Access, string) error { return nil }
func (p *apiRepositoryProvider) Resolve(context.Context, sourceprovider.Access, string) (sourceprovider.Revision, error) {
	p.calls++
	return sourceprovider.Revision{}, sourceprovider.ErrUnavailable
}
func (p *apiRepositoryProvider) Archive(context.Context, sourceprovider.Access, sourceprovider.Revision) (sourceprovider.Archive, error) {
	p.calls++
	return sourceprovider.Archive{}, sourceprovider.ErrUnavailable
}

func TestRepositoryAPIRegistersAndQueuesWithoutRemoteRequests(t *testing.T) {
	provider := &apiRepositoryProvider{}
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"go"}, Providers: map[string]sourceprovider.Provider{"github": provider}}
	})
	c.signIn()
	if status, value := c.call("POST", "/api/v1/projects", map[string]string{"id": "repository", "name": "Repository"}, nil); status != 200 {
		t.Fatal(value)
	}
	project := "/api/v1/projects/repository"
	if status, value := c.call("PUT", project+"/suites/unit", map[string]any{"id": "unit", "name": "Unit", "requiresTarget": false, "applications": []string{}}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(value)
	}
	config := map[string]any{"name": "Project repository", "provider": "github", "repository": "owner/suite", "reference": "heads/main", "enabled": true, "autoActivate": false, "bindings": []map[string]string{{"suiteId": "unit", "manifestPath": "suites/unit.json", "runner": "go", "buildProfile": "go-module"}}}
	base := project + "/repository-sources/source"
	if status, _ := c.call("PUT", base, config, map[string]string{"If-Match": "0", "X-CSRF-Token": ""}); status != 403 {
		t.Fatal("repository configuration skipped CSRF protection")
	}
	status, source := c.call("PUT", base, config, map[string]string{"If-Match": "0"})
	if status != 200 {
		t.Fatal(source)
	}
	if provider.calls != 0 {
		t.Fatal("registration contacted a remote provider")
	}
	status, options := c.call("GET", project+"/repository-source-options", nil, nil)
	if status != 200 || len(options["providers"].([]any)) != 1 || len(options["credentials"].([]any)) != 0 {
		t.Fatal("repository options exposed credentials or lost enabled providers")
	}
	if status, _ := c.call("POST", base+"/syncs", map[string]bool{"force": false}, nil); status != 422 {
		t.Fatal("sync accepted a missing idempotency key")
	}
	status, queued := c.call("POST", base+"/syncs", map[string]bool{"force": false}, map[string]string{"Idempotency-Key": "manual-sync"})
	if status != 200 {
		t.Fatal(queued)
	}
	status, repeated := c.call("POST", base+"/syncs", map[string]bool{"force": false}, map[string]string{"Idempotency-Key": "manual-sync"})
	if status != 200 || repeated["id"] != queued["id"] || provider.calls != 0 {
		t.Fatal("request duplicated synchronization or executed remote work")
	}
	if processed, err := c.api.Sources.ProcessRepositorySync(t.Context(), "source-worker"); err != nil || !processed {
		t.Fatal("queued source worker", processed, err)
	}
	status, finished := c.call("GET", base+"/syncs/"+queued["id"].(string), nil, nil)
	if status != 200 || finished["data"].(map[string]any)["state"] != "failed" {
		t.Fatal("provider failure became a successful sync")
	}
	status, runs := c.call("GET", project+"/runs", nil, nil)
	if status != 200 || len(runs["items"].([]any)) != 0 {
		t.Fatal("source failure created a test run")
	}
	status, current := c.call("GET", base, nil, nil)
	if status != 200 {
		t.Fatal(current)
	}
	config["enabled"] = false
	if status, value := c.call("PUT", base, config, map[string]string{"If-Match": jsonRevision(current)}); status != 200 {
		t.Fatal(value)
	}
	if status, value := c.call("POST", base+"/syncs", map[string]bool{"force": false}, map[string]string{"Idempotency-Key": "disabled"}); status != 422 || !strings.Contains(value["message"].(string), "disabled") {
		t.Fatal("disabled source accepted synchronization")
	}
}
func jsonRevision(row map[string]any) string {
	return strconv.FormatInt(int64(row["revision"].(float64)), 10)
}
