package authorization_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestPolicyTransactionCommitsInheritedObjectGrantsAndMetadataTogether(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	principal := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	object := "alpha/suite/health"
	err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		for _, id := range []string{"reviewer", "reader", "another-reader"} {
			if err := bound.Engine.CreateRole(ctx, permissions.Role{ID: id, Name: "Shared display name", Scope: permissions.RoleScopeTeam, Tags: []string{"review"}}); err != nil {
				return err
			}
		}
		role, err := bound.Engine.RoleDefinition(ctx, "reader")
		if err != nil {
			return err
		}
		role.Description = "Permission-backed reader"
		role.Permissions = []string{"teststudio.catalog.read"}
		if err := bound.Engine.UpdateRole(ctx, role); err != nil {
			return err
		}
		roles, err := bound.Engine.RoleDefinitions(ctx)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(roles, func(r permissions.Role) bool {
			return r.ID == "reader" && r.Description == role.Description && reflect.DeepEqual(r.Tags, role.Tags) && reflect.DeepEqual(r.Permissions, role.Permissions)
		}) {
			t.Fatal("pending role metadata is missing", roles)
		}
		if err := bound.Engine.AddRoleInheritance(ctx, "reviewer", "reader"); err != nil {
			return err
		}
		if err := bound.Engine.AddRoleInheritance(ctx, "reader", "reviewer"); err == nil {
			t.Fatal("inheritance cycle accepted")
		}
		if err := bound.Engine.AssignRole(ctx, principal, "reviewer", map[string]any{"team": "alpha"}); err != nil {
			return err
		}
		expired := time.Now().Add(-time.Minute)
		if err := bound.Engine.CreateGrants(ctx, []permissions.Grant{
			{OwnerKind: permissions.PrincipalRole, OwnerID: "reader", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow},
			{OwnerKind: permissions.PrincipalRole, OwnerID: "reader", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.catalog.read", Effect: permissions.EffectDeny, ExpiresAt: &expired},
			{OwnerKind: permissions.PrincipalUser, OwnerID: principal.ID, TeamScope: "beta", PermissionName: "teststudio.runs.read", Effect: permissions.EffectAllow},
		}); err != nil {
			return err
		}
		if err := bound.Check(ctx, principal.ID, "alpha", "suite/health", "catalog.read"); err != nil {
			return err
		}
		for _, scope := range []struct{ project, object string }{{"beta", "suite/health"}, {"alpha", "suite/another"}, {"alpha", ""}} {
			if err := bound.Check(ctx, principal.ID, scope.project, scope.object, "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
				t.Fatal("object grant crossed its exact scope", scope, err)
			}
		}
		if grants, err := bound.Engine.GrantsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: "reader"}); err != nil || len(grants) != 1 {
			t.Fatal("pending active literal-scope grants unavailable", grants, err)
		}
		if hits, err := bound.Engine.PrincipalsWithGrant(ctx, permissions.Request{TeamID: "alpha", Object: object, Perm: "teststudio.catalog.read"}); err != nil || len(hits) != 1 {
			t.Fatal("pending principal report missing", hits, err)
		}
		page, err := bound.Engine.GetStore().ListGrants(ctx, permissions.GrantQuery{PermissionPrefix: "teststudio.catalog.", Limit: 1})
		if err != nil {
			return err
		}
		if len(page.Grants) != 1 || page.TotalCount != 2 || page.NextCursor == "" {
			t.Fatal("pending grant page missing", page)
		}
		next, err := bound.Engine.GetStore().ListGrants(ctx, permissions.GrantQuery{PermissionPrefix: "teststudio.catalog.", Limit: 1, Cursor: page.NextCursor})
		if err != nil {
			return err
		}
		if len(next.Grants) != 1 || next.Grants[0].ID == page.Grants[0].ID || next.NextCursor != "" {
			t.Fatal("grant continuation duplicated data", next)
		}
		c, _ := storage.Put("audit", "custom-role", "alpha", 0, "role and grants created")
		return bound.Directory.Store.Commit(ctx, c)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(ctx, principal.ID, "alpha", "suite/health", "catalog.read"); err != nil {
		t.Fatal("committed inherited grant missing", err)
	}
	if grants, err := auth.Engine.GrantsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: "reader"}); err != nil || len(grants) != 1 {
		t.Fatal("committed literal-scope grant read differs from transaction", grants, err)
	}
	if _, err := auth.Directory.Store.Get(ctx, "audit", "custom-role"); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyTransactionRollsBackRoleGrantInheritanceOverridesAndAudit(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	if err := auth.Engine.CreateRole(ctx, permissions.Role{ID: "existing", Name: "Existing"}); err != nil {
		t.Fatal(err)
	}
	before, err := auth.Directory.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("administrator check rejects entire policy")
	err = auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.Engine.CreateRole(ctx, permissions.Role{ID: "discarded", Name: "Discarded"}); err != nil {
			return err
		}
		role, err := bound.Engine.RoleDefinition(ctx, "existing")
		if err != nil {
			return err
		}
		role.Name = "Must not survive"
		role.IsDisabled = true
		if err := bound.Engine.UpdateRole(ctx, role); err != nil {
			return err
		}
		if err := bound.Engine.AddRoleInheritance(ctx, "discarded", "existing"); err != nil {
			return err
		}
		if err := bound.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "operator", TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow}); err != nil {
			return err
		}
		if err := bound.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "discarded", nil); err != nil {
			return err
		}
		if err := bound.Engine.DisableBuiltInRole(ctx, "builtin.teststudio.viewer"); err != nil {
			return err
		}
		if role, err := bound.Engine.RoleDefinition(ctx, "builtin.teststudio.viewer"); err != nil || !role.IsDisabled {
			t.Fatal("pending override not visible", role, err)
		}
		c, _ := storage.Put("audit", "discarded", "", 0, "must not survive")
		if err := bound.Directory.Store.Commit(ctx, c); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := auth.Engine.RoleDefinition(ctx, "discarded"); err == nil {
		t.Fatal("role leaked")
	}
	if role, err := auth.Engine.RoleDefinition(ctx, "existing"); err != nil || role.Name != "Existing" || role.IsDisabled {
		t.Fatal("role update leaked", role, err)
	}
	if grants, err := auth.Engine.GrantsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}); err != nil || len(grants) != 0 {
		t.Fatal("grant leaked", grants, err)
	}
	if roles, err := auth.Engine.RoleAssignmentsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}); err != nil || len(roles) != 0 {
		t.Fatal("assignment leaked", roles, err)
	}
	if disabled, err := auth.Engine.DisabledBuiltInRoles(ctx); err != nil || len(disabled) != 0 {
		t.Fatal("override leaked", disabled, err)
	}
	if _, err := auth.Directory.Store.Get(ctx, "audit", "discarded"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("audit leaked", err)
	}
	after, err := auth.Directory.Revision(ctx)
	if err != nil || before.Revision != after.Revision {
		t.Fatal("directory clock leaked", before, after, err)
	}
}

