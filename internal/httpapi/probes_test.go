package httpapi_test

import (
	"errors"
	"fmt"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/probes"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
	"strings"
	"testing"
	"time"
)

func probeFixture(t *testing.T) (*client, string, probes.Binding) {
	t.Helper()
	c, id := preparedRun(t)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, _, err := c.api.Runs.Get(t.Context(), "example", id)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(r.State) {
			break
		}
		time.Sleep(time.Millisecond * 5)
	}
	v := probes.Binding{Name: "Local homepage", Suite: "health", Test: "HEALTH-1", Environment: "local", Endpoint: "api", Location: "local"}
	status, row := c.call("POST", "/api/v1/projects/example/probes", v, nil)
	if status != 200 {
		t.Fatal(status, row)
	}
	v.ID = row["id"].(string)
	return c, id, v
}
func probeList(t *testing.T, c *client) map[string]any {
	t.Helper()
	status, page := c.call("GET", "/api/v1/projects/example/probes?hours=24", nil, nil)
	if status != 200 {
		t.Fatal(status, page)
	}
	return page
}
func TestProbeTargetsConflictsAndCurrentPermissions(t *testing.T) {
	c, id, v := probeFixture(t)
	base := "/api/v1/projects/example/probes"
	first := probeList(t, c)["items"].([]any)[0].(map[string]any)
	if first["state"] != "passed" || first["successRate"] != float64(100) {
		t.Fatal(first)
	}
	if status, row := c.call("POST", base, v, nil); status != 200 || row["id"] != v.ID {
		t.Fatal(status, row)
	}
	if status, row := c.call("PUT", base+"/"+v.ID, v, map[string]string{"If-Match": "99"}); status != 409 {
		t.Fatal(status, row)
	}
	row, err := c.api.Store.Get(t.Context(), projects.Bucket("environments", "example"), "local")
	if err != nil {
		t.Fatal(err)
	}
	env, _ := storage.Decode[projects.Environment](row)
	env.Endpoints["api"] = "https://changed.example.com"
	if err = c.api.Projects.SaveEnvironment(t.Context(), "admin", "example", env, row.Revision); err != nil {
		t.Fatal(err)
	}
	changed := probeList(t, c)["items"].([]any)[0].(map[string]any)
	if changed["state"] != "never-checked" || changed["latest"] != nil || changed["history"].([]any)[0].(map[string]any)["targetChanged"] != true {
		t.Fatal(changed)
	}
	grants := &access.Service{Store: c.api.Store, Auth: c.api.Auth}
	_, err = grants.CreateGrant(t.Context(), "admin", "example", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.runs.read", Effect: permissions.EffectDeny, Object: "run/" + id, Reason: "Probe history must respect exact run denial"})
	if err != nil {
		t.Fatal(err)
	}
	hidden := probeList(t, c)["items"].([]any)[0].(map[string]any)
	if len(hidden["history"].([]any)) != 0 {
		t.Fatal("denied run leaked", hidden)
	}
	_, err = grants.CreateGrant(t.Context(), "admin", "example", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: "test/health/HEALTH-1", Reason: "Probe binding must respect exact test denial"})
	if err != nil {
		t.Fatal(err)
	}
	if len(probeList(t, c)["items"].([]any)) != 0 {
		t.Fatal("denied test leaked")
	}
}
func TestProbeLatestUsesRunOrderAndSeparatesEnvironments(t *testing.T) {
	c, id, v := probeFixture(t)
	original, _, err := c.api.Runs.Get(t.Context(), "example", id)
	if err != nil {
		t.Fatal(err)
	}
	for _, copyID := range []string{"zz-003", "zz-002", "zz-004"} {
		copy := original
		copy.ID = copyID
		copy.Results = append([]execution.Result{}, original.Results...)
		if copyID == "zz-003" {
			copy.Results[0].State = "timed-out"
			copy.Results[0].HTTP = nil
			copy.Results[0].Error = "deadline exceeded"
		}
		if copyID == "zz-004" {
			copy.Request.Environment = "foreign"
		}
		copy.CreatedAt = time.Now()
		copy.FinishedAt = &copy.CreatedAt
		change, _ := storage.Put("runs", copyID, "example", 0, copy)
		if err := c.api.Store.Commit(t.Context(), change); err != nil {
			t.Fatal(err)
		}
	}
	value := probeList(t, c)["items"].([]any)[0].(map[string]any)
	if value["state"] != "timed-out" || value["latest"].(map[string]any)["runId"] != "zz-003" {
		t.Fatal(value)
	}
	row, _ := c.api.Store.Get(t.Context(), "runs", "zz-002")
	late, _ := storage.Decode[runs.Run](row)
	late.UpdatedAt = time.Now().Add(time.Hour)
	change, _ := storage.Put("runs", late.ID, "example", row.Revision, late)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	value = probeList(t, c)["items"].([]any)[0].(map[string]any)
	if value["latest"].(map[string]any)["runId"] != "zz-003" {
		t.Fatal("late completion reordered runs")
	}
	v.Archived = true
	if status, row := c.call("PUT", "/api/v1/projects/example/probes/"+v.ID, v, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status, row)
	}
	if len(probeList(t, c)["items"].([]any)) != 0 {
		t.Fatal("archived binding remained active")
	}
	if _, _, err := c.api.Runs.Get(t.Context(), "example", id); errors.Is(err, storage.ErrNotFound) {
		t.Fatal("archive removed run history")
	}
}

