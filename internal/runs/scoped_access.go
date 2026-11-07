package runs

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) AuthorizeRequest(ctx context.Context, user, project string, req Request, permission string) (context.Context, error) {
	err := s.Auth.Check(ctx, user, project, "", permission)
	if err == nil {
		return ctx, nil
	}
	if !errors.Is(err, authorization.ErrForbidden) {
		return ctx, err
	}
	app := req.Application
	if req.Environment != "" {
		row, err := s.Store.Get(ctx, projects.Bucket("environments", project), req.Environment)
		if err != nil {
			return ctx, err
		}
		env, err := storage.Decode[projects.Environment](row)
		if err != nil {
			return ctx, err
		}
		if row.Project != project || app != "" && app != env.Application {
			return ctx, authorization.ErrForbidden
		}
		app = env.Application
	}
	if app == "" {
		apps, err := s.Auth.ApplicationsForObject(ctx, project, "suite/"+req.Suite, permission)
		if err != nil {
			return ctx, err
		}
		if len(apps) == 1 {
			app = apps[0]
		}
	}
	if app == "" {
		return ctx, authorization.ErrForbidden
	}
	if err := s.Auth.CheckApplication(ctx, user, project, app, "", permission); err != nil {
		return ctx, err
	}
	return authorization.WithApplication(ctx, project, app), nil
}

func (s *Service) validateApplicationRun(ctx context.Context, user string, run Run) error {
	app := authorization.ScopedApplication(ctx, run.Project)
	if app == "" {
		return nil
	}
	if len(run.Applications) == 0 {
		return authorization.ErrForbidden
	}
	for _, test := range run.Release.Draft.Tests {
		if test.Application != "" && test.Application != app {
			return authorization.ErrForbidden
		}
	}
	for _, test := range run.Tests {
		if test.Application != "" && test.Application != app {
			return authorization.ErrForbidden
		}
	}
	if run.Environment != nil && run.Environment.Application != app {
		return authorization.ErrForbidden
	}
	for _, test := range run.Tests {
		if err := s.Auth.Check(ctx, user, run.Project, "test/"+run.Request.Suite+"/"+test.ID, "catalog.read"); err != nil {
			return err
		}
	}
	return nil
}
