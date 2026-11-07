package sources

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) repositoryMutationAuthority(ctx context.Context, user, project, id string, input RepositoryConfig, revision int64) error {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return errors.New("valid project and source identifiers are required")
	}
	if err := s.Auth.CheckProjectObject(ctx, user, project, "source/"+id, "sources.manage"); err == nil {
		return nil
	} else if !errors.Is(err, authorization.ErrForbidden) {
		return err
	}
	if len(input.Bindings) == 0 {
		return authorization.ErrForbidden
	}
	app := ""
	for _, binding := range input.Bindings {
		apps, err := s.Auth.ApplicationsForObject(ctx, project, "suite/"+binding.Suite, "sources.manage")
		if err != nil {
			return err
		}
		if len(apps) != 1 || app != "" && app != apps[0] {
			return authorization.ErrForbidden
		}
		app = apps[0]
	}
	if err := s.Auth.CheckApplication(ctx, user, project, app, "", "sources.manage"); err != nil {
		return err
	}
	if revision == 0 {
		if _, err := s.Store.Get(ctx, repositoryBucket(project), id); !errors.Is(err, storage.ErrNotFound) {
			if err == nil {
				return storage.ErrConflict
			}
			return err
		}
		return nil
	}
	apps, err := s.Auth.ApplicationsForObject(ctx, project, "source/"+id, "sources.manage")
	if err != nil {
		return err
	}
	if len(apps) != 1 || apps[0] != app {
		return authorization.ErrForbidden
	}
	return s.checkRepository(ctx, user, project, id, "manage")
}
