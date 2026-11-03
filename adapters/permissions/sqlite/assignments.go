package sqlite

import (
	"context"
	"errors"
	"slices"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type assignment struct {
	Principal permissions.PrincipalRef
	Value     permissions.RoleAssignment
}

func (s *Store) AssignRole(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role, bindings map[string]any) error {
	if err := principal.Validate(); err != nil {
		return err
	}
	if principal.Kind == permissions.PrincipalRole {
		return errors.New("assign roles to users, groups, or teams")
	}
	if role == "" {
		return errors.New("role is required")
	}
	if !slices.ContainsFunc(builtins, func(r permissions.Role) bool { return r.ID == role }) {
		if _, err := s.RoleDefinition(ctx, role); err != nil {
			return err
		}
	}
	if bindings == nil {
		bindings = map[string]any{}
	}
	a := assignment{Principal: principal, Value: permissions.RoleAssignment{RoleID: role, BindingValues: bindings}}
	if _, err := s.db.Get(ctx, "permission-assignments", key(a)); err == nil {
		return nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	err := s.create(ctx, "permission-assignments", key(a), a)
	if errors.Is(err, storage.ErrConflict) {
		return nil
	}
	return err
}

func (s *Store) UnassignRole(ctx context.Context, principal permissions.PrincipalRef, role string, _ []permissions.Role, bindings map[string]any) error {
	if err := principal.Validate(); err != nil {
		return err
	}
	if bindings == nil {
		bindings = map[string]any{}
	}
	a := assignment{Principal: principal, Value: permissions.RoleAssignment{RoleID: role, BindingValues: bindings}}
	return s.delete(ctx, "permission-assignments", key(a))
}

func (s *Store) RoleAssignmentsForPrincipal(ctx context.Context, p permissions.PrincipalRef) ([]permissions.RoleAssignment, error) {
	items, err := values[assignment](ctx, s.db, "permission-assignments")
	if err != nil {
		return nil, err
	}
	result := []permissions.RoleAssignment{}
	for _, a := range items {
		if a.Principal == p {
			result = append(result, a.Value)
		}
	}
	return result, nil
}

func (s *Store) RoleAssignmentsForRoleID(ctx context.Context, role string) ([]permissions.RoleAssignmentHit, error) {
	items, err := values[assignment](ctx, s.db, "permission-assignments")
	if err != nil {
		return nil, err
	}
	result := []permissions.RoleAssignmentHit{}
	for _, a := range items {
		if a.Value.RoleID == role {
			result = append(result, permissions.RoleAssignmentHit{RoleID: role, PrincipalRef: a.Principal, BindingValues: a.Value.BindingValues})
		}
	}
	return result, nil
}
