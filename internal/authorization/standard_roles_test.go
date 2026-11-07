package authorization_test

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestStandardRolesUseNativeScopedEvaluationAndDenyPrecedence(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "roles.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	for _, id := range []string{"reader", "member", "developer", "contributor", "owner"} {
		if err := dir.SaveUser(t.Context(), identity.User{ID: id, Name: id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	a, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, app := range []string{"website", "admin"} {
		c, err := storage.Put("applications/alpha", app, "alpha", 0, map[string]string{"id": app})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Commit(t.Context(), c); err != nil {
			t.Fatal(err)
		}
	}
	roles := []string{authorization.ReaderRole, authorization.MemberRole, authorization.DeveloperRole, authorization.ContributorRole, authorization.OwnerRole}
	actions := []string{"catalog.read", "runs.create", "catalog.publish", "projects.manage", "members.owners"}
	for i, role := range roles {
		id := []string{"reader", "member", "developer", "contributor", "owner"}[i]
		if err := a.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: id}, role, map[string]any{"team": "alpha", "application": "website"}); err != nil {
			t.Fatal(err)
		}
		for j, action := range actions {
			err := a.CheckApplication(t.Context(), id, "alpha", "website", "", action)
			if (err == nil) != (j <= i) {
				t.Fatalf("%s %s: %v", id, action, err)
			}
			if err := a.Check(t.Context(), id, "alpha", "", action); !errors.Is(err, authorization.ErrForbidden) {
				t.Fatalf("application assignment escaped to project: %v", err)
			}
			if err := a.CheckApplication(t.Context(), id, "alpha", "admin", "", action); !errors.Is(err, authorization.ErrForbidden) {
				t.Fatalf("application assignment escaped to sibling: %v", err)
			}
		}
		if i < 4 {
			for _, action := range []string{"catalog.delete", "evidence.delete", "schedules.delete"} {
				if err := a.CheckApplication(t.Context(), id, "alpha", "website", "", action); !errors.Is(err, authorization.ErrForbidden) {
					t.Fatalf("%s can delete: %v", id, err)
				}
			}
		}
	}
	object := "alpha/application/website"
	deny := permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "owner", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.catalog.read", Effect: permissions.EffectDeny}
	if err := a.Engine.CreateGrant(t.Context(), deny); err != nil {
		t.Fatal(err)
	}
	if err := a.Check(t.Context(), "owner", "alpha", "application/website", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("object deny lost", err)
	}
	if err := a.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "reader"}, authorization.DeveloperRole, map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := a.CheckApplication(t.Context(), "reader", "alpha", "website", "", "catalog.publish"); err != nil {
		t.Fatal("lower direct role removed inheritance", err)
	}
	if err := a.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "member"}, authorization.MemberRole, map[string]any{"team": "alpha", "application": "website"}); err != nil {
		t.Fatal(err)
	}
	if err := a.CheckApplication(t.Context(), "member", "alpha", "website", "", "runs.create"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("revocation did not apply", err)
	}
	expired := time.Now().Add(-time.Minute)
	if err := a.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "developer", TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectDeny, ExpiresAt: &expired}); err != nil {
		t.Fatal(err)
	}
	if err := a.CheckApplication(t.Context(), "developer", "alpha", "website", "", "catalog.read"); err != nil {
		t.Fatal("expired deny retained", err)
	}
}

func TestStandardRoleManifestPreservesLegacyPermissions(t *testing.T) {
	roles := authorization.BuiltInRoles()
	previous := []string{}
	for _, id := range []string{authorization.ReaderRole, authorization.MemberRole, authorization.DeveloperRole, authorization.ContributorRole, authorization.OwnerRole} {
		for _, role := range roles {
			if role.ID == id {
				for _, permission := range previous {
					if !slices.Contains(role.Permissions, permission) {
						t.Fatalf("%s is not cumulative: %s", id, permission)
					}
				}
				previous = role.Permissions
			}
		}
	}
	for _, role := range roles {
		if role.ID == "builtin.teststudio.project-admin" && slices.Contains(role.Permissions, "teststudio.catalog.publish") {
			t.Fatal("legacy administrator broadened")
		}
		if role.ID == "builtin.teststudio.operator" && !slices.Contains(role.Permissions, "teststudio.schedules.delete") {
			t.Fatal("legacy operator lost permission")
		}
		if authorization.StandardRole(role.ID) {
			for _, permission := range role.Permissions {
				if authorization.InstallationPermission(permission) || permission == "teststudio.runs.execute-protected" || permission == "teststudio.evidence.override-capture" {
					t.Fatalf("%s contains advanced authority %s", role.ID, permission)
				}
			}
		}
	}
}

func TestApplicationStandardRolePersistsAcrossDatabaseRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	db, err := testkit.Open(t, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dsn := testkit.ConnectionString(t, db, path)
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "member", Name: "Member"}, 0); err != nil {
		t.Fatal(err)
	}
	change, err := storage.Put("applications/alpha", "app", "alpha", 0, map[string]string{"id": "app"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "member"}, authorization.MemberRole, map[string]any{"team": "alpha", "application": "app"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reopened.Close() })
	fresh, err := authorization.New(t.Context(), testkit.PermissionStore(t, reopened), &identity.Directory{Store: reopened})
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.CheckApplication(t.Context(), "member", "alpha", "app", "", "runs.create"); err != nil {
		t.Fatal("scoped role did not survive restart", err)
	}
	if err := fresh.Check(t.Context(), "member", "alpha", "", "runs.create"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("restarted role escaped to project", err)
	}
}
