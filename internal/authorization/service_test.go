package authorization_test

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestPermissionChangesReachIndependentServiceInstances(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "permissions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "operator", Name: "Operator"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := dir.SaveGroup(t.Context(), identity.Group{ID: "editors", Name: "Editors", Project: "alpha"}, 0); err != nil {
		t.Fatal(err)
	}
	a, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	group := permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: "editors"}
	if err := a.Engine.AssignRole(t.Context(), group, "builtin.teststudio.editor", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	check := func(project, object, name string, allowed bool) {
		t.Helper()
		err := b.Check(t.Context(), "operator", project, object, name)
		if allowed && err != nil {
			t.Fatalf("expected access to %s/%s: %v", project, name, err)
		}
		if !allowed && !errors.Is(err, authorization.ErrForbidden) {
			t.Fatalf("expected denial for %s/%s: %v", project, name, err)
		}
	}
	waitForRole := func(allowed bool) {
		t.Helper()
		deadline := time.Now().Add(5100 * time.Millisecond)
		for {
			err := b.Check(t.Context(), "operator", "alpha", "", "catalog.update")
			if (allowed && err == nil) || (!allowed && errors.Is(err, authorization.ErrForbidden)) {
				return
			}
			if err != nil && !errors.Is(err, authorization.ErrForbidden) {
				t.Fatal(err)
			}
			if time.Now().After(deadline) {
				t.Fatal("built-in role override exceeded its five-second cache bound")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	check("alpha", "", "catalog.update", false)
	if err := dir.AddMember(t.Context(), identity.Membership{UserID: "operator", GroupID: "editors"}); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.update", true)
	check("beta", "", "catalog.update", false)
	if err := dir.RemoveMember(t.Context(), "operator", "editors", "local"); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.update", false)
	if err := dir.AddMember(t.Context(), identity.Membership{UserID: "operator", GroupID: "editors"}); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Minute)
	grant := permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "operator", Effect: permissions.EffectDeny, TeamScope: "alpha", PermissionName: "teststudio.catalog.update", ExpiresAt: &expired}
	if err := a.Engine.CreateGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.update", true)
	grant.ExpiresAt = nil
	if err := a.Engine.CreateGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.update", false)
	if err := a.Engine.DeleteGrantsForOwner(t.Context(), permissions.PrincipalUser, "operator"); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.update", true)
	if err := a.Engine.DisableBuiltInRole(t.Context(), "builtin.teststudio.editor"); err != nil {
		t.Fatal(err)
	}
	if err := a.Check(t.Context(), "operator", "alpha", "", "catalog.update"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatalf("local role cache did not invalidate: %v", err)
	}
	waitForRole(false)
	if err := a.Engine.EnableBuiltInRole(t.Context(), "builtin.teststudio.editor"); err != nil {
		t.Fatal(err)
	}
	if err := a.Check(t.Context(), "operator", "alpha", "", "catalog.update"); err != nil {
		t.Fatalf("local role enable did not invalidate: %v", err)
	}
	waitForRole(true)
	if err := a.Engine.UnassignRole(t.Context(), group, "builtin.teststudio.editor", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.update", false)
	for _, id := range []string{"project-reviewer", "project-reader"} {
		if err := a.Engine.CreateRole(t.Context(), permissions.Role{ID: id, Name: id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Engine.AddRoleInheritance(t.Context(), "project-reviewer", "project-reader"); err != nil {
		t.Fatal(err)
	}
	if err := a.Engine.AddRoleInheritance(t.Context(), "project-reader", "project-reviewer"); err == nil {
		t.Fatal("accepted an inheritance cycle")
	}
	if err := a.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: "project-reader", TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow}); err != nil {
		t.Fatal(err)
	}
	if err := a.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "project-reviewer", nil); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.read", true)
	check("beta", "", "catalog.read", false)
	if err := dir.SaveUser(t.Context(), identity.User{ID: "operator", Name: "Operator", Disabled: true}, 1); err != nil {
		t.Fatal(err)
	}
	check("alpha", "", "catalog.read", false)
}

func TestPermissionMigrationsSerializeAtStartup(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "startup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.Dialect != "postgres" {
		t.Skip("PostgreSQL migration contract uses the local database test profile")
	}
	u, err := url.Parse(testkit.ConnectionString(t, db, ""))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("pool_max_conns", "1")
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			store, err := permissionpg.Open(ctx, u.String())
			if err != nil {
				t.Error(err)
				return
			}
			defer store.Close()
			if _, err := authorization.New(ctx, store, &identity.Directory{Store: db}); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
}

func TestConcurrentInheritanceCannotCreateCycles(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	stores := []permissions.PermissionStore{testkit.PermissionStore(t, db), testkit.PermissionStore(t, db)}
	for _, id := range []string{"role-a", "role-b"} {
		if err := stores[0].CreateRole(t.Context(), permissions.Role{ID: id, Name: id}); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	var wins atomic.Int32
	var workers sync.WaitGroup
	for i, edge := range [][2]string{{"role-a", "role-b"}, {"role-b", "role-a"}} {
		workers.Go(func() {
			<-start
			if err := stores[i].AddRoleInheritance(t.Context(), edge[0], edge[1]); err == nil {
				wins.Add(1)
			}
		})
	}
	close(start)
	workers.Wait()
	if wins.Load() != 1 {
		t.Fatalf("opposing inheritance edges admitted %d writers", wins.Load())
	}
	for _, id := range []string{"role-a", "role-b"} {
		if _, err := stores[0].ExpandRoles(t.Context(), []string{id}); err != nil {
			t.Fatal(err)
		}
	}
}
