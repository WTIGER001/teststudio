package projects

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) EnsureDefaultSuite(ctx context.Context, user, project, application string, revision int64) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		if err := auth.CheckApplication(ctx, user, project, application, "application/"+application, "applications.manage"); err != nil {
			return err
		}
		if err := auth.CheckApplication(ctx, user, project, application, "", "suites.manage"); err != nil {
			return err
		}
		row, err := auth.Directory.Store.Get(ctx, Bucket("applications", project), application)
		if err != nil {
			return err
		}
		if row.Revision != revision {
			return storage.ErrConflict
		}
		app, err := storage.Decode[Application](row)
		if err != nil {
			return err
		}
		if app.DefaultSuite != "" {
			return nil
		}
		app.DefaultSuite = DefaultSuiteID(application)
		if _, err := auth.Directory.Store.Get(ctx, Bucket("suites", project), app.DefaultSuite); err == nil {
			return errors.New("the default suite identifier is already used; review suite settings before continuing")
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.saveSuite(ctx, user, project, Suite{ID: app.DefaultSuite, Name: "Default", Applications: []string{application}}, 0); err != nil {
			return err
		}
		return bound.save(ctx, user, Bucket("applications", project), application, project, revision, app)
	})
}
