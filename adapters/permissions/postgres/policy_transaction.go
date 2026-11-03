package postgres

import (
	"context"
	"database/sql"
	"errors"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type policyTransaction struct{ tx *sql.Tx }

var _ permissions.PermissionStore = (*policyTransaction)(nil)
var _ permissions.BulkGrantStore = (*policyTransaction)(nil)

func (s *Store) WithPolicyTransaction(ctx context.Context, metadata *storage.Store, fn func(*storage.Store, permissions.PermissionStore) error) error {
	if metadata.Dialect != "postgres" {
		return errors.New("policy transaction requires PostgreSQL")
	}
	return metadata.Transaction(ctx, func(bound *storage.Store, tx *sql.Tx) error {
		if err := s.sameDatabase(ctx, tx); err != nil {
			return err
		}
		return fn(bound, &policyTransaction{tx: tx})
	})
}

func (s *policyTransaction) AssignRole(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role, bindings map[string]any) error {
	if err := s.validateRoleAssignment(ctx, principal, role, builtins); err != nil {
		return err
	}
	return writeAssignment(ctx, s.tx, principal, role, bindings, false)
}

func (s *policyTransaction) UnassignRole(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role, bindings map[string]any) error {
	if err := s.validateRoleAssignment(ctx, principal, role, builtins); err != nil {
		return err
	}
	return writeAssignment(ctx, s.tx, principal, role, bindings, true)
}

func (s *policyTransaction) RoleAssignmentsForPrincipal(ctx context.Context, principal permissions.PrincipalRef) ([]permissions.RoleAssignment, error) {
	return (&assignmentTransaction{tx: s.tx}).RoleAssignmentsForPrincipal(ctx, principal)
}

func (s *policyTransaction) RoleAssignmentsForRoleID(ctx context.Context, role string) ([]permissions.RoleAssignmentHit, error) {
	return (&assignmentTransaction{tx: s.tx}).RoleAssignmentsForRoleID(ctx, role)
}

func (s *policyTransaction) validateRoleAssignment(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role) error {
	if err := principal.Validate(); err != nil {
		return err
	}
	if principal.Kind == permissions.PrincipalRole {
		return errors.New("assign roles to users, groups, or teams")
	}
	for _, builtin := range builtins {
		if role != "" && builtin.ID == role {
			return nil
		}
	}
	_, err := s.RoleDefinition(ctx, role)
	return err
}
