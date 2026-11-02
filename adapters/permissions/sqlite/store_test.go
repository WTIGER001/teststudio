package sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	permissionstore "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestScopedRolesDenyExpiryAndRevocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "permissions.db")
	db, err := storage.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "alice", Name: "Alice"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := dir.SaveGroup(t.Context(), identity.Group{ID: "editors", Name: "Editors"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := dir.AddMember(t.Context(), identity.Membership{UserID: "alice", GroupID: "editors"}); err != nil {
		t.Fatal(err)
	}
	store := permissionstore.New(db)
	auth, err := authorization.New(t.Context(), store, dir)
	if err != nil {
		t.Fatal(err)
	}
	p := permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: "editors"}
	for _, project := range []string{"alpha", "beta"} {
		if err := auth.Engine.AssignRole(t.Context(), p, "builtin.teststudio.editor", map[string]any{"team": project}); err != nil {
			t.Fatal(err)
		}
	}
	for _, project := range []string{"alpha", "beta"} {
		if err := auth.Check(t.Context(), "alice", project, "", "catalog.update"); err != nil {
			t.Fatal(err)
		}
	}
	if err := auth.Check(t.Context(), "alice", "gamma", "", "catalog.update"); err == nil {
		t.Fatal("role crossed project boundary")
	}
	expired := time.Now().Add(-time.Minute)
	if err := auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "alice", TeamScope: "alpha", Effect: permissions.EffectDeny, PermissionName: "teststudio.catalog.update", ExpiresAt: &expired}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), "alice", "alpha", "", "catalog.update"); err != nil {
		t.Fatalf("expired deny remained active: %v", err)
	}
	if err := auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "alice", TeamScope: "alpha", Effect: permissions.EffectDeny, PermissionName: "teststudio.catalog.update"}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), "alice", "alpha", "", "catalog.update"); err == nil {
		t.Fatal("explicit deny lost to group role")
	}
	if err := auth.Engine.UnassignRole(t.Context(), p, "builtin.teststudio.editor", map[string]any{"team": "beta"}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), "alice", "beta", "", "catalog.update"); err == nil {
		t.Fatal("revoked binding retained access")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = storage.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth, err = authorization.New(t.Context(), permissionstore.New(db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), "alice", "alpha", "", "catalog.read"); err != nil {
		t.Fatalf("lost durable role: %v", err)
	}
	if err := auth.Check(t.Context(), "alice", "alpha", "", "catalog.update"); err == nil {
		t.Fatal("lost durable deny")
	}
}

func TestCustomRolesInheritanceAndDisabledOverrides(t *testing.T) {
	db, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "roles.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := permissionstore.New(db)
	for _, id := range []string{"reviewer", "reader", "auditor"} {
		if err := store.CreateRole(t.Context(), permissions.Role{ID: id, Name: id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.AddRoleInheritance(t.Context(), "reviewer", "reader"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddRoleInheritance(t.Context(), "reader", "auditor"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddRoleInheritance(t.Context(), "auditor", "reviewer"); err == nil {
		t.Fatal("accepted inheritance cycle")
	}
	roles, err := store.ExpandRoles(t.Context(), []string{"reviewer"})
	if err != nil || len(roles) != 3 {
		t.Fatalf("transitive roles: %v %v", roles, err)
	}
	if err := store.DisableBuiltInRole(t.Context(), "builtin.teststudio.viewer"); err != nil {
		t.Fatal(err)
	}
	disabled, err := store.DisabledBuiltInRoles(t.Context())
	if err != nil || len(disabled) != 1 {
		t.Fatalf("disabled overrides: %v %v", disabled, err)
	}
	if err := store.EnableBuiltInRole(t.Context(), disabled[0]); err != nil {
		t.Fatal(err)
	}
	disabled, err = store.DisabledBuiltInRoles(t.Context())
	if err != nil || len(disabled) != 0 {
		t.Fatalf("enabled overrides: %v %v", disabled, err)
	}
}

func TestPrincipalLookupOmitsExpiredAndDeniedGrants(t *testing.T) {
	db, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "principal.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := permissionstore.New(db)
	expired := time.Now().Add(-time.Minute)
	for _, owner := range []string{"active", "expired", "denied"} {
		g := permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: owner, Effect: permissions.EffectAllow, TeamScope: "project", PermissionName: "teststudio.catalog.read"}
		if owner == "expired" {
			g.ExpiresAt = &expired
		}
		if err := store.CreateGrant(t.Context(), g); err != nil {
			t.Fatal(err)
		}
		if owner == "denied" {
			g.Effect = permissions.EffectDeny
			if err := store.CreateGrant(t.Context(), g); err != nil {
				t.Fatal(err)
			}
		}
	}
	hits, err := store.PrincipalsWithGrant(t.Context(), permissions.Request{TeamID: "project", Perm: "teststudio.catalog.read"})
	if err != nil || len(hits) != 1 || hits[0].ID != "active" {
		t.Fatalf("active principal hits: %+v %v", hits, err)
	}
}
