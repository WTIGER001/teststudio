package access_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestBuiltinRoleGuardSerializesGlobalDisablingWithAlternateAccountRemoval(t *testing.T) {
	first := fixture(t)
	backup := providerUser(t, first, "backup")
	policy := access.RolePolicy{ID: "alternate-administration", Name: "Alternate administration", Rules: []access.RoleRule{}, Inherits: []string{}}
	for _, name := range authorization.PermissionNames() {
		policy.Rules = append(policy.Rules, access.RoleRule{Permission: name, Effect: permissions.EffectAllow})
	}
	if _, err := first.SaveRolePolicy(t.Context(), "admin", access.RolePolicyMutation{RolePolicy: policy, Reason: "Retain independent recovery access"}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := first.CreateAssignment(t.Context(), "admin", "", access.AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: backup.ID, RoleID: policy.ID, Reason: "Retain provider administration"}); err != nil {
		t.Fatal(err)
	}
	var local string
	if first.Store.Dialect == "sqlite" {
		if err := first.Store.DB.QueryRowContext(t.Context(), "SELECT file FROM pragma_database_list WHERE name='main'").Scan(&local); err != nil {
			t.Fatal(err)
		}
	}
	db, err := storage.Open(t.Context(), testkit.ConnectionString(t, first.Store, local))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	second := &access.Service{Store: db, Auth: auth, LocalUser: first.LocalUser}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	results, ready := make(chan error, 2), make(chan struct{})
	go func() {
		<-ready
		_, err := first.UpdateBuiltinRole(ctx, "admin", "builtin.teststudio.installation-admin", access.BuiltinRoleUpdate{Disabled: true, Reason: "Keep alternate custom administration"}, 1)
		results <- err
	}()
	go func() {
		<-ready
		results <- second.UserState(ctx, backup.ID, backup.ID, "Keep predefined administration", 1, true)
	}()
	close(ready)
	wins := 0
	for range 2 {
		err := <-results
		if err == nil {
			wins++
		} else if !errors.Is(err, storage.ErrConflict) && !strings.Contains(err.Error(), "last usable") {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatalf("expected one committed removal path, got %d", wins)
	}
	rootAllowed := first.Auth.Check(t.Context(), "admin", "", "", "system.manage") == nil
	backupAllowed := second.Auth.Check(t.Context(), backup.ID, "", "", "system.manage") == nil
	if rootAllowed == backupAllowed {
		t.Fatal("expected one usable administrator after competing changes")
	}
}
