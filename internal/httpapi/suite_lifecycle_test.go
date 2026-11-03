package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
)

func TestSuiteLifecyclePreservesPublishedRunsEvidenceAndSchedulePins(t *testing.T) {
	c, runID := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	runPath := "/api/v1/projects/example/runs/" + runID
	var frozen map[string]any
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, response := c.call("GET", runPath, nil, nil)
		frozen = response["data"].(map[string]any)
		if frozen["state"] == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if frozen["state"] != "completed" || frozen["outcome"] != "passed" {
		t.Fatal("fixture run did not finish", frozen)
	}
	release := frozen["release"].(map[string]any)["id"].(string)
	pin := schedules.Definition{ID: "pinned", Name: "Pinned history", Kind: "interval", IntervalSeconds: 60, Request: runs.Request{Suite: "health", Environment: "local", Release: release}}
	if status, value := c.call("POST", "/api/v1/projects/example/schedules", pin, nil); status != 200 {
		t.Fatal(status, value)
	}
	before := readSuiteLifecycle(t, c, base)
	if len(before.Schedules) != 1 || before.Schedules[0].Release != release || !before.CanManage {
		t.Fatal("missing schedule or authority", before)
	}
	input := projects.SuiteSettings{Name: "Renamed availability", Description: "Retained local history", Applications: []string{"app"}, RequiresTarget: true}
	if status, value := c.call("PATCH", base+"/settings", input, nil); status != 422 {
		t.Fatal("settings accepted a missing revision", status, value)
	}
	header := func(revision int64) map[string]string {
		return map[string]string{"If-Match": strconv.FormatInt(revision, 10)}
	}
	if status, value := c.call("PATCH", base+"/settings", input, header(before.Revision)); status != 200 {
		t.Fatal("settings update", status, value)
	}
	after := readSuiteLifecycle(t, c, base)
	if after.Suite.Name != input.Name || after.Suite.ActiveRelease != release || after.Revision != before.Revision+1 || !reflect.DeepEqual(after.Suite.Capture, before.Suite.Capture) || !reflect.DeepEqual(after.Suite.Timing, before.Suite.Timing) {
		t.Fatal("settings changed execution configuration", before, after)
	}
	if status, _ := c.call("PATCH", base+"/settings", input, header(before.Revision)); status != 409 {
		t.Fatal("stale suite settings did not conflict")
	}
	invalid := input
	invalid.Applications = []string{"app", "app"}
	if status, _ := c.call("PATCH", base+"/settings", invalid, header(after.Revision)); status != 422 {
		t.Fatal("duplicate applications accepted")
	}
	if status, value := c.call("POST", base+"/actions/archive", nil, header(after.Revision)); status != 200 {
		t.Fatal("archive", status, value)
	}
	archived := readSuiteLifecycle(t, c, base)
	if !archived.Suite.Archived || archived.Suite.ActiveRelease != release || !reflect.DeepEqual(archived.Schedules, before.Schedules) {
		t.Fatal("archive changed release or schedule pins", archived)
	}
	if status, _ := c.call("PATCH", base+"/settings", input, header(archived.Revision)); status != 422 {
		t.Fatal("archived suite settings changed")
	}
	if status, _ := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local"}, map[string]string{"Idempotency-Key": "archived-dispatch"}); status != 422 {
		t.Fatal("archived suite dispatched a new run")
	}
	if status, _ := c.call("GET", base+"/release-registry", nil, nil); status != 200 {
		t.Fatal("archived suite lost its releases")
	}
	if status, history := c.call("GET", runPath, nil, nil); status != 200 || !reflect.DeepEqual(frozen, history["data"]) {
		t.Fatal("archive changed historical run", status, history)
	}
	status, artifacts := c.call("GET", runPath+"/artifacts", nil, nil)
	if status != 200 || len(artifacts["items"].([]any)) == 0 {
		t.Fatal("archive removed evidence", status, artifacts)
	}
	id := artifacts["items"].([]any)[0].(map[string]any)["id"].(string)
	status, access := c.call("POST", runPath+"/artifacts/"+id+"/access", map[string]string{"action": "download"}, nil)
	if status != 200 {
		t.Fatal(status, access)
	}
	response, err := http.Get(access["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || len(content) == 0 {
		t.Fatal("archived evidence is unavailable", response.StatusCode, err)
	}
	if status, value := c.call("POST", base+"/actions/restore", nil, header(archived.Revision)); status != 200 {
		t.Fatal("restore", status, value)
	}
	restored := readSuiteLifecycle(t, c, base)
	if restored.Suite.Archived || restored.Suite.ID != before.Suite.ID || restored.Suite.ActiveRelease != release || !reflect.DeepEqual(restored.Schedules, before.Schedules) {
		t.Fatal("restore changed retained identity", restored)
	}
	if status, value := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local"}, map[string]string{"Idempotency-Key": "restored-dispatch"}); status != 200 {
		t.Fatal("restored suite could not dispatch", status, value)
	}
}

