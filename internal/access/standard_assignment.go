package access

import (
	"context"
	"reflect"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

type assignmentAuthorizationKey struct{}

func (s *Service) authorizeAssignmentBatch(ctx context.Context, actor, project string, definitions []AssignmentDefinition) (context.Context, error) {
	allowed := map[string]bool{}
	for _, d := range definitions {
		if _, err := s.assignmentScope(ctx, actor, project, "members.manage"); err != nil {
			return ctx, err
		}
		if err := s.assignmentAuthority(ctx, actor, project, d, true); err != nil {
			return ctx, err
		}
		if d.PrincipalKind == permissions.PrincipalGroup {
			if _, _, err := s.group(ctx, actor, d.PrincipalID, "groups.manage"); err != nil {
				return ctx, err
			}
		}
		allowed[assignmentKey(project, d).ID()] = true
	}
	return context.WithValue(ctx, assignmentAuthorizationKey{}, allowed), nil
}

func (s *Service) ReplaceStandardAssignment(ctx context.Context, actor, project string, d AssignmentDefinition, revision int64) (AssignmentRecord, error) {
	if !authorization.StandardRole(d.RoleID) {
		return AssignmentRecord{}, authorization.ErrForbidden
	}
	if err := reason(d.Reason); err != nil {
		return AssignmentRecord{}, err
	}
	d.Application = authorization.ScopedApplication(ctx, project)
	var result AssignmentRecord
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := *s
		bound.Auth, bound.Store = auth, auth.Directory.Store
		scopes, err := bound.ownerCheckpoint(ctx)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, replacingStandardRoleKey{}, true)
		if err := bound.assignmentAuthority(ctx, actor, project, d, true); err != nil {
			return err
		}
		rows, err := auth.Engine.RoleAssignmentsForPrincipal(ctx, permissions.PrincipalRef{Kind: d.PrincipalKind, ID: d.PrincipalID})
		if err != nil {
			return err
		}
		var prior *AssignmentDefinition
		for _, row := range rows {
			if authorization.StandardRole(row.RoleID) && reflect.DeepEqual(row.BindingValues, assignmentKey(project, d).Bindings) {
				if prior != nil {
					return storage.ErrConflict
				}
				old := d
				old.RoleID = row.RoleID
				prior = &old
			}
		}
		if prior == nil {
			if revision != 0 || d.PreviousRole != "" {
				return storage.ErrConflict
			}
		} else {
			if d.PreviousRole != prior.RoleID {
				return storage.ErrConflict
			}
			row, _, err := bound.assignmentRecord(ctx, project, assignmentKey(project, *prior))
			if err != nil {
				return err
			}
			if row.Revision != revision {
				return storage.ErrConflict
			}
			if prior.RoleID == d.RoleID {
				result, err = bound.assignmentCapability(ctx, actor, row)
				return err
			}
			ctx, err = bound.authorizeAssignmentBatch(ctx, actor, project, []AssignmentDefinition{d, *prior})
			if err != nil {
				return err
			}
			if _, err := bound.applyAssignment(ctx, actor, project, *prior, revision, true); err != nil {
				return err
			}
		}
		result, err = bound.applyAssignment(ctx, actor, project, d, 0, false)
		if err != nil {
			return err
		}
		return bound.preserveOwners(ctx, scopes)
	})
	return result, err
}
