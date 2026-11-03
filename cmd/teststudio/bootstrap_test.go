package main

import (
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func bootstrapFixture(t *testing.T) (*authorization.Service, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	db, err := testkit.Open(t, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	return auth, testkit.ConnectionString(t, db, path)
}

func TestBootstrapPreservesLocalRoleRevocationAcrossRestartAndExplicitRecovery(t *testing.T) {
	auth, dsn := bootstrapFixture(t)
	if err := bootstrapLocalAdministrator(t.Context(), auth, false); err != nil {
		t.Fatal(err)
	}
	principal := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "local-administrator"}
	if err := auth.Check(t.Context(), principal.ID, "", "", "system.manage"); err != nil {
		t.Fatal(err)
	}
	provider, _ := storage.Put("providers", "example", "", 0, map[string]any{"issuer": "https://identity.example.test", "clientId": "teststudio", "disabled": false})
	if err := auth.Directory.Store.Commit(t.Context(), provider); err != nil {
		t.Fatal(err)
	}
	backup, err := auth.Directory.ResolveExternal(t.Context(), "https://identity.example.test", "backup", "Backup", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	administration := &access.Service{Store: auth.Directory.Store, Auth: auth, LocalUser: principal.ID}
	if _, err := administration.CreateAssignment(t.Context(), principal.ID, "", access.AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: backup.ID, RoleID: "builtin.teststudio.installation-admin", Reason: "Keep a configured backup"}); err != nil {
		t.Fatal(err)
	}
	removed := access.AssignmentDefinition{PrincipalKind: principal.Kind, PrincipalID: principal.ID, RoleID: "builtin.teststudio.installation-admin", Reason: "Use provider administration"}
	if err := administration.RemoveAssignment(t.Context(), backup.ID, "", removed, 1); err != nil {
		t.Fatal(err)
	}
	if err := auth.Directory.Store.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth, err = authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrapLocalAdministrator(t.Context(), auth, false); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), principal.ID, "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("ordinary restart restored revoked administration", err)
	}
	sessionService := &sessions.Service{Store: db, Directory: auth.Directory}
	old, err := sessionService.Create(t.Context(), httptest.NewRecorder(), principal.ID, "local", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrapLocalAdministrator(t.Context(), auth, true); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), principal.ID, "", "", "system.manage"); err != nil {
		t.Fatal("explicit recovery did not restore administration", err)
	}
	if _, err := sessionService.Validate(t.Context(), old.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("recovery revived a prior cookie", err)
	}
	audit, err := db.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, row := range audit {
		value, err := storage.Decode[struct {
			Action string `json:"action"`
		}](row)
		if err != nil {
			t.Fatal(err)
		}
		counts[value.Action]++
	}
	if counts["identity.bootstrap"] != 1 || counts["identity.local-recovery"] != 1 || counts["role.revoke"] != 1 {
		t.Fatal("bootstrap, revocation, or recovery audit count", counts)
	}
	administration = &access.Service{Store: db, Auth: auth, LocalUser: principal.ID}
	current, err := administration.ReadAssignment(t.Context(), backup.ID, "", removed)
	if err != nil || current.Revision <= 1 || current.Data.Managed || current.Updated != nil {
		t.Fatal("recovery reused an obsolete assignment revision or management time", current, err)
	}
	if err := administration.RemoveAssignment(t.Context(), backup.ID, "", removed, 1); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("obsolete removal revoked recovered access", err)
	}
}

func TestBootstrapAdoptsLegacyLocalProfilesWithoutRestoringAccessOrState(t *testing.T) {
	for _, assigned := range []bool{false, true} {
		t.Run(map[bool]string{true: "assigned", false: "revoked"}[assigned], func(t *testing.T) {
			auth, _ := bootstrapFixture(t)
			user := identity.User{ID: "local-administrator", Name: "Existing", Disabled: true, SessionEpoch: 4}
			if err := auth.Directory.SaveUser(t.Context(), user, 0); err != nil {
				t.Fatal(err)
			}
			if assigned {
				if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: user.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
					t.Fatal(err)
				}
			}
			for range 2 {
				if err := bootstrapLocalAdministrator(t.Context(), auth, false); err != nil {
					t.Fatal(err)
				}
			}
			after, err := auth.Directory.User(t.Context(), user.ID)
			if err != nil || !after.Disabled || after.SessionEpoch != 4 || after.Name != "Existing" {
				t.Fatal("ordinary bootstrap changed a retained profile", after, err)
			}
			roles, err := auth.Engine.RoleAssignmentsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: user.ID})
			expected := 0
			if assigned {
				expected = 1
			}
			if err != nil || len(roles) != expected {
				t.Fatal("legacy assignment state changed", roles, err)
			}
		})
	}
}

func TestBootstrapExternalRevocationAndDisablingDoNotBlockStartupOrRestoreAccess(t *testing.T) {
	auth, _ := bootstrapFixture(t)
	issuer, subject := "https://identity.example.test", "initial-administrator"
	if err := bootstrapExternalAdministrator(t.Context(), auth, issuer, subject); err != nil {
		t.Fatal(err)
	}
	link, err := auth.Directory.ExternalLink(t.Context(), issuer, subject)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), link.UserID, "", "", "system.manage"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: link.UserID}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapExternalAdministrator(t.Context(), auth, issuer, subject); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(t.Context(), link.UserID, "", "", "system.manage"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("external bootstrap restored revoked administration", err)
	}
	user, err := auth.Directory.User(t.Context(), link.UserID)
	if err != nil {
		t.Fatal(err)
	}
	user.Disabled = true
	if err := auth.Directory.SaveUser(t.Context(), user, 1); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapExternalAdministrator(t.Context(), auth, issuer, subject); err != nil {
		t.Fatal("disabled seed account blocked startup", err)
	}
	if err := bootstrapExternalAdministrator(t.Context(), auth, issuer, ""); err == nil {
		t.Fatal("partial bootstrap identity accepted")
	}
}

func TestBootstrapExternalAdoptsExistingIdentityWithoutGrantingAdministration(t *testing.T) {
	auth, _ := bootstrapFixture(t)
	issuer, subject := "https://identity.example.test", "existing-user"
	user, err := auth.Directory.ResolveExternal(t.Context(), issuer, subject, "Existing user", "", false, true)
	if err != nil {
		t.Fatal(err)
	}
	user.Disabled = true
	if err := auth.Directory.SaveUser(t.Context(), user, 1); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapExternalAdministrator(t.Context(), auth, issuer, subject); err != nil {
		t.Fatal("disabled legacy identity blocked adoption", err)
	}
	roles, err := auth.Engine.RoleAssignmentsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: user.ID})
	if err != nil || len(roles) != 0 {
		t.Fatal("existing identity gained administration", roles, err)
	}
}