func TestSuiteLifecycleHonorsExactReadManageAndScheduleDenies(t *testing.T) {
	for _, denied := range []string{"read", "manage"} {
		t.Run(denied, func(t *testing.T) {
			c, _ := preparedRun(t)
			base := "/api/v1/projects/example/suites/health"
			initial := readSuiteLifecycle(t, c, base)
			pin := schedules.Definition{ID: "hidden", Name: "Private schedule", Kind: "interval", IntervalSeconds: 60, Request: runs.Request{Suite: "health", Environment: "local"}}
			if status, value := c.call("POST", "/api/v1/projects/example/schedules", pin, nil); status != 200 {
				t.Fatal(status, value)
			}
			object := "example/schedule/hidden"
			if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.schedules.read", TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
				t.Fatal(err)
			}
			if state := readSuiteLifecycle(t, c, base); len(state.Schedules) != 0 {
				t.Fatal("suite view exposed a denied schedule", state)
			}
			object = "example/suite/health"
			if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.suites." + denied, TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
				t.Fatal(err)
			}
			if denied == "manage" {
				if state := readSuiteLifecycle(t, c, base); state.CanManage {
					t.Fatal("exact management deny left controls enabled")
				}
			} else {
				if status, _ := c.call("GET", base+"/lifecycle", nil, nil); status != 403 {
					t.Fatal("exact read deny exposed lifecycle")
				}
				if status, listing := c.call("GET", "/api/v1/projects/example/suites", nil, nil); status != 200 || len(listing["items"].([]any)) != 0 {
					t.Fatal("suite listing exposed denied metadata", status, listing)
				}
			}
			header := map[string]string{"If-Match": strconv.FormatInt(initial.Revision, 10)}
			if status, _ := c.call("PUT", base, initial.Suite, header); status != 403 {
				t.Fatal("legacy suite save bypassed exact authority")
			}
			if status, _ := c.call("PATCH", base+"/settings", projects.SuiteSettings{Name: "Denied change"}, header); status != 403 {
				t.Fatal("suite settings bypassed current authority")
			}
			if status, _ := c.call("POST", base+"/actions/archive", nil, header); status != 403 {
				t.Fatal("suite archive bypassed current authority")
			}
			stored, err := c.api.Store.Get(t.Context(), "suites/example", "health")
			if err != nil || stored.Revision != initial.Revision {
				t.Fatal("rejected changes mutated the suite", stored, err)
			}
		})
	}
}

func readSuiteLifecycle(t *testing.T, c *client, base string) projects.SuiteLifecycle {
	t.Helper()
	status, value := c.call("GET", base+"/lifecycle", nil, nil)
	if status != 200 {
		t.Fatal(status, value)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var state projects.SuiteLifecycle
	if err := json.Unmarshal(encoded, &state); err != nil {
		t.Fatal(err)
	}
	return state
}
