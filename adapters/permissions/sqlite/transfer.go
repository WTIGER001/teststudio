package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/installstate"
	"github.com/wtiger001/teststudio/internal/storage"
)

func ExportState(ctx context.Context, db *storage.Store, emit func(installstate.Permission) error) error {
	for _, section := range []struct{ bucket, kind string }{{"permission-roles", "role"}, {"permission-grants", "grant"}, {"permission-inheritance", "edge"}, {"permission-assignments", "assignment"}, {"permission-disabled", "disabled"}} {
		cursor := ""
		for {
			page, err := db.List(ctx, section.bucket, "", cursor, 100)
			if err != nil {
				return err
			}
			for _, row := range page.Items {
				data := json.RawMessage(row.Data)
				if section.kind == "edge" {
					value, err := storage.Decode[edge](row)
					if err != nil {
						return err
					}
					data, err = json.Marshal(installstate.Edge{Parent: value.Parent, Child: value.Child})
					if err != nil {
						return err
					}
				}
				if err := emit(installstate.Permission{Kind: section.kind, Data: data}); err != nil {
					return err
				}
			}
			if page.Cursor == "" {
				break
			}
			cursor = page.Cursor
		}
	}
	return nil
}

func ImportState(ctx context.Context, db *storage.Store, item installstate.Permission) error {
	s := New(db)
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
		if err := validGrant(value); err != nil {
			return err
		}
		return s.create(ctx, "permission-grants", grantID(value.ID), value)
	case "edge":
		value, err := installstate.Decode[installstate.Edge](item.Data)
		if err != nil {
			return err
		}
		if err := requireNewState(ctx, db, "permission-inheritance", key(edge{Parent: value.Parent, Child: value.Child})); err != nil {
			return err
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
		if value.Value.BindingValues == nil {
			value.Value.BindingValues = map[string]any{}
		}
		if err := requireNewState(ctx, db, "permission-assignments", key(assignment{Principal: value.Principal, Value: value.Value})); err != nil {
			return err
		}
		return s.AssignRole(ctx, value.Principal, value.Value.RoleID, builtins, value.Value.BindingValues)
	case "disabled":
		value, err := installstate.Decode[string](item.Data)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(value, "builtin.") {
			return errors.New("a built-in role identifier is required")
		}
		if err := requireNewState(ctx, db, "permission-disabled", value); err != nil {
			return err
		}
		return s.DisableBuiltInRole(ctx, value)
	}
	return errors.New("unsupported permission archive section")
}

func requireNewState(ctx context.Context, db *storage.Store, bucket, id string) error {
	_, err := db.Get(ctx, bucket, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("duplicate native permission state")
}
