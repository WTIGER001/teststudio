package access

import (
	"context"
	"errors"
	"slices"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

type replacingStandardRoleKey struct{}

func (s *Service) ownerScopes(ctx context.Context) ([]authorization.ApplicationScope, error) {
	rows, err := s.Auth.Engine.RoleAssignmentsForRoleID(ctx, authorization.OwnerRole)
	if err != nil {
		return nil, err
	}
	result := []authorization.ApplicationScope{}
	for _, row := range rows {
		project, _ := row.BindingValues["team"].(string)
		app, _ := row.BindingValues["application"].(string)
		if project == "" {
			return nil, errors.New("owner assignment has invalid scope")
		}
		scope := authorization.ApplicationScope{Project: project, Application: app}
		if !slices.Contains(result, scope) {
			result = append(result, scope)
		}
	}
	return result, nil
}

func (s *Service) usableOwner(ctx context.Context, scope authorization.ApplicationScope, changes ...storage.Change) (bool, error) {
	engine, view, logins, err := s.administratorState(ctx)
	if err != nil {
		return false, err
	}
	for _, change := range changes {
		if err := view.apply(change.Record, change.Delete); err != nil {
			return false, err
		}
	}
	role, err := engine.RoleDefinition(ctx, authorization.OwnerRole)
	if err != nil {
		return false, err
	}
	if role.IsDisabled {
		return false, nil
	}
	owners, err := s.Auth.Engine.RoleAssignmentsForRoleID(ctx, authorization.OwnerRole)
	if err != nil {
		return false, err
	}
	ctx = authorization.WithApplication(ctx, scope.Project, scope.Application)
	for user, profile := range view.users {
		if profile.Disabled || !logins[user] && !view.local[user] {
			continue
		}
		groups, err := view.GetUserGroups(ctx, user)
		if err != nil {
			return false, err
		}
		candidate := false
		for _, owner := range owners {
			app, _ := owner.BindingValues["application"].(string)
			if owner.BindingValues["team"] != scope.Project || app != "" && app != scope.Application {
				continue
			}
			if owner.Kind == permissions.PrincipalUser && owner.ID == user || owner.Kind == permissions.PrincipalGroup && slices.Contains(groups, owner.ID) {
				candidate = true
				break
			}
		}
		if !candidate {
			continue
		}
		usable := true
		for _, action := range []string{"members.owners", "members.manage", "permissions.grant", "projects.manage"} {
			ok, err := engine.HasPermission(ctx, authorization.RequestFor(user, scope.Project, "", action))
			if err != nil {
				return false, err
			}
			if !ok {
				usable = false
				break
			}
		}
		if usable {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) ownerCheckpoint(ctx context.Context) ([]authorization.ApplicationScope, error) {
	scopes, err := s.ownerScopes(ctx)
	if err != nil {
		return nil, err
	}
	result := []authorization.ApplicationScope{}
	for _, scope := range scopes {
		ok, err := s.usableOwner(ctx, scope)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, scope)
		}
	}
	return result, nil
}

func (s *Service) preserveOwners(ctx context.Context, scopes []authorization.ApplicationScope, changes ...storage.Change) error {
	for _, scope := range scopes {
		ok, err := s.usableOwner(ctx, scope, changes...)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("the change would remove the last usable Owner for the scope")
		}
	}
	return nil
}

func (s *Service) protectOwnerPolicy(ctx context.Context, actor, project, id string) error {
	policies, err := s.ownedRolePolicies(ctx, project)
	if err != nil {
		return err
	}
	impacted := map[string]bool{id: true}
	for changed := true; changed; {
		changed = false
		for parent, policy := range policies {
			for _, child := range policy.Inherits {
				if impacted[child] && !impacted[parent] {
					impacted[parent] = true
					changed = true
				}
			}
		}
	}
	for role := range impacted {
		assignments, err := s.Auth.Engine.RoleAssignmentsForRoleID(ctx, role)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if err := s.protectOwnerPrincipal(ctx, actor, permissions.PrincipalRef{Kind: assignment.Kind, ID: assignment.ID}, project); err != nil {
				return err
			}
		}
	}
	return nil
}
