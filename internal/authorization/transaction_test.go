package authorization_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func transactionFixture(t *testing.T) *authorization.Service {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "assignment.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "operator", Name: "Operator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	return auth
}

func TestAssignmentTransactionCommitsAuditAndExactBindingRevocation(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	principal := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	role := "builtin.teststudio.editor"
	version, err := auth.Directory.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, project := range []string{"alpha", "beta"} {
		err := auth.WithAssignmentTransaction(ctx, func(bound *authorization.Service) error {
			if err := bound.Engine.AssignRole(ctx, principal, role, map[string]any{"team": project}); err != nil {
				return err
			}
			if err := bound.Check(ctx, principal.ID, project, "", "catalog.update"); err != nil {
				return err
			}
			hits, err := bound.Engine.RoleAssignmentsForRoleID(ctx, role)
			if err != nil || len(hits) < 1 {
				t.Fatalf("role assignment reads miss pending writes: %+v %v", hits, err)
			}
			c, _ := storage.Put("audit", project, project, 0, "assignment added")
			return bound.Directory.Store.Commit(ctx, c)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	after, err := auth.Directory.Revision(ctx)
	if err != nil || after.Revision != version.Revision+2 {
		t.Fatal("directory revision not atomic", after, err)
	}
	err = auth.WithAssignmentTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.Engine.UnassignRole(ctx, principal, role, map[string]any{"team": "alpha"}); err != nil {
			return err
		}
		if err := bound.Check(ctx, principal.ID, "alpha", "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("pending removal not visible to evaluator", err)
		}
		if err := bound.Check(ctx, principal.ID, "beta", "", "catalog.update"); err != nil {
			t.Fatal("another assignment removed", err)
		}
		c, _ := storage.Put("audit", "removed", "alpha", 0, "exact assignment removed")
		return bound.Directory.Store.Commit(ctx, c)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(ctx, principal.ID, "alpha", "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal(err)
	}
	if err := auth.Check(ctx, principal.ID, "beta", "", "catalog.update"); err != nil {
		t.Fatal(err)
	}
	rows, err := auth.Directory.Store.All(ctx, "audit", "")
	if err != nil || len(rows) != 3 {
		t.Fatal("audit did not commit with assignments", rows, err)
	}
}

func TestAssignmentTransactionRollsBackPermissionsAuditAndDirectoryClock(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	principal := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	role := "builtin.teststudio.editor"
	if err := auth.Engine.AssignRole(ctx, principal, role, map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	version, err := auth.Directory.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("last administrator safeguard rejects change")
	err = auth.WithAssignmentTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.Engine.UnassignRole(ctx, principal, role, map[string]any{"team": "alpha"}); err != nil {
			return err
		}
		if err := bound.Engine.AssignRole(ctx, principal, role, map[string]any{"team": "beta"}); err != nil {
			return err
		}
		c, _ := storage.Put("audit", "rollback", "alpha", 0, "must not survive")
		if err := bound.Directory.Store.Commit(ctx, c); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err := auth.Check(ctx, principal.ID, "alpha", "", "catalog.update"); err != nil {
		t.Fatal("revocation leaked", err)
	}
	if err := auth.Check(ctx, principal.ID, "beta", "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("assignment leaked", err)
	}
	if _, err := auth.Directory.Store.Get(ctx, "audit", "rollback"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("audit leaked", err)
	}
	after, err := auth.Directory.Revision(ctx)
	if err != nil || after.Revision != version.Revision {
		t.Fatal("directory clock leaked", after, err)
	}
}

func TestAssignmentTransactionRejectsUnversionedStoreWrappers(t *testing.T) {
	auth := transactionFixture(t)
	engine, err := authorization.NewEngine(t.Context(), struct{ permissions.PermissionStore }{auth.Engine.GetStore()}, auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := &authorization.Service{Engine: engine, Directory: auth.Directory}
	if err := wrapped.WithAssignmentTransaction(t.Context(), func(*authorization.Service) error { t.Fatal("unsafe callback executed"); return nil }); err == nil {
		t.Fatal("transaction silently fell back to non-atomic writes")
	}
}

func TestAssignmentTransactionChecksExplicitDeniesFromPersistentStore(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	p := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	if err := auth.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: p.Kind, OwnerID: p.ID, TeamScope: "alpha", PermissionName: "teststudio.catalog.update", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if err := auth.WithAssignmentTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.Engine.AssignRole(ctx, p, "builtin.teststudio.editor", map[string]any{"team": "alpha"}); err != nil {
			return err
		}
		return bound.Check(context.Background(), p.ID, "alpha", "", "catalog.update")
	}); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("transaction bypassed persistent deny", err)
	}
	if rows, err := auth.Engine.RoleAssignmentsForPrincipal(ctx, p); err != nil || len(rows) != 0 {
		t.Fatal("denied change persisted", rows, err)
	}
}

type transactionBarrier struct {
	permissions.PermissionStore
	authorization.AssignmentTransactor
	arrivals *atomic.Int32
	ready    chan struct{}
}

func (s *transactionBarrier) WithAssignmentTransaction(ctx context.Context, metadata *storage.Store, fn func(*storage.Store, permissions.PermissionStore) error) error {
	if s.arrivals.Add(1) == 2 {
		close(s.ready)
	}
	select {
	case <-s.ready:
	case <-ctx.Done():
		return ctx.Err()
	}
	return s.AssignmentTransactor.WithAssignmentTransaction(ctx, metadata, fn)
}

func TestAssignmentTransactionSerializesConcurrentDirectoryChanges(t *testing.T) {
	auth := transactionFixture(t)
	var path string
	if auth.Directory.Store.Dialect == "sqlite" {
		if err := auth.Directory.Store.DB.QueryRowContext(t.Context(), "SELECT file FROM pragma_database_list WHERE name='main'").Scan(&path); err != nil {
			t.Fatal(err)
		}
	}
	db, err := storage.Open(t.Context(), testkit.ConnectionString(t, auth.Directory.Store, path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	second, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	ready, arrivals := make(chan struct{}), &atomic.Int32{}
	services := []*authorization.Service{}
	for _, instance := range []*authorization.Service{auth, second} {
		barrier := &transactionBarrier{PermissionStore: instance.Engine.GetStore(), AssignmentTransactor: instance.Engine.GetStore().(authorization.AssignmentTransactor), ready: ready, arrivals: arrivals}
		engine, err := authorization.NewEngine(t.Context(), barrier, instance.Directory)
		if err != nil {
			t.Fatal(err)
		}
		services = append(services, &authorization.Service{Engine: engine, Directory: instance.Directory})
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	results := make(chan error, 2)
	for i, project := range []string{"alpha", "beta"} {
		guarded := services[i]
		go func() {
			results <- guarded.WithAssignmentTransaction(ctx, func(bound *authorization.Service) error {
				c, _ := storage.Put("audit", project, project, 0, "competing change")
				if err := bound.Directory.Store.Commit(ctx, c); err != nil {
					return err
				}
				return bound.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "builtin.teststudio.editor", map[string]any{"team": project})
			})
		}()
	}
	wins := 0
	for range 2 {
		err := <-results
		if err == nil {
			wins++
		} else if !errors.Is(err, storage.ErrConflict) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatalf("expected one commit, got %d", wins)
	}
	assignments, err := auth.Engine.RoleAssignmentsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"})
	if err != nil || len(assignments) != 1 {
		t.Fatal("competing permission change leaked", assignments, err)
	}
	audit, err := auth.Directory.Store.All(ctx, "audit", "")
	if err != nil || len(audit) != 1 {
		t.Fatal("competing audit leaked", audit, err)
	}
}

func TestAssignmentTransactionKeepsIdempotentAssignmentAndNilRevocationExact(t *testing.T) {
	auth := transactionFixture(t)
	p := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	role := "builtin.teststudio.editor"
	for range 2 {
		if err := auth.WithAssignmentTransaction(t.Context(), func(bound *authorization.Service) error {
			return bound.Engine.AssignRole(t.Context(), p, role, map[string]any{"team": "alpha"})
		}); err != nil {
			t.Fatal("idempotent assignment failed", err)
		}
	}
	if err := auth.WithAssignmentTransaction(t.Context(), func(bound *authorization.Service) error {
		return bound.Engine.UnassignRole(t.Context(), p, role, nil)
	}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), p.ID, "alpha", "", "catalog.update"); err != nil {
		t.Fatal("nil revocation removed a bound assignment", err)
	}
}

func TestAssignmentTransactionRejectsUnrelatedPostgresPolicyWrites(t *testing.T) {
	auth := transactionFixture(t)
	if auth.Directory.Store.Dialect != "postgres" {
		t.Skip("native PostgreSQL assignment adapter")
	}
	failure := errors.New("reject workflow")
	err := auth.WithAssignmentTransaction(t.Context(), func(bound *authorization.Service) error {
		store := bound.Engine.GetStore()
		for _, mutate := range []func() error{
			func() error {
				return store.CreateRole(t.Context(), permissions.Role{ID: "must-not-exist", Name: "Rejected"})
			},
			func() error {
				return store.UpdateRole(t.Context(), permissions.Role{ID: "must-not-exist", Name: "Rejected"})
			},
			func() error { return store.DeleteRole(t.Context(), "must-not-exist") },
			func() error { return store.AddRoleInheritance(t.Context(), "one", "two") },
			func() error {
				return store.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "operator", PermissionName: "teststudio.system.manage", TeamScope: "*", Effect: permissions.EffectAllow})
			},
			func() error { return store.DeleteGrantsForOwner(t.Context(), permissions.PrincipalUser, "operator") },
			func() error { return store.DisableBuiltInRole(t.Context(), "builtin.teststudio.editor") },
			func() error { return store.EnableBuiltInRole(t.Context(), "builtin.teststudio.editor") },
		} {
			if err := mutate(); err == nil {
				t.Fatal("unrelated policy write escaped the assignment transaction")
			}
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := auth.Engine.RoleDefinition(t.Context(), "must-not-exist"); err == nil {
		t.Fatal("role leaked outside transaction")
	}
	if err := auth.Check(t.Context(), "operator", "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("grant leaked outside transaction", err)
	}
	disabled, err := auth.Engine.GetStore().DisabledBuiltInRoles(t.Context())
	if err != nil || len(disabled) != 0 {
		t.Fatal("role override leaked", disabled, err)
	}
}