func TestProbeSummaryBoundsCoverageAndShowsMissedSchedule(t *testing.T) {
	c, _, _ := probeFixture(t)
	changes := []storage.Change{}
	for i := 0; i < 1001; i++ {
		run := runs.Run{ID: fmt.Sprintf("zz-%04d", i), Project: "example", CreatedAt: time.Now(), State: "completed", Request: runs.Request{Suite: "unrelated", Environment: "local", Location: "local"}}
		change, _ := storage.Put("runs", run.ID, "example", 0, run)
		changes = append(changes, change)
	}
	if err := c.api.Store.CommitAll(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	page := probeList(t, c)
	value := page["items"].([]any)[0].(map[string]any)
	if page["partial"] != true || value["successRate"] != nil || value["state"] != "not-in-window" {
		t.Fatal(page)
	}
	past := time.Now().Add(-time.Hour)
	schedule := schedules.Schedule{Definition: schedules.Definition{ID: "missed", Name: "Missed schedule", Enabled: true, Kind: "interval", IntervalSeconds: 300, Timezone: "UTC", Overlap: "skip", Request: runs.Request{Suite: "health", Environment: "local", Location: "local", IDs: []string{"HEALTH-1"}, TimeoutSeconds: 10}}, Project: "example", Owner: "admin", Generation: 1, NextAt: &past}
	change, _ := storage.Put(schedules.Bucket("example"), schedule.ID, "example", 0, schedule)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	value = probeList(t, c)["items"].([]any)[0].(map[string]any)
	if value["freshness"] != "stale" || value["scheduleState"] != "enabled" {
		t.Fatal(value)
	}
	future := time.Now().Add(time.Hour)
	schedule.ID = "z-future"
	schedule.Name = "Future firing"
	schedule.NextAt = &future
	change, _ = storage.Put(schedules.Bucket("example"), schedule.ID, "example", 0, schedule)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	value = probeList(t, c)["items"].([]any)[0].(map[string]any)
	if value["freshness"] != "stale" || value["scheduleState"] != "multiple" {
		t.Fatal("a future schedule hid the missed occurrence", value)
	}
}

func TestProbePagesRejectChangedScopeAndHistoryKeepsTarget(t *testing.T) {
	c, id, v := probeFixture(t)
	row, _ := c.api.Store.Get(t.Context(), projects.Bucket("environments", "example"), "local")
	env, _ := storage.Decode[projects.Environment](row)
	env.ID = "other"
	env.Name = "Other target"
	if err := c.api.Projects.SaveEnvironment(t.Context(), "admin", "example", env, 0); err != nil {
		t.Fatal(err)
	}
	v.ID = ""
	v.Name = "Other binding"
	v.Environment = "other"
	if status, value := c.call("POST", "/api/v1/projects/example/probes", v, nil); status != 200 {
		t.Fatal(status, value)
	}
	status, page := c.call("GET", "/api/v1/projects/example/probes?limit=1", nil, nil)
	if status != 200 || page["cursor"] == nil {
		t.Fatal(status, page)
	}
	cursor := page["cursor"].(string)
	if status, next := c.call("GET", "/api/v1/projects/example/probes?limit=1&cursor="+cursor, nil, nil); status != 200 || len(next["items"].([]any)) != 1 {
		t.Fatal(status, next)
	}
	if status, _ := c.call("GET", "/api/v1/projects/example/probes?limit=1&stage=production&cursor="+cursor, nil, nil); status != 422 {
		t.Fatal("cursor escaped filter scope", status)
	}
	for _, target := range []string{"local", "other"} {
		status, history := c.call("GET", "/api/v1/projects/example/suites/health/tests/HEALTH-1/run-history?environment="+target+"&location=local", nil, nil)
		if status != 200 {
			t.Fatal(status, history)
		}
		items := history["items"].([]any)
		if target == "local" && (len(items) != 1 || items[0].(map[string]any)["runId"] != id) || target == "other" && len(items) != 0 {
			t.Fatal(target, history)
		}
	}
}

func TestProbeDraftAndActiveDefinitionKeepObservationBoundaries(t *testing.T) {
	c, _, _ := probeFixture(t)
	row, err := c.api.Store.Get(t.Context(), catalog.Bucket("tests", "example", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	definition, _ := storage.Decode[catalog.Test](row)
	definition.Check.Path = "/new-health"
	if err := c.api.Catalog.SaveTest(t.Context(), "admin", "example", "health", definition, row.Revision); err != nil {
		t.Fatal(err)
	}
	value := probeList(t, c)["items"].([]any)[0].(map[string]any)
	if value["draftChanged"] != true || value["state"] != "passed" || !strings.HasSuffix(value["url"].(string), "/healthz") {
		t.Fatal("draft was displayed as the observed active target", value)
	}
	draft, err := c.api.Catalog.CreateDraft(t.Context(), "admin", "example", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.api.Catalog.Publish(t.Context(), "admin", "example", "health", draft.ID, "new-probe-definition", 1, true); err != nil {
		t.Fatal(err)
	}
	value = probeList(t, c)["items"].([]any)[0].(map[string]any)
	if value["state"] != "never-checked" || value["latest"] != nil || value["history"].([]any)[0].(map[string]any)["definitionChanged"] != true {
		t.Fatal("old assertion success became current after activation", value)
	}
}