func TestPolicyTransactionPreservesDenyExpiryAndRoleLifecycle(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	p := permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}
	if err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.Engine.CreateRole(ctx, permissions.Role{ID: "reader", Name: "Reader"}); err != nil {
			return err
		}
		if err := bound.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: "reader", TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow}); err != nil {
			return err
		}
		return bound.Engine.AssignRole(ctx, p, "reader", nil)
	}); err != nil {
		t.Fatal(err)
	}
	if err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: p.Kind, OwnerID: p.ID, TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectDeny}); err != nil {
			return err
		}
		if err := bound.Check(ctx, p.ID, "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("pending explicit deny lost", err)
		}
		if hits, err := bound.Engine.PrincipalsWithGrant(ctx, permissions.Request{TeamID: "alpha", Perm: "teststudio.catalog.read"}); err != nil || slices.ContainsFunc(hits, func(h permissions.PrincipalHit) bool { return h.Kind == p.Kind && h.ID == p.ID }) {
			t.Fatal("direct denied principal reported", hits, err)
		}
		if err := bound.Engine.DeleteGrantsForOwner(ctx, p.Kind, p.ID); err != nil {
			return err
		}
		if err := bound.Check(ctx, p.ID, "alpha", "", "catalog.read"); err != nil {
			return err
		}
		role, err := bound.Engine.RoleDefinition(ctx, "reader")
		if err != nil {
			return err
		}
		role.IsDisabled = true
		if err := bound.Engine.UpdateRole(ctx, role); err != nil {
			return err
		}
		if err := bound.Check(ctx, p.ID, "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("pending disabled custom role retained access", err)
		}
		if err := bound.Engine.DeleteRole(ctx, "reader"); err == nil {
			t.Fatal("active role dependencies deleted")
		}
		if err := bound.Engine.UnassignRole(ctx, p, "reader", nil); err != nil {
			return err
		}
		if err := bound.Engine.DeleteGrantsForOwner(ctx, permissions.PrincipalRole, "reader"); err != nil {
			return err
		}
		return bound.Engine.DeleteRole(ctx, "reader")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Engine.RoleDefinition(ctx, "reader"); err == nil {
		t.Fatal("role was not deleted")
	}
}

func TestPolicyTransactionRejectsUnversionedStoreWrappers(t *testing.T) {
	auth := transactionFixture(t)
	engine, err := authorization.NewEngine(t.Context(), struct{ permissions.PermissionStore }{auth.Engine.GetStore()}, auth.Directory)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := &authorization.Service{Engine: engine, Directory: auth.Directory}
	if err := wrapped.WithPolicyTransaction(t.Context(), func(*authorization.Service) error { t.Fatal("unsafe callback executed"); return nil }); err == nil {
		t.Fatal("policy transaction fell back to non-atomic writes")
	}
}
