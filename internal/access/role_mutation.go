package access

import (
	"context"
	"errors"
	"slices"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) SaveRolePolicy(ctx context.Context, actor string, input RolePolicyMutation, revision int64) (storage.Record, error) {
	if err := reason(input.Reason); err != nil {
		return storage.Record{}, err
	}
	value, err := normalizeRolePolicy(input.RolePolicy)
	if err != nil {
		return storage.Record{}, err
	}
	var result storage.Record
	err = s.Auth.WithPolicyTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts, SessionIdleTimeout: s.SessionIdleTimeout}
		prior, row, err := bound.loadRolePolicy(ctx, value.ID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if row.Revision > 0 {
			if err := bound.rolePolicyAuthority(ctx, actor, prior, true); err != nil {
				return err
			}
			if prior.Project != value.Project {
				return errors.New("role ownership is permanent")
			}
		} else if err := bound.rolePolicyAuthority(ctx, actor, value, true); err != nil {
			return err
		}
		if row.Revision != revision {
			return storage.ErrConflict
		}
		projectRow, err := bound.assignmentScope(ctx, actor, value.Project, "roles.manage")
		if err != nil {
			return err
		}
		if projectRow.Revision > 0 {
			p, err := storage.Decode[struct {
				Archived bool `json:"archived"`
			}](projectRow)
			if err != nil {
				return err
			}
			if p.Archived {
				return errors.New("restore the archived project before changing its roles")
			}
		}
		if row.Revision > 0 {
			if err := bound.roleRulesAuthority(ctx, actor, prior, false); err != nil {
				return err
			}
		}
		if err := bound.roleRulesAuthority(ctx, actor, value, true); err != nil {
			return err
		}
		if err := bound.roleInheritanceAuthority(ctx, actor, value); err != nil {
			return err
		}
		hadAdmin := false
		if value.Project == "" {
			hadAdmin, err = bound.hasUsableAdministrator(ctx)
			if err != nil {
				return err
			}
		}
		actions := []string{}
		for _, rule := range value.Rules {
			if !slices.Contains(actions, rule.Permission) {
				actions = append(actions, rule.Permission)
			}
		}
		slices.Sort(actions)
		scope := permissions.RoleScopeTeam
		if value.Project == "" {
			scope = permissions.RoleScopeSystem
		}
		role := permissions.Role{ID: value.ID, Name: value.Name, Description: value.Description, Scope: scope, Permissions: actions, IsDisabled: value.Disabled}
		if row.Revision == 0 {
			if _, err := auth.Engine.RoleDefinition(ctx, value.ID); err == nil {
				return storage.ErrConflict
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			if err := auth.Engine.CreateRole(ctx, role); err != nil {
				return err
			}
		} else if err := auth.Engine.UpdateRole(ctx, role); err != nil {
			return err
		}
		if err := bound.materializeRoleGrants(ctx, value); err != nil {
			return err
		}
		if err := bound.materializeRoleInheritance(ctx, prior, value); err != nil {
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
		change, err := storage.Put(rolePolicyBucket(value.Project), value.ID, value.Project, revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(actor, value.Project, "role.policy.save", value.ID, prior, map[string]any{"value": value, "reason": input.Reason})
		if err != nil {
			return err
		}
		changes := []storage.Change{change, audit}
		if row.Revision == 0 {
			index, err := storage.Put("access-role-index", value.ID, value.Project, 0, struct {
				Project string `json:"projectId"`
			}{value.Project})
			if err != nil {
				return err
			}
			changes = append(changes, index)
		}
		if projectRow.Revision > 0 {
			changes = append(changes, storage.Change{Record: projectRow, Assert: true})
		}
		if err := bound.Store.Commit(ctx, changes...); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, rolePolicyBucket(value.Project), value.ID)
		return err
	})
	if err != nil {
		return storage.Record{}, err
	}
	return result, nil
}
