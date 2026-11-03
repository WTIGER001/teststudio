package access

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type BuiltinRoleView struct {
	authorization.RoleDefinition
	Scope     string `json:"scope"`
	Disabled  bool   `json:"disabled"`
	CanManage bool   `json:"canManage"`
}

type BuiltinRoleRecord struct {
	ID               string          `json:"id"`
	Revision         int64           `json:"revision"`
	Updated          *time.Time      `json:"updatedAt,omitempty"`
	Data             BuiltinRoleView `json:"data"`
	metadataRevision int64
}

func builtinRevision(metadata int64, disabled bool) int64 {
	revision := 2*metadata + 1
	if disabled {
		revision++
	}
	return revision
}

type BuiltinRoleUpdate struct {
	Disabled bool   `json:"disabled"`
	Reason   string `json:"reason"`
}

type builtinRoleState struct {
	ID       string `json:"id"`
	Disabled bool   `json:"disabled"`
}

func predefinedRole(id string) (authorization.RoleDefinition, error) {
	for _, value := range authorization.BuiltInRoles() {
		if value.ID == id {
			return value, nil
		}
	}
	return authorization.RoleDefinition{}, storage.ErrNotFound
}

func (s *Service) builtinRoleAuthority(ctx context.Context, actor string, role authorization.RoleDefinition, manage bool) error {
	actions := []string{"roles.read"}
	if manage {
		actions = []string{"roles.manage", "permissions.grant", "system.manage"}
	}
	for _, action := range actions {
		if err := s.Auth.Check(ctx, actor, "", "", action); err != nil {
			return err
		}
		if err := s.Auth.Check(ctx, actor, "", "role/"+role.ID, action); err != nil {
			return err
		}
	}
	if manage {
		for _, action := range role.Permissions {
			if err := s.Auth.Check(ctx, actor, "", "", action); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) builtinRoleRecord(ctx context.Context, actor string, definition authorization.RoleDefinition) (BuiltinRoleRecord, error) {
	if err := s.builtinRoleAuthority(ctx, actor, definition, false); err != nil {
		return BuiltinRoleRecord{}, err
	}
	disabled, err := s.Auth.Engine.DisabledBuiltInRoles(ctx)
	if err != nil {
		return BuiltinRoleRecord{}, err
	}
	row, err := s.Store.Get(ctx, "access-builtin-roles", definition.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return BuiltinRoleRecord{}, err
	}
	state := slices.Contains(disabled, definition.ID)
	view := BuiltinRoleRecord{ID: definition.ID, Revision: builtinRevision(row.Revision, state), metadataRevision: row.Revision, Data: BuiltinRoleView{RoleDefinition: definition, Scope: "project", Disabled: state}}
	if definition.ID == "builtin.teststudio.installation-admin" {
		view.Data.Scope = "installation"
	}
	if row.Revision > 0 {
		stored, err := storage.Decode[builtinRoleState](row)
		if err != nil {
			return view, err
		}
		if stored.ID != definition.ID || row.Project != "" || stored.Disabled != state {
			return view, storage.ErrConflict
		}
		view.Updated = &row.Updated
	}
	view.Data.CanManage, err = roleCapability(func() error { return s.builtinRoleAuthority(ctx, actor, definition, true) })
	return view, err
}

func (s *Service) BuiltinRoles(ctx context.Context, actor string) ([]BuiltinRoleRecord, error) {
	if err := s.Auth.Check(ctx, actor, "", "", "roles.read"); err != nil {
		return nil, err
	}
	result := []BuiltinRoleRecord{}
	for _, definition := range authorization.BuiltInRoles() {
		row, err := s.builtinRoleRecord(ctx, actor, definition)
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}

func (s *Service) ReadBuiltinRole(ctx context.Context, actor, id string) (BuiltinRoleRecord, error) {
	if err := s.Auth.Check(ctx, actor, "", "", "roles.read"); err != nil {
		return BuiltinRoleRecord{}, err
	}
	definition, err := predefinedRole(id)
	if err != nil {
		return BuiltinRoleRecord{}, err
	}
	return s.builtinRoleRecord(ctx, actor, definition)
}

func (s *Service) UpdateBuiltinRole(ctx context.Context, actor, id string, input BuiltinRoleUpdate, revision int64) (BuiltinRoleRecord, error) {
	if err := reason(input.Reason); err != nil {
		return BuiltinRoleRecord{}, err
	}
	if revision < 1 {
		return BuiltinRoleRecord{}, errors.New("a positive built-in role revision is required")
	}
	definition, err := predefinedRole(id)
	if err != nil {
		return BuiltinRoleRecord{}, err
	}
	var result BuiltinRoleRecord
	err = s.Auth.WithPolicyTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts, SessionIdleTimeout: s.SessionIdleTimeout}
		if err := bound.builtinRoleAuthority(ctx, actor, definition, true); err != nil {
			return err
		}
		before, err := bound.builtinRoleRecord(ctx, actor, definition)
		if err != nil {
			return err
		}
		if before.Revision != revision {
			return storage.ErrConflict
		}
		hadAdmin, err := bound.hasUsableAdministrator(ctx)
		if err != nil {
			return err
		}
		if input.Disabled {
			err = auth.Engine.DisableBuiltInRole(ctx, id)
		} else {
			err = auth.Engine.EnableBuiltInRole(ctx, id)
		}
		if err != nil {
			return err
		}
		if hadAdmin {
			remains, err := bound.hasUsableAdministrator(ctx)
			if err != nil {
				return err
			}
			if !remains {
				return errors.New("the change would remove the last usable installation administrator")
			}
		}
		state := builtinRoleState{ID: id, Disabled: input.Disabled}
		change, err := storage.Put("access-builtin-roles", id, "", before.metadataRevision, state)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(actor, "", "role.builtin.state", id, before.Data, map[string]any{"value": state, "reason": input.Reason})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, audit); err != nil {
			return err
		}
		row, err := bound.Store.Get(ctx, "access-builtin-roles", id)
		if err != nil {
			return err
		}
		result = BuiltinRoleRecord{ID: id, Revision: builtinRevision(row.Revision, input.Disabled), metadataRevision: row.Revision, Updated: &row.Updated, Data: before.Data}
		result.Data.Disabled = input.Disabled
		result.Data.CanManage, err = roleCapability(func() error { return bound.builtinRoleAuthority(ctx, actor, definition, true) })
		return err
	})
	if err != nil {
		return BuiltinRoleRecord{}, err
	}
	return result, nil
}
