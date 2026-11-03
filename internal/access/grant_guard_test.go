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

func TestDirectGrantGuardSerializesCompetingAdministratorDenies(t *testing.T) {
	first := fixture(t)
	backup := providerUser(t, first, "backup")
	if err := first.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: backup.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
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
	results := make(chan error, 2)
	ready := make(chan struct{})
	for i, user := range []string{"admin", backup.ID} {
		service := []*access.Service{first, second}[i]
		go func() {
			<-ready
			_, err := service.CreateGrant(ctx, user, "", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: user, Permission: "teststudio.system.manage", Effect: permissions.EffectDeny, Reason: "Keep the other administrator"})
			results <- err
		}()
	}
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
		t.Fatalf("expected one committed deny, got %d", wins)
	}
	rootAllowed := first.Auth.Check(t.Context(), "admin", "", "", "system.manage") == nil
	backupAllowed := first.Auth.Check(t.Context(), backup.ID, "", "", "system.manage") == nil
	if rootAllowed == backupAllowed {
		t.Fatal("expected one administrator to retain access")
	}
	rows, err := first.Store.All(t.Context(), "access-grants/", "")
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
}
