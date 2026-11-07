package access

import (
	"context"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
)

type MemberCandidate struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

func (s *Service) MemberCandidate(ctx context.Context, actor, project, user string) (MemberCandidate, error) {
	if !domain.ValidID(user) {
		return MemberCandidate{}, authorization.ErrForbidden
	}
	for _, action := range []string{"members.read", "members.manage", "permissions.grant"} {
		if _, err := s.assignmentScope(ctx, actor, project, action); err != nil {
			return MemberCandidate{}, err
		}
		if err := s.Auth.Check(ctx, actor, project, "user/"+user, action); err != nil {
			return MemberCandidate{}, err
		}
	}
	profile, err := s.Auth.Directory.User(ctx, user)
	if err != nil {
		return MemberCandidate{}, err
	}
	return MemberCandidate{ID: profile.ID, Name: profile.Name, Disabled: profile.Disabled}, nil
}

type OwnershipStatus struct {
	Project     string `json:"projectId"`
	Application string `json:"applicationId,omitempty"`
	State       string `json:"state"`
}

func (s *Service) Ownership(ctx context.Context, actor, project string) (OwnershipStatus, error) {
	result := OwnershipStatus{Project: project, Application: authorization.ScopedApplication(ctx, project), State: "system-managed"}
	if _, err := s.assignmentScope(ctx, actor, project, "members.read"); err != nil {
		return result, err
	}
	scopes, err := s.ownerScopes(ctx)
	if err != nil {
		return result, err
	}
	for _, scope := range scopes {
		if scope.Project != project || scope.Application != "" && scope.Application != result.Application {
			continue
		}
		result.State = "owner-unavailable"
		if ok, err := s.usableOwner(ctx, authorization.ApplicationScope{Project: project, Application: result.Application}); err != nil {
			return result, err
		} else if ok {
			result.State = "owned"
			return result, nil
		}
	}
	return result, nil
}
