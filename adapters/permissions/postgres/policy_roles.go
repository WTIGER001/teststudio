package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

const roleColumns = `id,code,coalesce(description,''),built_in,is_disabled,scope,to_json(tags),to_json(permissions)`

func scanRole(row interface{ Scan(...any) error }) (permissions.Role, error) {
	var r permissions.Role
	var tags, actions []byte
	if err := row.Scan(&r.ID, &r.Name, &r.Description, &r.BuiltIn, &r.IsDisabled, &r.Scope, &tags, &actions); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r, storage.ErrNotFound
		}
		return r, err
	}
	if err := json.Unmarshal(tags, &r.Tags); err != nil {
		return r, err
	}
	err := json.Unmarshal(actions, &r.Permissions)
	return r, err
}

func (s *policyTransaction) RoleDefinition(ctx context.Context, id string) (permissions.Role, error) {
	return scanRole(s.tx.QueryRowContext(ctx, `SELECT `+roleColumns+` FROM teststudio_permissions.roles WHERE id=$1`, id))
}

func (s *policyTransaction) RoleDefinitions(ctx context.Context) ([]permissions.Role, error) {
	rows, err := s.tx.QueryContext(ctx, `SELECT `+roleColumns+` FROM teststudio_permissions.roles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []permissions.Role{}
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func roleJSON(role permissions.Role) (string, string, error) {
	if err := role.Validate(); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(role.Name) == "" {
		return "", "", errors.New("role name is required")
	}
	if strings.HasPrefix(role.ID, "builtin.") || role.BuiltIn {
		return "", "", errors.New("built-in roles are immutable")
	}
	tags, err := json.Marshal(role.Tags)
	if err != nil {
		return "", "", err
	}
	actions, err := json.Marshal(role.Permissions)
	return string(tags), string(actions), err
}

func (s *policyTransaction) CreateRole(ctx context.Context, role permissions.Role) error {
	tags, actions, err := roleJSON(role)
	if err != nil {
		return err
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.roles (id,code,description,built_in,is_disabled,scope,tags,permissions)
VALUES ($1,$2,$3,false,$4,$5,ARRAY(SELECT jsonb_array_elements_text(nullif($6::jsonb,'null'::jsonb))),ARRAY(SELECT jsonb_array_elements_text(nullif($7::jsonb,'null'::jsonb))))`, role.ID, role.Name, role.Description, role.IsDisabled, string(role.Scope), tags, actions)
	if err != nil {
		return err
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.role_closure (ancestor_role_id,descendant_role_id,depth) VALUES ($1,$1,0)`, role.ID)
	return err
}

func (s *policyTransaction) UpdateRole(ctx context.Context, role permissions.Role) error {
	tags, actions, err := roleJSON(role)
	if err != nil {
		return err
	}
	result, err := s.tx.ExecContext(ctx, `UPDATE teststudio_permissions.roles SET code=$2,description=$3,is_disabled=$4,scope=$5,tags=ARRAY(SELECT jsonb_array_elements_text(nullif($6::jsonb,'null'::jsonb))),permissions=ARRAY(SELECT jsonb_array_elements_text(nullif($7::jsonb,'null'::jsonb))),updated_at=now() WHERE id=$1 AND NOT built_in`, role.ID, role.Name, role.Description, role.IsDisabled, string(role.Scope), tags, actions)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return storage.ErrNotFound
	}
	return err
}

func (s *policyTransaction) DeleteRole(ctx context.Context, id string) error {
	if strings.HasPrefix(id, "builtin.") {
		return errors.New("built-in roles are immutable")
	}
	r, err := s.RoleDefinition(ctx, id)
	if err != nil {
		return err
	}
	if r.BuiltIn {
		return errors.New("built-in roles are immutable")
	}
	var used bool
	if err := s.tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teststudio_permissions.principal_roles WHERE role_id=$1) OR EXISTS(SELECT 1 FROM teststudio_permissions.role_inheritance WHERE parent_role_id=$1 OR child_role_id=$1) OR EXISTS(SELECT 1 FROM teststudio_permissions.permission_grants WHERE owner_kind='role' AND owner_id=$1)`, id).Scan(&used); err != nil {
		return err
	}
	if used {
		return errors.New("remove role assignments, inheritance, and grants before deleting the role")
	}
	if _, err := s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.role_closure WHERE ancestor_role_id=$1 OR descendant_role_id=$1`, id); err != nil {
		return err
	}
	_, err = s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.roles WHERE id=$1`, id)
	return err
}

func (s *policyTransaction) DisableBuiltInRole(ctx context.Context, id string) error {
	if !strings.HasPrefix(id, "builtin.") {
		return errors.New("a built-in role is required")
	}
	_, err := s.tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.disabled_builtin_roles (role_id) VALUES ($1) ON CONFLICT DO NOTHING`, id)
	return err
}

func (s *policyTransaction) EnableBuiltInRole(ctx context.Context, id string) error {
	if !strings.HasPrefix(id, "builtin.") {
		return errors.New("a built-in role is required")
	}
	_, err := s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.disabled_builtin_roles WHERE role_id=$1`, id)
	return err
}

func (s *policyTransaction) DisabledBuiltInRoles(ctx context.Context) ([]string, error) {
	rows, err := s.tx.QueryContext(ctx, `SELECT role_id FROM teststudio_permissions.disabled_builtin_roles ORDER BY role_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
