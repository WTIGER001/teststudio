package access

import (
	"context"
	"errors"
	"slices"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RolePolicyDetail struct {
	storage.Record
	CanManage   bool `json:"canManage"`
	Delegatable bool `json:"delegatable"`
}

type RoleChoicePage struct {
	Items   []RoleView `json:"items"`
	Cursor  string     `json:"cursor,omitempty"`
	Partial bool       `json:"partial"`
}

type RolePolicyCatalog struct {
	Page
	CanCreate bool `json:"canCreate"`
}

func (s *Service) RolePolicies(ctx context.Context, actor, project, cursor string, limit int) (RolePolicyCatalog, error) {
	page, err := s.ListRolePolicies(ctx, actor, project, cursor, limit)
	if err != nil {
		return RolePolicyCatalog{}, err
	}
	result := RolePolicyCatalog{Page: page}
	active, err := s.roleProjectActive(ctx, actor, project)
	if err != nil || !active {
		return result, err
	}
	result.CanCreate, err = roleCapability(func() error {
		for _, name := range []string{"roles.manage", "permissions.grant"} {
			if err := s.Auth.Check(ctx, actor, project, "", name); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) roleProjectActive(ctx context.Context, actor, project string) (bool, error) {
	row, err := s.assignmentScope(ctx, actor, project, "roles.read")
	if err != nil {
		return false, err
	}
	if project == "" {
		return true, nil
	}
	value, err := storage.Decode[struct {
		Archived bool `json:"archived"`
	}](row)
	return !value.Archived, err
}

func roleCapability(check func() error) (bool, error) {
	if err := check(); err != nil {
		if errors.Is(err, authorization.ErrForbidden) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) roleDelegatable(ctx context.Context, actor string, value RolePolicy) (bool, error) {
	if value.Disabled {
		return false, nil
	}
	return roleCapability(func() error {
		for _, permission := range []string{"members.manage", "permissions.grant"} {
			if err := s.Auth.Check(ctx, actor, value.Project, "", permission); err != nil {
				return err
			}
			if err := s.Auth.Check(ctx, actor, value.Project, "role/"+value.ID, permission); err != nil {
				return err
			}
		}
		return s.customRoleDelegation(ctx, actor, value.Project, value.ID)
	})
}

func (s *Service) RolePolicyDetails(ctx context.Context, actor, id string) (RolePolicyDetail, error) {
	row, err := s.ReadRolePolicy(ctx, actor, id)
	if err != nil {
		return RolePolicyDetail{}, err
	}
	value, err := storage.Decode[RolePolicy](row)
	if err != nil {
		return RolePolicyDetail{}, err
	}
	result := RolePolicyDetail{Record: row}
	active, err := s.roleProjectActive(ctx, actor, value.Project)
	if err != nil || !active {
		return result, err
	}
	result.CanManage, err = roleCapability(func() error {
		if err := s.rolePolicyAuthority(ctx, actor, value, true); err != nil {
			return err
		}
		if err := s.roleRulesAuthority(ctx, actor, value, false); err != nil {
			return err
		}
		return s.roleInheritanceAuthority(ctx, actor, value)
	})
	if err != nil {
		return result, err
	}
	result.Delegatable, err = s.roleDelegatable(ctx, actor, value)
	return result, err
}

func (s *Service) CustomRoleChoices(ctx context.Context, actor, project, cursor string, limit int) (RoleChoicePage, error) {
	page, err := s.listRolePolicies(ctx, actor, project, cursor, limit, "custom-role-choices")
	if err != nil {
		return RoleChoicePage{}, err
	}
	active, err := s.roleProjectActive(ctx, actor, project)
	if err != nil {
		return RoleChoicePage{}, err
	}
	result := RoleChoicePage{Items: []RoleView{}, Cursor: page.Cursor, Partial: page.Partial}
	for _, row := range page.Items {
		value, err := storage.Decode[RolePolicy](row)
		if err != nil {
			return result, err
		}
		delegate := false
		if active {
			delegate, err = s.roleDelegatable(ctx, actor, value)
			if err != nil {
				return result, err
			}
		}
		actions := []string{}
		for _, rule := range value.Rules {
			if !slices.Contains(actions, rule.Permission) {
				actions = append(actions, rule.Permission)
			}
		}
		slices.Sort(actions)
		scope := "project"
		if project == "" {
			scope = "installation"
		}
		result.Items = append(result.Items, RoleView{ID: value.ID, Name: value.Name, Scope: scope, Permissions: actions, Disabled: value.Disabled, Delegatable: delegate})
	}
	return result, nil
}
