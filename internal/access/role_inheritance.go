package access

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ownedRolePolicies(ctx context.Context, project string) (map[string]RolePolicy, error) {
	page, err := s.Store.List(ctx, rolePolicyBucket(project), project, "", 256)
	if err != nil {
		return nil, err
	}
	if page.Cursor != "" {
		return nil, errors.New("a scope permits at most 256 managed custom roles")
	}
	result := map[string]RolePolicy{}
	for _, row := range page.Items {
		value, err := storage.Decode[RolePolicy](row)
		if err != nil {
			return nil, err
		}
		if value.Project != project || row.Project != project || value.ID != row.ID {
			return nil, errors.New("role metadata does not match its stored scope")
		}
		result[value.ID] = value
	}
	return result, nil
}

func (s *Service) roleInheritanceAuthority(ctx context.Context, actor string, value RolePolicy) error {
	policies, err := s.ownedRolePolicies(ctx, value.Project)
	if err != nil {
		return err
	}
	if _, exists := policies[value.ID]; !exists && len(policies) >= 256 {
		return errors.New("a scope permits at most 256 managed custom roles")
	}
	policies[value.ID] = value
	if err := validatePolicyGraph(policies); err != nil {
		return err
	}
	visited := map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		policy, exists := policies[id]
		if !exists {
			return storage.ErrNotFound
		}
		if id != value.ID {
			if err := s.rolePolicyAuthority(ctx, actor, policy, false); err != nil {
				return err
			}
			if err := s.Auth.Check(ctx, actor, value.Project, "role/"+id, "permissions.grant"); err != nil {
				return err
			}
			if err := s.roleRulesAuthority(ctx, actor, policy, false); err != nil {
				return err
			}
		}
		visited[id] = true
		for _, child := range policy.Inherits {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(value.ID); err != nil {
		return err
	}
	impacted := map[string]bool{value.ID: true}
	for changed := true; changed; {
		changed = false
		for id, policy := range policies {
			if impacted[id] {
				continue
			}
			for _, child := range policy.Inherits {
				if impacted[child] {
					impacted[id] = true
					changed = true
					break
				}
			}
		}
	}
	for id := range impacted {
		if err := s.rolePolicyAuthority(ctx, actor, policies[id], true); err != nil {
			return err
		}
	}
	return nil
}

func validatePolicyGraph(policies map[string]RolePolicy) error {
	path, heights := map[string]bool{}, map[string]int{}
	var height func(string) (int, error)
	height = func(id string) (int, error) {
		if path[id] {
			return 0, errors.New("role inheritance cycle")
		}
		if n, exists := heights[id]; exists {
			return n, nil
		}
		value, exists := policies[id]
		if !exists {
			return 0, errors.New("inherited roles must exist in the same ownership scope")
		}
		path[id] = true
		n := 0
		for _, child := range value.Inherits {
			h, err := height(child)
			if err != nil {
				return 0, err
			}
			if h+1 > n {
				n = h + 1
			}
			if n > 16 {
				return 0, errors.New("role inheritance permits at most 16 levels")
			}
		}
		delete(path, id)
		heights[id] = n
		return n, nil
	}
	for id := range policies {
		if _, err := height(id); err != nil {
			return err
		}
	}
	return nil
}
