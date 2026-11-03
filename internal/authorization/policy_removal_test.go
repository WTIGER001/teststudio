package authorization_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func TestPolicyRemovalKeepsAlternateInheritancePathsAndOtherGrants(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	if err := auth.RemovePolicyGrant(ctx, 1); err == nil {
		t.Fatal("grant removal allowed outside transaction")
	}
	if err := auth.RemoveRoleInheritance(ctx, "top", "left"); err == nil {
		t.Fatal("inheritance removal allowed outside transaction")
	}
	err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		for _, id := range []string{"top", "left", "right", "leaf"} {
			if err := bound.Engine.CreateRole(ctx, permissions.Role{ID: id, Name: id}); err != nil {
				return err
			}
		}
		for _, edge := range [][2]string{{"top", "left"}, {"top", "right"}, {"left", "leaf"}, {"right", "leaf"}} {
			if err := bound.Engine.AddRoleInheritance(ctx, edge[0], edge[1]); err != nil {
				return err
			}
		}
		for _, name := range []string{"teststudio.catalog.read", "teststudio.runs.read"} {
			if err := bound.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: "leaf", TeamScope: "alpha", PermissionName: name, Effect: permissions.EffectAllow}); err != nil {
				return err
			}
		}
		return bound.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "top", nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	grants, err := auth.Engine.GrantsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: "leaf"})
	if err != nil || len(grants) != 2 {
		t.Fatal(grants, err)
	}
	err = auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.RemoveRoleInheritance(ctx, "top", "left"); err != nil {
			return err
		}
		if err := bound.Check(ctx, "operator", "alpha", "", "catalog.read"); err != nil {
			return err
		}
		if expanded, err := bound.Engine.ExpandRoles(ctx, []string{"top"}); err != nil || slices.Contains(expanded, "left") || !slices.Contains(expanded, "leaf") {
			t.Fatal("alternate path closure incorrect", expanded, err)
		}
		if err := bound.RemoveRoleInheritance(ctx, "right", "leaf"); err != nil {
			return err
		}
		if err := bound.Check(ctx, "operator", "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
			t.Fatal("last inheritance path retained access", err)
		}
		return bound.RemovePolicyGrant(ctx, grants[0].ID)
	})
	if err != nil {
		t.Fatal(err)
	}
	remaining, err := auth.Engine.GrantsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: "leaf"})
	if err != nil || len(remaining) != 1 || remaining[0].ID != grants[1].ID {
		t.Fatal("single grant removal changed another grant", remaining, err)
	}
	if err := auth.Check(ctx, "operator", "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("committed closure retained access", err)
	}
	if err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error { return bound.Engine.AddRoleInheritance(ctx, "right", "leaf") }); err != nil {
		t.Fatal(err)
	}
	if err := auth.Check(ctx, "operator", "alpha", "", "runs.read"); err != nil {
		t.Fatal("recreated inheritance path missing", err)
	}
	if err := auth.Check(ctx, "operator", "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("removed grant revived", err)
	}
}

func TestPolicyRemovalRollbackRetainsGrantIDsAndGraph(t *testing.T) {
	auth := transactionFixture(t)
	ctx := t.Context()
	if err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		for _, id := range []string{"parent", "child"} {
			if err := bound.Engine.CreateRole(ctx, permissions.Role{ID: id, Name: id}); err != nil {
				return err
			}
		}
		if err := bound.Engine.AddRoleInheritance(ctx, "parent", "child"); err != nil {
			return err
		}
		if err := bound.Engine.CreateGrant(ctx, permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: "child", TeamScope: "alpha", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow, RestrictedFields: []string{"name"}}); err != nil {
			return err
		}
		return bound.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "parent", nil)
	}); err != nil {
		t.Fatal(err)
	}
	owner := permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: "child"}
	before, err := auth.Engine.GrantsForPrincipal(ctx, owner)
	if err != nil || len(before) != 1 {
		t.Fatal(before, err)
	}
	failure := errors.New("reject graph and grant removal")
	if err := auth.WithPolicyTransaction(ctx, func(bound *authorization.Service) error {
		if err := bound.RemoveRoleInheritance(ctx, "parent", "child"); err != nil {
			return err
		}
		if err := bound.RemovePolicyGrant(ctx, before[0].ID); err != nil {
			return err
		}
		return failure
	}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	after, err := auth.Engine.GrantsForPrincipal(ctx, owner)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("rolled back removal changed the grant", before, after, err)
	}
	if err := auth.Check(ctx, "operator", "alpha", "", "catalog.read"); err != nil {
		t.Fatal("rolled back removal changed inheritance", err)
	}
}
