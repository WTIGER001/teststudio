package httpapi_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestSavedSelectionsFreezeReviewedReleaseAndRejectChangedDefinitions(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	selections := base + "/saved-selections"
	definition := catalog.SavedSelection{ID: "daily", Name: "Daily health", Spec: catalog.SavedSelectionSpec{Version: 1, Kind: "explicit", IDs: []string{"HEALTH-1"}}}
	if status, value := c.call("POST", selections, definition, nil); status != 200 {
		t.Fatal(status, value)
	}
	if status, _ := c.call("PUT", selections+"/daily", definition, map[string]string{"If-Match": "0"}); status != 422 {
		t.Fatal("selection update accepted a create revision")
	}
	status, preview := c.call("POST", selections+"/daily/previews", map[string]any{}, nil)
	if status != 200 {
		t.Fatal(status, preview)
	}
	oldRelease := preview["releaseId"].(string)
	_, test := c.call("GET", base+"/tests/HEALTH-1", nil, nil)
	data := test["data"].(map[string]any)
	data["title"] = "New published title"
	if status, value := c.call("PUT", base+"/tests/HEALTH-1", data, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status, value)
	}
	publishSavedSelectionFixture(t, c, base)
	request := runs.Request{Suite: "health", Environment: "local", SelectionPreview: preview["id"].(string)}
	status, created := c.call("POST", "/api/v1/projects/example/runs", request, map[string]string{"Idempotency-Key": "reviewed-selection"})
	if status != 200 {
		t.Fatal(status, created)
	}
	stored := created["data"].(map[string]any)
	if stored["release"].(map[string]any)["id"] != oldRelease || stored["tests"].([]any)[0].(map[string]any)["title"] != "Availability" {
		t.Fatal("run did not freeze the reviewed published release", stored)
	}
	if !reflect.DeepEqual(stored["request"].(map[string]any)["testIds"], []any{"HEALTH-1"}) || stored["savedSelection"].(map[string]any)["revision"] != float64(1) {
		t.Fatal("missing frozen selection identity", stored)
	}
	definition.Name = "Changed daily health"
	if status, value := c.call("PUT", selections+"/daily", definition, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status, value)
	}
	if status, value := c.call("POST", "/api/v1/projects/example/runs", request, map[string]string{"Idempotency-Key": "reviewed-selection"}); status != 200 || value["id"] != created["id"] {
		t.Fatal("idempotent replay changed the queued snapshot", status, value)
	}
	if status, _ := c.call("POST", "/api/v1/projects/example/runs", request, map[string]string{"Idempotency-Key": "stale-selection"}); status != 409 {
		t.Fatal("a changed saved definition reused an old preview")
	}
	if status, current := c.call("GET", "/api/v1/projects/example/runs/"+created["id"].(string), nil, nil); status != 200 || current["data"].(map[string]any)["savedSelection"].(map[string]any)["definition"].(map[string]any)["name"] != "Daily health" {
		t.Fatal("saved selection edits rewrote an existing run", current)
	}
	status, fresh := c.call("POST", selections+"/daily/previews", map[string]any{}, nil)
	if status != 200 || fresh["releaseId"] == oldRelease {
		t.Fatal("fresh preview did not use the new active release", status, fresh)
	}
	request.SelectionPreview = fresh["id"].(string)
	if err := c.api.Catalog.ChangeState(t.Context(), "admin", "example", "health", "HEALTH-1", "disabled", 2); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("POST", "/api/v1/projects/example/runs", request, map[string]string{"Idempotency-Key": "disabled-selection"}); status != 422 {
		t.Fatal("saved preview bypassed current disabled state")
	}
	if err := c.api.Catalog.ChangeState(t.Context(), "admin", "example", "health", "HEALTH-1", "enabled", 3); err != nil {
		t.Fatal(err)
	}
	row, err := c.api.Store.Get(t.Context(), catalog.Bucket("selection-previews", "example", "health"), request.SelectionPreview)
	if err != nil {
		t.Fatal(err)
	}
	expired, _ := storage.Decode[catalog.SavedSelectionPreview](row)
	expired.Expires = time.Now().Add(-time.Second)
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, expired)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("POST", "/api/v1/projects/example/runs", request, map[string]string{"Idempotency-Key": "expired-selection"}); status != 422 {
		t.Fatal("expired preview dispatched a run")
	}
	if _, _, err := c.api.Catalog.SelectionFromPreview(t.Context(), "another-user", "example", "health", request.SelectionPreview); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("preview escaped its actor", err)
	}
}

