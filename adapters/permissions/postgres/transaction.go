package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type assignmentTransaction struct {
	assignmentReads
	tx *sql.Tx
}

var _ permissions.PermissionStore = (*assignmentTransaction)(nil)

func (s *Store) WithAssignmentTransaction(ctx context.Context, metadata *storage.Store, fn func(*storage.Store, permissions.PermissionStore) error) error {
	if metadata.Dialect != "postgres" {
		return errors.New("assignment transaction requires PostgreSQL")
	}
	return metadata.Transaction(ctx, func(bound *storage.Store, tx *sql.Tx) error {
		if err := s.sameDatabase(ctx, tx); err != nil {
			return err
		}
		return fn(bound, &assignmentTransaction{assignmentReads: assignmentReads{PermissionStore: s}, tx: tx})
	})
}

func (s *Store) sameDatabase(ctx context.Context, tx *sql.Tx) error {
	var database string
	if err := tx.QueryRowContext(ctx, "SELECT current_database()").Scan(&database); err != nil {
		return err
	}
	if database != s.pool.Config().ConnConfig.Database {
		return errors.New("permission transaction requires the same PostgreSQL database")
	}
	return nil
}

func (s *assignmentTransaction) validate(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role) error {
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
	return nil
}

func assignmentBindings(bindings map[string]any) (string, error) {
	if bindings == nil {
		bindings = map[string]any{}
	}
	raw, err := json.Marshal(bindings)
	return string(raw), err
}

func (s *assignmentTransaction) AssignRole(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role, bindings map[string]any) error {
	if err := s.validate(ctx, principal, role, builtins); err != nil {
		return err
	}
	return writeAssignment(ctx, s.tx, principal, role, bindings, false)
}

func (s *assignmentTransaction) UnassignRole(ctx context.Context, principal permissions.PrincipalRef, role string, builtins []permissions.Role, bindings map[string]any) error {
	if err := s.validate(ctx, principal, role, builtins); err != nil {
		return err
	}
	return writeAssignment(ctx, s.tx, principal, role, bindings, true)
}

func writeAssignment(ctx context.Context, tx *sql.Tx, principal permissions.PrincipalRef, role string, bindings map[string]any, remove bool) error {
	raw, err := assignmentBindings(bindings)
	if err != nil {
		return err
	}
	statement := `INSERT INTO teststudio_permissions.principal_roles (principal_kind,principal_id,role_id,binding_values) VALUES ($1,$2,$3,$4::jsonb) ON CONFLICT (principal_kind,principal_id,role_id,binding_values) DO NOTHING`
	if remove {
		statement = `DELETE FROM teststudio_permissions.principal_roles WHERE principal_kind=$1 AND principal_id=$2 AND role_id=$3 AND binding_values=$4::jsonb`
	}
	_, err = tx.ExecContext(ctx, statement, string(principal.Kind), principal.ID, role, raw)
	return err
}

func (s *assignmentTransaction) RoleAssignmentsForPrincipal(ctx context.Context, principal permissions.PrincipalRef) ([]permissions.RoleAssignment, error) {
	if err := principal.Validate(); err != nil {
		return nil, err
	}
	rows, err := s.tx.QueryContext(ctx, `SELECT role_id,binding_values FROM teststudio_permissions.principal_roles WHERE principal_kind=$1 AND principal_id=$2 ORDER BY id`, string(principal.Kind), principal.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []permissions.RoleAssignment{}
	for rows.Next() {
		var value permissions.RoleAssignment
		var raw []byte
		if err := rows.Scan(&value.RoleID, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &value.BindingValues); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *assignmentTransaction) RoleAssignmentsForRoleID(ctx context.Context, role string) ([]permissions.RoleAssignmentHit, error) {
	rows, err := s.tx.QueryContext(ctx, `SELECT role_id,binding_values,principal_kind,principal_id FROM teststudio_permissions.principal_roles WHERE role_id=$1 ORDER BY id`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []permissions.RoleAssignmentHit{}
	for rows.Next() {
		var value permissions.RoleAssignmentHit
		var raw []byte
		if err := rows.Scan(&value.RoleID, &raw, &value.PrincipalRef.Kind, &value.PrincipalRef.ID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &value.BindingValues); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
