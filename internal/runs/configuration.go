package runs

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) checkConfiguration(ctx context.Context, project string, environment projects.Environment, tests []catalog.Test, artifact catalog.Artifact) (projects.Environment, storage.Record, error) {
	row, err := s.Store.Get(ctx, projects.Bucket("applications", project), environment.Application)
	if err != nil {
		return environment, row, err
	}
	application, err := storage.Decode[projects.Application](row)
	if err != nil {
		return environment, row, err
	}
	required := []projects.Requirement{}
	for _, r := range application.Requirements {
		if r.Required {
			required = append(required, r)
		}
	}
	for _, test := range tests {
		for _, name := range test.RequiredParameters {
			index := slices.IndexFunc(application.Requirements, func(r projects.Requirement) bool { return r.Name == name && r.Kind == "parameter" })
			if index < 0 {
				return environment, row, fmt.Errorf("declare parameter %s in application configuration before running this test", name)
			}
			required = append(required, application.Requirements[index])
		}
		for _, name := range test.RequiredBindings {
			required = append(required, projects.Requirement{Name: name, Kind: "secret"})
		}
		if test.Check != nil {
			required = append(required, projects.Requirement{Name: test.Check.Endpoint, Kind: "endpoint"})
		}
		if artifact.Browser != nil {
			required = append(required, projects.Requirement{Name: artifact.Browser.Endpoint, Kind: "endpoint"})
		}
	}
	readiness := projects.CheckConfiguration(ctx, s.Secrets, project, application, environment, required, true)
	if err := readiness.Error(environment.Name); err != nil {
		return environment, row, err
	}
	environment.Parameters = projects.EffectiveParameters(application, environment)
	return environment, row, nil
}

func ownedSuite(suite projects.Suite) error {
	if len(suite.Applications) != 1 {
		return errors.New("assign this suite to one application before running tests")
	}
	return nil
}
