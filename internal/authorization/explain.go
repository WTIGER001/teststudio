package authorization

import (
	"context"
	"slices"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
)

type GrantPath struct {
	Principal  permissions.PrincipalRef
	Assignment *permissions.RoleAssignment
	Grant      permissions.Grant
}

type explanationStore struct {
	permissions.PermissionStore
	path     *GrantPath
	request  permissions.Request
	baseline bool
	err      error
}

func (s *explanationStore) RoleDefinition(ctx context.Context, id string) (permissions.Role, error) {
	role, err := s.PermissionStore.RoleDefinition(ctx, id)
	if err != nil {
		s.err = err
	}
	return role, err
}

func (s *explanationStore) RoleAssignmentsForPrincipal(ctx context.Context, principal permissions.PrincipalRef) ([]permissions.RoleAssignment, error) {
	if s.path == nil {
		return s.PermissionStore.RoleAssignmentsForPrincipal(ctx, principal)
	}
	if principal == s.path.Principal && s.path.Assignment != nil {
		return []permissions.RoleAssignment{*s.path.Assignment}, nil
	}
	return []permissions.RoleAssignment{}, nil
}

func (s *explanationStore) GrantsForOwners(ctx context.Context, owners []permissions.PrincipalRef, req permissions.Request) ([]permissions.Grant, error) {
	grants := []permissions.Grant{}
	if s.path != nil {
		grant := s.path.Grant
		if slices.Contains(owners, permissions.PrincipalRef{Kind: grant.OwnerKind, ID: grant.OwnerID}) {
			grants = append(grants, grant)
		}
	} else {
		values, err := s.PermissionStore.GrantsForOwners(ctx, owners, req)
		if err != nil {
			return nil, err
		}
		for _, grant := range values {
			if grant.Effect == permissions.EffectDeny {
				grants = append(grants, grant)
			}
		}
	}
	if s.baseline && slices.Contains(owners, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: s.request.UserID}) {
		grants = append(grants, permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: s.request.UserID, PermissionName: s.request.Perm, Effect: permissions.EffectAllow, TeamScope: "*"})
	}
	return grants, nil
}

type explanationIdentity struct {
	permissions.IdentityProvider
	user   string
	groups []string
}

func (s *explanationIdentity) GetUserGroups(ctx context.Context, user string) ([]string, error) {
	if user == s.user {
		return slices.Clone(s.groups), nil
	}
	return s.IdentityProvider.GetUserGroups(ctx, user)
}

func (s *Service) ExplainMatchingDeny(ctx context.Context, engine *permissions.Service, request permissions.Request, groups []string) (bool, error) {
	store := &explanationStore{PermissionStore: engine.GetStore(), request: request, baseline: true}
	identity := &explanationIdentity{IdentityProvider: s.Directory, user: request.UserID, groups: groups}
	probe, err := NewEngine(ctx, store, identity)
	if err != nil {
		return false, err
	}
	allowed, err := probe.HasPermission(ctx, request)
	if store.err != nil {
		return false, store.err
	}
	return !allowed, err
}

func (s *Service) ExplainGrantPath(ctx context.Context, engine *permissions.Service, request permissions.Request, groups []string, path GrantPath) (bool, error) {
	store := &explanationStore{PermissionStore: engine.GetStore(), path: &path, request: request, baseline: path.Grant.Effect == permissions.EffectDeny}
	identity := &explanationIdentity{IdentityProvider: s.Directory, user: request.UserID, groups: groups}
	probe, err := roleEngine(ctx, store, identity)
	if err != nil {
		return false, err
	}
	if path.Grant.OwnerKind == permissions.PrincipalRole && strings.HasPrefix(path.Grant.OwnerID, "builtin.") {
		if err := probe.CreateGrant(ctx, path.Grant); err != nil {
			return false, err
		}
	}
	allowed, err := probe.HasPermission(ctx, request)
	if store.err != nil {
		return false, store.err
	}
	if path.Grant.Effect == permissions.EffectDeny {
		return !allowed, err
	}
	return allowed, err
}
