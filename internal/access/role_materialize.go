package access

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	permissions "github.com/wtiger001/go-permissions"
)

func roleGrant(value RolePolicy, rule RoleRule) permissions.Grant {
	scope := value.Project
	if scope == "" {
		scope = "*"
	}
	grant := permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: value.ID, TeamScope: scope, PermissionName: rule.Permission, Effect: rule.Effect, ExpiresAt: rule.ExpiresAt}
	if rule.Object != "" {
		object := rule.Object
		if value.Project != "" {
			object = value.Project + "/" + object
		}
		grant.ObjectScope = &object
	}
	return grant
}

func roleGrantKey(grant permissions.Grant) string {
	grant.ID = 0
	if len(grant.VariableSpec) == 0 {
		grant.VariableSpec = nil
	}
	if len(grant.RestrictedFields) == 0 {
		grant.RestrictedFields = nil
	}
	if grant.ExpiresAt != nil {
		expiry := grant.ExpiresAt.UTC()
		grant.ExpiresAt = &expiry
	}
	raw, _ := json.Marshal(grant)
	return string(raw)
}

func (s *Service) materializeRoleGrants(ctx context.Context, value RolePolicy) error {
	wanted := map[string]permissions.Grant{}
	if !value.Disabled {
		for _, rule := range value.Rules {
			g := roleGrant(value, rule)
			wanted[roleGrantKey(g)] = g
		}
	}
	page, err := s.Auth.Engine.GetStore().ListGrants(ctx, permissions.GrantQuery{Principals: []permissions.PrincipalRef{{Kind: permissions.PrincipalRole, ID: value.ID}}, Limit: 1000})
	if err != nil {
		return err
	}
	if page.NextCursor != "" {
		return errors.New("managed role grant storage exceeds its limit")
	}
	kept := map[string]bool{}
	for _, grant := range page.Grants {
		key := roleGrantKey(grant)
		if _, exists := wanted[key]; exists && !kept[key] {
			kept[key] = true
			continue
		}
		if err := s.Auth.RemovePolicyGrant(ctx, grant.ID); err != nil {
			return err
		}
	}
	keys := []string{}
	for key := range wanted {
		if !kept[key] {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	grants := []permissions.Grant{}
	for _, key := range keys {
		grants = append(grants, wanted[key])
	}
	return s.Auth.Engine.CreateGrants(ctx, grants)
}

func (s *Service) materializeRoleInheritance(ctx context.Context, before, after RolePolicy) error {
	old, next := before.Inherits, after.Inherits
	if before.Disabled {
		old = nil
	}
	if after.Disabled {
		next = nil
	}
	for _, child := range old {
		if !slices.Contains(next, child) {
			if err := s.Auth.RemoveRoleInheritance(ctx, after.ID, child); err != nil {
				return err
			}
		}
	}
	for _, child := range next {
		if !slices.Contains(old, child) {
			if err := s.Auth.Engine.AddRoleInheritance(ctx, after.ID, child); err != nil {
				return err
			}
		}
	}
	return nil
}
