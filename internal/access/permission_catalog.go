package access

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/authorization"
)

type PermissionChoice struct {
	authorization.PermissionDefinition
	CanGrant bool `json:"canGrant"`
}

func (s *Service) PermissionCatalog(ctx context.Context, actor, project, object string) ([]PermissionChoice, error) {
	if _, err := s.assignmentScope(ctx, actor, project, "roles.read"); err != nil {
		return nil, err
	}
	if err := s.checkPolicyObject(ctx, project, object); err != nil {
		return nil, err
	}
	result := []PermissionChoice{}
	for _, definition := range authorization.PermissionDefinitions() {
		if project != "" && definition.InstallationOnly {
			continue
		}
		canGrant := true
		for _, action := range []string{"permissions.grant", definition.ID} {
			if err := s.Auth.Check(ctx, actor, project, object, action); err != nil {
				if !errors.Is(err, authorization.ErrForbidden) {
					return nil, err
				}
				canGrant = false
			}
		}
		result = append(result, PermissionChoice{PermissionDefinition: definition, CanGrant: canGrant})
	}
	return result, nil
}
