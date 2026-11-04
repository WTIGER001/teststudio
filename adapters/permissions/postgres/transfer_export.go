package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/wtiger001/teststudio/internal/installstate"
)

func ExportState(ctx context.Context, tx *sql.Tx, emit func(installstate.Permission) error) error {
	write := func(kind string, value any) error {
		item, err := installstate.Encode(kind, value)
		if err != nil {
			return err
		}
		return emit(item)
	}
	for _, section := range []struct{ kind, query string }{
		{"role", `SELECT ` + roleColumns + ` FROM teststudio_permissions.roles ORDER BY id`},
		{"grant", `SELECT ` + grantColumns + ` FROM teststudio_permissions.permission_grants ORDER BY id`},
		{"edge", `SELECT parent_role_id,child_role_id FROM teststudio_permissions.role_inheritance ORDER BY parent_role_id,child_role_id`},
		{"assignment", `SELECT principal_kind,principal_id,role_id,binding_values FROM teststudio_permissions.principal_roles ORDER BY id`},
		{"disabled", `SELECT role_id FROM teststudio_permissions.disabled_builtin_roles ORDER BY role_id`},
	} {
		rows, err := tx.QueryContext(ctx, section.query)
		if err != nil {
			return err
		}
		for rows.Next() {
			var value any
			switch section.kind {
			case "role":
				value, err = scanRole(rows)
			case "grant":
				value, err = scanGrant(rows)
			case "edge":
				var edge installstate.Edge
				err = rows.Scan(&edge.Parent, &edge.Child)
				value = edge
			case "assignment":
				var assignment installstate.Assignment
				var data []byte
				err = rows.Scan(&assignment.Principal.Kind, &assignment.Principal.ID, &assignment.Value.RoleID, &data)
				if err == nil {
					assignment.Value.BindingValues, err = installstate.Decode[map[string]any](json.RawMessage(data))
				}
				value = assignment
			case "disabled":
				var id string
				err = rows.Scan(&id)
				value = id
			}
			if err == nil {
				err = write(section.kind, value)
			}
			if err != nil {
				rows.Close()
				return err
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func GrantSequence(ctx context.Context, tx *sql.Tx) (int64, error) {
	var value int64
	var called bool
	err := tx.QueryRowContext(ctx, `SELECT last_value,is_called FROM teststudio_permissions.permission_grants_id_seq`).Scan(&value, &called)
	if !called {
		return 0, err
	}
	return value, err
}

func ResetGrantSequence(ctx context.Context, tx *sql.Tx, value int64) error {
	_, err := tx.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('teststudio_permissions.permission_grants','id'),$1,$2)`, max(1, value), value > 0)
	return err
}

func EmptyState(ctx context.Context, tx *sql.Tx) (bool, error) {
	var present bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teststudio_permissions.roles) OR EXISTS(SELECT 1 FROM teststudio_permissions.permission_grants) OR EXISTS(SELECT 1 FROM teststudio_permissions.principal_roles) OR EXISTS(SELECT 1 FROM teststudio_permissions.role_inheritance) OR EXISTS(SELECT 1 FROM teststudio_permissions.role_closure) OR EXISTS(SELECT 1 FROM teststudio_permissions.disabled_builtin_roles)`).Scan(&present)
	return !present, err
}
