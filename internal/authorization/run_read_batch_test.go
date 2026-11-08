package authorization_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestRunReadBatchPreservesObjectDeniesScopeAndExpiry(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "batch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := &identity.Directory{Store: db}
	for _, id := range []string{"reader", "temporary", "app-reader"} {
		if err = dir.SaveUser(t.Context(), identity.User{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "reader"}, authorization.ReaderRole, map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err = auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "app-reader"}, authorization.ReaderRole, map[string]any{"team": "alpha", "application": "app"}); err != nil {
		t.Fatal(err)
	}
	object := "alpha/run/blocked"
	if err = auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "reader", TeamScope: "alpha", PermissionName: "teststudio.runs.read", Effect: permissions.EffectDeny, ObjectScope: &object}); err != nil {
		t.Fatal(err)
	}
	batch, err := auth.NewRunReadBatch(t.Context(), "reader")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"good", "another", "blocked", "final"} {
		err = batch.Check(t.Context(), "alpha", storage.Record{Bucket: "runs", ID: id, Project: "alpha", Data: json.RawMessage(`{}`)})
		if id == "blocked" && !errors.Is(err, authorization.ErrForbidden) || id != "blocked" && err != nil {
			t.Fatal(id, err)
		}
	}
	if err = batch.Check(t.Context(), "beta", storage.Record{Bucket: "runs", ID: "good", Project: "beta", Data: json.RawMessage(`{}`)}); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("project scope broadened", err)
	}
	appBatch, err := auth.NewRunReadBatch(t.Context(), "app-reader")
	if err != nil {
		t.Fatal(err)
	}
	for _, app := range []string{"app", "other", "app"} {
		data, _ := json.Marshal(map[string]any{"request": map[string]string{"applicationId": app}, "tests": []map[string]string{{"applicationId": app}}})
		err = appBatch.Check(t.Context(), "alpha", storage.Record{Bucket: "runs", ID: app, Project: "alpha", Data: data})
		if app == "app" && err != nil || app != "app" && !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("application scope changed", app, err)
		}
	}
	expiry := time.Now().Add(100 * time.Millisecond)
	if err = auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "temporary", TeamScope: "alpha", PermissionName: "teststudio.runs.read", Effect: permissions.EffectAllow, ExpiresAt: &expiry}); err != nil {
		t.Fatal(err)
	}
	temporary, err := auth.NewRunReadBatch(t.Context(), "temporary")
	if err != nil {
		t.Fatal(err)
	}
	row := storage.Record{Bucket: "runs", ID: "one", Project: "alpha", Data: json.RawMessage(`{}`)}
	if err = temporary.Check(t.Context(), "alpha", row); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(expiry.Add(5 * time.Millisecond)))
	if err = temporary.Check(t.Context(), "alpha", row); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("expired grant reused", err)
	}
	if err = dir.SaveUser(t.Context(), identity.User{ID: "reader", Name: "Reader", Disabled: true}, 1); err != nil {
		t.Fatal(err)
	}
	if err = batch.Current(t.Context()); err == nil {
		t.Fatal("directory change was not rejected")
	}
}
