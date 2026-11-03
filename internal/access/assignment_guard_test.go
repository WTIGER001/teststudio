package access_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type assignmentGuardStore struct {
	*barrierStore
	transactor authorization.AssignmentTransactor
}

func (s *assignmentGuardStore) WithAssignmentTransaction(ctx context.Context, db *storage.Store, fn func(*storage.Store, permissions.PermissionStore) error) error {
	return s.transactor.WithAssignmentTransaction(ctx, db, func(bound *storage.Store, store permissions.PermissionStore) error {
		return fn(bound, &barrierStore{PermissionStore: store, arrivals: s.arrivals, ready: s.ready})
	})
}

func TestAccessAssignmentGuardSerializesRevocationWithAdministratorDisabling(t *testing.T) {
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
	arrivals, ready := &atomic.Int32{}, make(chan struct{})
	for _, service := range []*access.Service{first, second} {
		store := service.Auth.Engine.GetStore()
		guarded := &assignmentGuardStore{barrierStore: &barrierStore{PermissionStore: store, arrivals: arrivals, ready: ready}, transactor: store.(authorization.AssignmentTransactor)}
		engine, err := authorization.NewEngine(t.Context(), guarded, service.Auth.Directory)
		if err != nil {
			t.Fatal(err)
		}
		service.Auth = &authorization.Service{Engine: engine, Directory: service.Auth.Directory}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	results := make(chan error, 2)
	go func() {
		results <- first.RemoveAssignment(ctx, "admin", "", access.AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: backup.ID, RoleID: "builtin.teststudio.installation-admin", Reason: "Keep local administration"}, 1)
	}()
	go func() { results <- second.UserState(ctx, backup.ID, "admin", "Keep provider administration", 1, true) }()
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
		t.Fatalf("expected one administrator removal, got %d", wins)
	}
	root, err := first.Auth.Directory.User(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	backupAccess := first.Auth.Check(t.Context(), backup.ID, "", "", "system.manage")
	if root.Disabled && backupAccess != nil {
		t.Fatal("concurrent changes removed both login paths", backupAccess)
	}
	if !root.Disabled && !errors.Is(backupAccess, authorization.ErrForbidden) {
		t.Fatal("concurrent changes did not remove exactly one path", backupAccess)
	}
}
