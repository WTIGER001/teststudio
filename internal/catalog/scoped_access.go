package catalog

import (
	"context"
	"strings"
)

func (s *Service) listAuthority(ctx context.Context, user, project, object string) error {
	if object != "" {
		return s.Auth.CheckSuiteTestList(ctx, user, project, strings.TrimPrefix(object, "suite/"))
	}
	return s.Auth.CheckCollection(ctx, user, project, "catalog.read")
}
