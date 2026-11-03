package main

import (
	"errors"
	"testing"

	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestLocalRecoveryRestoresDisabledBuiltinAdministrationWithNewRevision(t *testing.T) {
	auth, dsn := bootstrapFixture(t)
	if err := bootstrapLocalAdministrator(t.Context(), auth, false); err != nil {
		t.Fatal(err)
	}
	id := "builtin.teststudio.installation-admin"
	if err := auth.WithPolicyTransaction(t.Context(), func(bound *authorization.Service) error {
		if err := bound.Engine.DisableBuiltInRole(t.Context(), id); err != nil {
			return err
		}
		change, err := storage.Put("access-builtin-roles", id, "", 0, map[string]any{"id": id, "disabled": true})
		if err != nil {
			return err
		}
		return bound.Directory.Store.Commit(t.Context(), change)
	}); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapLocalAdministrator(t.Context(), auth, false); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), "local-administrator", "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("ordinary startup enabled a disabled role", err)
	}
	db, err := storage.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	second, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Check(t.Context(), "local-administrator", "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if err := bootstrapLocalAdministrator(t.Context(), auth, true); err != nil {
		t.Fatal(err)
	}
	if err := second.Check(t.Context(), "local-administrator", "", "", "system.manage"); err != nil {
		t.Fatal("recovery did not restore independent checks", err)
	}
	s := &access.Service{Store: auth.Directory.Store, Auth: auth, LocalUser: "local-administrator"}
	row, err := s.ReadBuiltinRole(t.Context(), "local-administrator", id)
	if err != nil || row.Data.Disabled || row.Revision <= 4 {
		t.Fatal(row, err)
	}
	if _, err := s.UpdateBuiltinRole(t.Context(), "local-administrator", id, access.BuiltinRoleUpdate{Disabled: false, Reason: "Reject the previous state revision"}, 4); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("recovery reused revision", err)
	}
	rows, err := s.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		value, _ := storage.Decode[map[string]any](r)
		if value["action"] == "role.builtin.recover" {
			found = true
		}
	}
	if !found {
		t.Fatal("built-in recovery audit missing")
	}
}
