package httpapi_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/storage"
)

type recordingDriver struct {
	mu        sync.Mutex
	snapshots map[string]recording.Snapshot
	starts    int
	stops     int
}

func (d *recordingDriver) Start(_ context.Context, id, address string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, snapshot := range d.snapshots {
		if snapshot.State == "recording" {
			return recording.ErrBusy
		}
	}
	d.starts++
	d.snapshots[id] = recording.Snapshot{State: "recording", Code: fmt.Sprintf("import { test, expect } from '@playwright/test';\n\ntest('test', async ({ page }) => {\n  await page.goto(%q);\n  await page.getByRole('button', { name: 'Continue' }).click();\n});\n", address)}
	return nil
}
func (d *recordingDriver) Read(_ context.Context, id string) (recording.Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.snapshots[id], nil
}
func (d *recordingDriver) Stop(_ context.Context, id string) (recording.Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stops++
	value := d.snapshots[id]
	value.State = "ready"
	d.snapshots[id] = value
	return value, nil
}

func TestRecordingAPIKeepsOwnerPrivateCodeAndCreatesReviewedSourceWithoutRuns(t *testing.T) {
	driver := &recordingDriver{snapshots: map[string]recording.Snapshot{}}
	c := fixture(t, func(api *httpapi.Server) {
		api.Recordings = &recording.Service{Store: api.Store, Auth: api.Auth, Driver: driver}
	})
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", map[string]string{"id": "recording", "name": "Recording"}, nil); status != 200 {
		t.Fatal(data)
	}
	suite := "/api/v1/projects/recording/suites/browser"
	if status, data := c.call("PUT", suite, map[string]any{"id": "browser", "name": "Browser", "applications": []string{}, "requiresTarget": true}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	base := suite + "/browser-recordings"
	input := recording.Input{ID: "session", TestID: "REC-001", Title: "Record a continuation", URL: "http://127.0.0.1:8070"}
	if status, _ := c.call("POST", base, input, map[string]string{"X-CSRF-Token": ""}); status != 403 {
		t.Fatal("recording accepted missing CSRF")
	}
	if status, data := c.call("POST", base, input, nil); status != 200 {
		t.Fatal(data)
	}
	if status, data := c.call("POST", base, input, nil); status != 200 || driver.starts != 1 {
		t.Fatal("recording retry spawned another browser", data)
	}
	changed := input
	changed.URL = "https://another.example.test"
	if status, _ := c.call("POST", base, changed, nil); status != 409 {
		t.Fatal("recording identifier accepted changed parameters")
	}
	if status, data := c.call("GET", base+"/session", nil, nil); status != 200 || data["data"].(map[string]any)["code"] == "" {
		t.Fatal("generated code not retained", data)
	}
	if err := c.api.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "recording-viewer", Name: "Viewer"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "recording-viewer"}, "builtin.teststudio.viewer", map[string]any{"team": "recording"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.api.Recordings.Read(t.Context(), "recording-viewer", "recording", "browser", "session"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("another actor read private recording", err)
	}
	list, err := c.api.Recordings.List(t.Context(), "recording-viewer", "recording", "browser", "", 30)
	if err != nil || len(list.Items) != 0 {
		t.Fatal("another actor's history exposed source", err)
	}
	response, err := http.Get(c.origin + base + "/session")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 401 {
		t.Fatal("anonymous recording read accepted")
	}
	if status, _ := c.call("GET", base+"/session/source", nil, nil); status != 422 {
		t.Fatal("unfinished recording became a draft")
	}
	if status, data := c.call("POST", base+"/session/actions/stop", nil, nil); status != 200 || data["data"].(map[string]any)["state"] != "ready" {
		t.Fatal(data)
	}
	if status, _ := c.call("POST", base+"/session/actions/stop", nil, nil); status != 200 || driver.stops != 1 {
		t.Fatal("stop retry changed recording")
	}
	status, draft := c.call("GET", base+"/session/source", nil, nil)
	if status != 200 || draft["targetUrl"] != input.URL {
		t.Fatal("recording target was lost", draft)
	}
	files := draft["files"].([]any)
	if files[0].(map[string]any)["path"] != "checks/recording.spec.ts" {
		t.Fatal("recording file absent")
	}
	id := draft["id"].(string)
	if status, data := c.call("PUT", suite+"/playwright-drafts/"+id, draft, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	if status, _ := c.call("POST", suite+"/playwright-drafts/"+id+"/prepare", nil, map[string]string{"If-Match": "1", "Idempotency-Key": "guarded"}); status != 422 {
		t.Fatal("recorded website used isolated execution profile")
	}
	if status, data := c.call("GET", "/api/v1/projects/recording/runs", nil, nil); status != 200 || len(data["items"].([]any)) != 0 {
		t.Fatal("recording created a run", data)
	}
	row, err := c.api.Store.Get(t.Context(), recording.Bucket(), "session")
	if err != nil {
		t.Fatal(err)
	}
	value, _ := storage.Decode[recording.Session](row)
	if value.State != "ready" || value.Code == "" {
		t.Fatal("finished source not durable")
	}
}
