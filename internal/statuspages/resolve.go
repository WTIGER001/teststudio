package statuspages

import (
	"context"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) resolve(ctx context.Context, user, project string, entry Entry) (reliability.Scope, catalog.Test, *projects.Environment, bool, error) {
	scope := reliability.Scope{Project: project, Suite: entry.Suite, Test: entry.Test, Environment: entry.Environment, Location: entry.Location}
	disabled := false
	if entry.Probe != "" {
		binding, _, err := s.Probes.Read(ctx, user, project, entry.Probe)
		if err != nil {
			return scope, catalog.Test{}, nil, false, err
		}
		scope.Suite = binding.Suite
		scope.Test = binding.Test
		scope.Environment = binding.Environment
		scope.Location = binding.Location
		disabled = binding.Archived
	}
	if !domain.ValidID(scope.Suite) || !domain.ValidID(scope.Test) || scope.Environment != "" && !domain.ValidID(scope.Environment) || len(scope.Location) > 128 {
		return scope, catalog.Test{}, nil, false, storage.ErrNotFound
	}
	if scope.Location == "" {
		scope.Location = s.Probes.Runs.Location
	}
	retained, err := s.Probes.Catalog.RetainedTest(ctx, user, project, scope.Suite, scope.Test)
	if err != nil {
		return scope, catalog.Test{}, nil, false, err
	}
	disabled = disabled || retained.SuiteArchived || !retained.Current || retained.Test.State != "enabled"
	desired := retained.Test
	if release, err := s.Probes.Catalog.ActiveRelease(ctx, project, scope.Suite); err == nil {
		for _, test := range release.Draft.Tests {
			if test.ID == scope.Test {
				desired = test
				break
			}
		}
	}
	var environment *projects.Environment
	if scope.Environment != "" {
		row, err := s.Store.Get(ctx, projects.Bucket("environments", project), scope.Environment)
		if err != nil {
			return scope, desired, nil, false, err
		}
		env, err := storage.Decode[projects.Environment](row)
		if err != nil {
			return scope, desired, nil, false, err
		}
		environment = &env
		disabled = disabled || env.Disabled
	}
	return scope, desired, environment, disabled, nil
}
