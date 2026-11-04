package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/installstate"
)

func ImportState(ctx context.Context, tx *sql.Tx, item installstate.Permission) error {
	s := &policyTransaction{tx: tx}
	switch item.Kind {
	case "role":
		value, err := installstate.Decode[permissions.Role](item.Data)
		if err != nil {
			return err
		}
		return s.CreateRole(ctx, value)
	case "grant":
		value, err := installstate.Decode[permissions.Grant](item.Data)
		if err != nil {
			return err
		}
		if value.ID < 1 {
			return errors.New("permission grant identifier must be positive")
		}
		if err := validatePolicyGrant(value); err != nil {
			return err
		}
		fields, err := json.Marshal(value.RestrictedFields)
		if err != nil {
			return err
		}
		variables, err := json.Marshal(value.VariableSpec)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.permission_grants (id,owner_kind,owner_id,effect,team_scope,object_scope,permission_name,expires_at,field_allowlist,variable_spec) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,CASE WHEN $9::jsonb='null'::jsonb THEN NULL ELSE ARRAY(SELECT jsonb_array_elements_text($9::jsonb)) END,$10::jsonb)`, value.ID, string(value.OwnerKind), value.OwnerID, string(value.Effect), value.TeamScope, value.ObjectScope, value.PermissionName, value.ExpiresAt, string(fields), string(variables))
		return err
	case "edge":
		value, err := installstate.Decode[installstate.Edge](item.Data)
		if err != nil {
			return err
		}
		var present bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teststudio_permissions.role_inheritance WHERE parent_role_id=$1 AND child_role_id=$2)`, value.Parent, value.Child).Scan(&present); err != nil {
			return err
		}
		if present {
			return errors.New("duplicate permission edge")
		}
		return s.AddRoleInheritance(ctx, value.Parent, value.Child)
	case "assignment":
		value, err := installstate.Decode[installstate.Assignment](item.Data)
		if err != nil {
			return err
		}
		builtins := []permissions.Role{}
		if strings.HasPrefix(value.Value.RoleID, "builtin.") {
			builtins = append(builtins, permissions.Role{ID: value.Value.RoleID})
		}
		bindings, err := assignmentBindings(value.Value.BindingValues)
		if err != nil {
			return err
		}
		var present bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teststudio_permissions.principal_roles WHERE principal_kind=$1 AND principal_id=$2 AND role_id=$3 AND binding_values=$4::jsonb)`, string(value.Principal.Kind), value.Principal.ID, value.Value.RoleID, bindings).Scan(&present); err != nil {
			return err
		}
		if present {
			return errors.New("duplicate role assignment")
		}
		return s.AssignRole(ctx, value.Principal, value.Value.RoleID, builtins, value.Value.BindingValues)
	case "disabled":
		value, err := installstate.Decode[string](item.Data)
		if err != nil {
			return err
		}
		var present bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teststudio_permissions.disabled_builtin_roles WHERE role_id=$1)`, value).Scan(&present); err != nil {
			return err
		}
		if present {
			return errors.New("duplicate built-in override")
		}
		return s.DisableBuiltInRole(ctx, value)
	}
	return errors.New("unsupported permission archive section")
}
