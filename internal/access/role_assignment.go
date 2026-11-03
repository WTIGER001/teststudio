package access

import (
	"context"
	"errors"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) roleForScope(ctx context.Context, project, id string) (permissions.Role, error) {
	if strings.HasPrefix(id, "builtin.") {
		if _, ok := builtinRole(project, id); !ok {
			return permissions.Role{}, errors.New("choose a predefined role for this scope")
		}
		return s.Auth.Engine.RoleDefinition(ctx, id)
	}
	value, _, err := s.loadRolePolicy(ctx, id)
	if err != nil {
		return permissions.Role{}, err
	}
	if value.Project != project {
		return permissions.Role{}, storage.ErrNotFound
	}
	role, err := s.Auth.Engine.RoleDefinition(ctx, id)
	if err != nil {
		return role, err
	}
	if role.IsDisabled != value.Disabled {
		return role, errors.New("role state does not match its managed policy")
	}
	return role, nil
}

func (s *Service) customRoleDelegation(ctx context.Context, actor, project, id string) error {
	path, seen := map[string]bool{}, map[string]bool{}
	var visit func(string, int) error
	visit = func(current string, depth int) error {
		if path[current] {
			return errors.New("role inheritance cycle")
		}
		if depth > 16 || len(seen) > 256 {
			return errors.New("role inheritance exceeds its bound")
		}
		if seen[current] {
			return nil
		}
		value, _, err := s.loadRolePolicy(ctx, current)
		if err != nil {
			return err
		}
		if value.Project != project {
			return storage.ErrNotFound
		}
		if err := s.rolePolicyAuthority(ctx, actor, value, false); err != nil {
			return err
		}
		if err := s.Auth.Check(ctx, actor, project, "role/"+current, "permissions.grant"); err != nil {
			return err
		}
		if err := s.roleRulesAuthority(ctx, actor, value, false); err != nil {
			return err
		}
		path[current] = true
		seen[current] = true
		for _, child := range value.Inherits {
			if err := visit(child, depth+1); err != nil {
				return err
			}
		}
		delete(path, current)
		return nil
	}
	return visit(id, 0)
}