func TestSavedSelectionFiltersResolveDependenciesAndRespectAuthority(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	_, original := c.call("GET", base+"/tests/HEALTH-1", nil, nil)
	encoded, _ := json.Marshal(original["data"])
	var child catalog.Test
	if err := json.Unmarshal(encoded, &child); err != nil {
		t.Fatal(err)
	}
	child.ID, child.Title, child.Category, child.Tags, child.Owner, child.Dependencies = "CHILD-1", "Child response", "Backend / Read", []string{"smoke", "local"}, "Example team", []string{"HEALTH-1"}
	if err := c.api.Catalog.SaveTest(t.Context(), "admin", "example", "health", child, 0); err != nil {
		t.Fatal(err)
	}
	publishSavedSelectionFixture(t, c, base)
	definition := catalog.SavedSelection{ID: "smoke", Name: "Backend smoke", Spec: catalog.SavedSelectionSpec{Version: 1, Kind: "filter", Filter: &catalog.SavedSelectionFilter{Category: "Backend", Descendants: true, Tags: []string{"smoke"}, Query: "child", Runner: "http", Owner: "Example team"}}}
	selections := base + "/saved-selections"
	if status, value := c.call("POST", selections, definition, nil); status != 200 {
		t.Fatal(status, value)
	}
	status, preview := c.call("POST", selections+"/smoke/previews", map[string]any{}, nil)
	if status != 200 || !reflect.DeepEqual(preview["requestedTestIds"], []any{"CHILD-1"}) || !reflect.DeepEqual(preview["resolvedTestIds"], []any{"CHILD-1", "HEALTH-1"}) {
		t.Fatal("filter omitted dependency closure", status, preview)
	}
	definition.Spec.Version = 2
	if status, _ := c.call("PUT", selections+"/smoke", definition, map[string]string{"If-Match": "1"}); status != 422 {
		t.Fatal("unsupported filter version accepted")
	}
	object := "example/test/health/HEALTH-1"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "example", ObjectScope: &object, PermissionName: "teststudio.catalog.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("POST", selections+"/smoke/previews", map[string]any{}, nil); status != 403 {
		t.Fatal("preview exposed a denied dependency")
	}
	if status, _ := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local", SelectionPreview: preview["id"].(string)}, map[string]string{"Idempotency-Key": "denied-dependency"}); status != 403 {
		t.Fatal("run ignored a revoked dependency read")
	}
	for index, action := range []string{"archive", "restore"} {
		if status, value := c.call("POST", selections+"/smoke/actions/"+action, nil, map[string]string{"If-Match": strconv.Itoa(index + 1)}); status != 200 {
			t.Fatal(action, status, value)
		}
		if action == "archive" {
			if status, _ := c.call("POST", selections+"/smoke/previews", map[string]any{}, nil); status != 422 {
				t.Fatal("archived selection resolved")
			}
			if status, page := c.call("GET", selections, nil, nil); status != 200 || len(page["items"].([]any)) != 0 {
				t.Fatal("archived selection remained in active choices", status, page)
			}
		}
	}
}

func publishSavedSelectionFixture(t *testing.T, c *client, base string) {
	t.Helper()
	status, draft := c.call("POST", base+"/catalog-drafts", catalog.Artifact{}, nil)
	if status != 200 {
		t.Fatal(status, draft)
	}
	if status, value := c.call("POST", base+"/catalog-drafts/"+draft["id"].(string)+"/publish", map[string]bool{"activate": true}, map[string]string{"If-Match": "1", "Idempotency-Key": "selection-release-" + draft["id"].(string)}); status != 200 {
		t.Fatal(status, value)
	}
}

func TestSavedSelectionFailureRerunSurvivesSelectionArchive(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	_, original := c.call("GET", base+"/tests/HEALTH-1", nil, nil)
	definition := original["data"].(map[string]any)
	definition["check"].(map[string]any)["statusCodes"] = []int{503}
	if status, value := c.call("PUT", base+"/tests/HEALTH-1", definition, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status, value)
	}
	publishSavedSelectionFixture(t, c, base)
	saved := catalog.SavedSelection{ID: "failed-check", Name: "Expected failure", Spec: catalog.SavedSelectionSpec{Version: 1, Kind: "explicit", IDs: []string{"HEALTH-1"}}}
	if status, value := c.call("POST", base+"/saved-selections", saved, nil); status != 200 {
		t.Fatal(status, value)
	}
	status, preview := c.call("POST", base+"/saved-selections/failed-check/previews", map[string]any{}, nil)
	if status != 200 {
		t.Fatal(status, preview)
	}
	status, parent := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local", SelectionPreview: preview["id"].(string)}, map[string]string{"Idempotency-Key": "selection-parent-failure"})
	if status != 200 {
		t.Fatal(status, parent)
	}
	var run runs.Run
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		var err error
		run, _, err = c.api.Runs.Get(t.Context(), "example", parent["id"].(string))
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(run.State) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Outcome != "failed" || run.SavedSelection == nil {
		t.Fatal("saved selection did not produce its expected failure", run)
	}
	if status, value := c.call("POST", base+"/saved-selections/failed-check/actions/archive", nil, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status, value)
	}
	status, child := c.call("POST", "/api/v1/projects/example/runs/"+run.ID+"/rerun-failures", nil, map[string]string{"Idempotency-Key": "selection-child-failure"})
	if status != 200 {
		t.Fatal("rerun depended on an archived selection preview", status, child)
	}
	request := child["data"].(map[string]any)["request"].(map[string]any)
	if request["savedSelectionPreviewId"] != nil || request["releaseId"] != run.Release.ID || !reflect.DeepEqual(request["testIds"], []any{"HEALTH-1"}) || request["parentRunId"] != run.ID {
		t.Fatal("rerun did not retain its parent release and failed identifiers", request)
	}
}
