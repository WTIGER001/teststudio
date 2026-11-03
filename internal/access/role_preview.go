package access

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
)

type RolePreviewInput struct {
	Policy   RolePolicyMutation `json:"policy"`
	Revision int64              `json:"revision"`
	User     string             `json:"userId"`
	Object   string             `json:"object"`
}

type PermissionComparison struct {
	Permission string `json:"permission"`
	Before     bool   `json:"before"`
	After      bool   `json:"after"`
}

type RolePreview struct {
	User         string                 `json:"userId"`
	Project      string                 `json:"projectId"`
	Object       string                 `json:"object"`
	UserDisabled bool                   `json:"userDisabled"`
	Decisions    []PermissionComparison `json:"decisions"`
}

func (s *Service) PreviewRolePolicy(ctx context.Context, actor string, input RolePreviewInput) (RolePreview, error) {
	if !domain.ValidID(input.User) || input.Revision < 0 {
		return RolePreview{}, errors.New("a registered user and nonnegative policy revision are required")
	}
	result := RolePreview{User: input.User, Project: input.Policy.Project, Object: input.Object, Decisions: []PermissionComparison{}}
	rollback := errors.New("discard temporary policy preview")
	input.Policy.Reason = "Preview proposed role policy"
	_, err := s.changeRolePolicy(ctx, actor, input.Policy, input.Revision, func(bound *Service, after bool) error {
		if !after {
			if err := bound.checkPolicyObject(ctx, result.Project, result.Object); err != nil {
				return err
			}
			for _, permission := range []string{"members.manage", "permissions.grant"} {
				if err := bound.Auth.Check(ctx, actor, result.Project, "user/"+input.User, permission); err != nil {
					return err
				}
			}
			if err := bound.Auth.Check(ctx, actor, result.Project, result.Object, "permissions.read"); err != nil {
				return err
			}
			user, err := bound.Auth.Directory.User(ctx, input.User)
			if err != nil {
				return err
			}
			result.UserDisabled = user.Disabled
			for _, permission := range authorization.PermissionNames() {
				if result.Project != "" && authorization.InstallationPermission(permission) {
					continue
				}
				allowed, err := previewDecision(ctx, bound, input.User, result.Project, result.Object, permission)
				if err != nil {
					return err
				}
				result.Decisions = append(result.Decisions, PermissionComparison{Permission: permission, Before: allowed})
			}
			return nil
		}
		if !input.Policy.Disabled && !result.UserDisabled {
			key := assignmentKey(result.Project, AssignmentDefinition{PrincipalKind: "user", PrincipalID: input.User, RoleID: input.Policy.ID})
			if err := bound.Auth.Engine.AssignRole(ctx, key.principal(), key.RoleID, key.Bindings); err != nil {
				return err
			}
		}
		for i, decision := range result.Decisions {
			allowed, err := previewDecision(ctx, bound, input.User, result.Project, result.Object, decision.Permission)
			if err != nil {
				return err
			}
			result.Decisions[i].After = allowed
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		return RolePreview{}, err
	}
	return result, nil
}

func previewDecision(ctx context.Context, s *Service, user, project, object, permission string) (bool, error) {
	err := s.Auth.Check(ctx, user, project, object, permission)
	if errors.Is(err, authorization.ErrForbidden) {
		return false, nil
	}
	return err == nil, err
}
