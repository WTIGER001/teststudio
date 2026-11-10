package catalog

import (
	"context"
	"fmt"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"slices"
)

func (s *Service) validateRequirements(ctx context.Context, project, application string, test Test) error {
	if len(test.RequiredParameters) == 0 && len(test.RequiredBindings) == 0 {
		return nil
	}
	row, err := s.Store.Get(ctx, projects.Bucket("applications", project), application)
	if err != nil {
		return err
	}
	app, err := storage.Decode[projects.Application](row)
	if err != nil {
		return err
	}
	for _, name := range test.RequiredParameters {
		if !slices.ContainsFunc(app.Requirements, func(r projects.Requirement) bool { return r.Kind == "parameter" && r.Name == name }) {
			return fmt.Errorf("declare parameter %s in application configuration before saving this test", name)
		}
	}
	for _, name := range test.RequiredBindings {
		if slices.ContainsFunc(app.Requirements, func(r projects.Requirement) bool { return r.Name == name && r.Kind != "secret" }) {
			return fmt.Errorf("configuration name %s is not declared as a secret", name)
		}
	}
	return nil
}
