package httpapi_test

import (
	"context"
	"errors"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/storage"
)

type interactiveDriver struct {
	recordingDriver
	inputs int
}

func (d *interactiveDriver) View(context.Context, string) (recording.View, error) {
	return recording.View{Image: "private-frame", Width: 1280, Height: 720, Mode: "recording", Steps: []recording.Step{}}, nil
}

func (d *interactiveDriver) Interact(ctx context.Context, id string, action recording.Action) (recording.View, error) {
	d.inputs++
	return d.View(ctx, id)
}

func TestEmbeddedRecordingRequiresOwnerCurrentAuthorityActiveSuiteAndCSRF(t *testing.T) {
	driver := &interactiveDriver{recordingDriver: recordingDriver{snapshots: map[string]recording.Snapshot{}}}
	c := fixture(t, func(api *httpapi.Server) {
		api.Recordings = &recording.Service{Store: api.Store, Auth: api.Auth, Driver: driver}
	})
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", map[string]string{"id": "recording", "name": "Recording"}, nil); status != 200 {
		t.Fatal(data)
	}
	suite := "/api/v1/projects/recording/suites/browser"
	if status, data := c.call("PUT", suite, map[string]any{"id": "browser", "name": "Browser", "applications": []string{}}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	base := suite + "/browser-recordings"
	input := recording.Input{ID: "embedded", TestID: "NEWS-001", Title: "News stories appear", URL: "https://example.test"}
	if status, data := c.call("POST", base, input, nil); status != 200 || data["data"].(map[string]any)["mode"] != "embedded" {
		t.Fatal(data)
	}
	if status, data := c.call("POST", base, input, nil); status != 200 || driver.starts != 1 {
		t.Fatal("retry opened a second browser", data)
	}
	if status, data := c.call("POST", base+"/embedded/view", nil, nil); status != 200 || data["image"] != "private-frame" {
		t.Fatal(data)
	}
	action := recording.Action{Sequence: 1, Action: "click", X: 20, Y: 20}
	if status, _ := c.call("POST", base+"/embedded/input", action, map[string]string{"X-CSRF-Token": ""}); status != 403 {
		t.Fatal("live input accepted missing CSRF")
	}
	if status, _ := c.call("POST", base+"/embedded/view", nil, map[string]string{"X-CSRF-Token": ""}); status != 403 {
		t.Fatal("private live view accepted missing CSRF")
	}
	if status, data := c.call("POST", base+"/embedded/input", action, nil); status != 200 || driver.inputs != 1 {
		t.Fatal(data)
	}
	for _, invalid := range []recording.Action{{Sequence: 1, Action: "evaluate", Text: "arbitrary code"}, {Sequence: 1, Action: "click", X: 1280}, {Sequence: 1, Action: "key", Key: "Enter|Tab"}, {Action: "click"}} {
		if status, _ := c.call("POST", base+"/embedded/input", invalid, nil); status != 422 {
			t.Fatal("invalid browser action was accepted")
		}
	}
	if driver.inputs != 1 {
		t.Fatal("invalid request reached browser")
	}
	if err := c.api.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "other", Name: "Other publisher"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "other"}, "builtin.teststudio.publisher", map[string]any{"team": "recording"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.api.Recordings.Interactive(t.Context(), "other", "recording", "browser", "embedded", nil); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("another publisher accessed the private browser", err)
	}
	if _, err := c.api.Recordings.Interactive(t.Context(), "admin", "recording", "different", "embedded", nil); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("another suite accessed the private browser", err)
	}
	if _, err := c.api.Projects.ChangeSuiteState(t.Context(), "admin", "recording", "browser", "archive", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.api.Recordings.Interactive(t.Context(), "admin", "recording", "browser", "embedded", &action); err == nil {
		t.Fatal("archived suite still accepted browser actions")
	}
	if _, err := c.api.Projects.ChangeSuiteState(t.Context(), "admin", "recording", "browser", "restore", 2); err != nil {
		t.Fatal(err)
	}
	row, err := c.api.Store.Get(t.Context(), recording.Bucket(), "embedded")
	if err != nil {
		t.Fatal(err)
	}
	value, _ := storage.Decode[recording.Session](row)
	value.DeadlineAt = time.Now().Add(-time.Minute)
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if _, err := c.api.Recordings.Interactive(t.Context(), "admin", "recording", "browser", "embedded", &action); err == nil {
		t.Fatal("expired browser still accepted input")
	}
	value.DeadlineAt = time.Now().Add(time.Minute)
	change, _ = storage.Put(row.Bucket, row.ID, row.Project, row.Revision+1, value)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	object := "recording/suite/browser"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "recording", ObjectScope: &object, PermissionName: "teststudio.sources.manage", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}

	if _, err := c.api.Recordings.Interactive(t.Context(), "admin", "recording", "browser", "embedded", &action); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("revoked source authority still accepted browser actions", err)
	}
}
