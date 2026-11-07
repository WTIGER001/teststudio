package access

import (
	"context"
	"errors"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func (s *Service) hasStandardAssignment(ctx context.Context, user, project string) bool {
	principals := []permissions.PrincipalRef{{Kind: permissions.PrincipalUser, ID: user}}
	groups, err := s.Auth.Directory.GetUserGroups(ctx, user)
	if err != nil {
		return true
	}
	for _, group := range groups {
		principals = append(principals, permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: group})
	}
	for _, principal := range principals {
		rows, err := s.Auth.Engine.RoleAssignmentsForPrincipal(ctx, principal)
		if err != nil {
			return true
		}
		for _, row := range rows {
			if authorization.StandardRole(row.RoleID) && row.BindingValues["team"] == project {
				return true
			}
		}
	}
	return false
}

func (s *Service) protectOwnerPrincipal(ctx context.Context, actor string, principal permissions.PrincipalRef, projects ...string) error {
	owners, err := s.Auth.Engine.RoleAssignmentsForRoleID(ctx, authorization.OwnerRole)
	if err != nil {
		return err
	}
	groups := []string{}
	if principal.Kind == permissions.PrincipalUser {
		groups, err = s.Auth.Directory.GetUserGroups(ctx, principal.ID)
		if err != nil {
			return err
		}
	}
	for _, owner := range owners {
		if len(projects) > 0 && projects[0] != "" && owner.BindingValues["team"] != projects[0] {
			continue
		}
		matches := owner.Kind == principal.Kind && owner.ID == principal.ID
		for _, group := range groups {
			matches = matches || owner.Kind == permissions.PrincipalGroup && owner.ID == group
		}
		if !matches {
			continue
		}
		project, ok := owner.BindingValues["team"].(string)
		if !ok || project == "" {
			return errors.New("owner assignment has invalid scope")
		}
		app, _ := owner.BindingValues["application"].(string)
		if err := s.Auth.Check(authorization.WithApplication(ctx, project, app), actor, project, "", "members.owners"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) protectGroupDelegation(ctx context.Context, actor, group string) error {
	rows, err := s.Auth.Engine.RoleAssignmentsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: group})
	if err != nil {
		return err
	}
	for _, row := range rows {
		project, _ := row.BindingValues["team"].(string)
		application, _ := row.BindingValues["application"].(string)
		if !s.hasStandardAssignment(ctx, actor, project) {
			continue
		}
		definition := AssignmentDefinition{Application: application, PrincipalKind: permissions.PrincipalGroup, PrincipalID: group, RoleID: row.RoleID}
		if err := s.assignmentAuthority(authorization.WithApplication(ctx, project, application), actor, project, definition, true); err != nil {
			return err
		}
	}
	return nil
}
