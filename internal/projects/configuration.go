package projects

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"net/url"
	"slices"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
)

type EnvironmentConfiguration struct {
	storage.Record
	Readiness Readiness `json:"readiness"`
}

type ApplicationConfiguration struct {
	Application  storage.Record             `json:"application"`
	Requirements []Requirement              `json:"requirements"`
	Environments []EnvironmentConfiguration `json:"environments"`
}

func (s *Service) Configuration(ctx context.Context, user, project, application string) (ApplicationConfiguration, error) {
	result := ApplicationConfiguration{Requirements: []Requirement{}, Environments: []EnvironmentConfiguration{}}
	if err := s.Auth.CheckApplication(ctx, user, project, application, "application/"+application, "projects.read"); err != nil {
		return result, err
	}
	row, err := s.Store.Get(ctx, Bucket("applications", project), application)
	if err != nil {
		return result, err
	}
	app, err := storage.Decode[Application](row)
	if err != nil {
		return result, err
	}
	result.Application = row
	result.Requirements = append(result.Requirements, app.Requirements...)
	seen := map[string]bool{}
	for _, r := range result.Requirements {
		seen[r.Kind+"/"+r.Name] = true
	}
	add := func(name, kind string) {
		if !seen[kind+"/"+name] {
			result.Requirements = append(result.Requirements, Requirement{Name: name, Kind: kind})
			seen[kind+"/"+name] = true
		}
	}
	suites, err := s.Store.All(ctx, Bucket("suites", project), project)
	if err != nil {
		return result, err
	}
	for _, row := range suites {
		suite, err := storage.Decode[Suite](row)
		if err != nil {
			return result, err
		}
		if suite.Archived || len(suite.Applications) != 1 || suite.Applications[0] != application {
			continue
		}
		if err := s.Auth.Check(ctx, user, project, "suite/"+suite.ID, "catalog.read"); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		tests, err := s.Store.All(ctx, "tests/"+project+"/"+suite.ID, project)
		if err != nil {
			return result, err
		}
		readableTest := false
		for _, row := range tests {
			if err := s.Auth.Check(ctx, user, project, "test/"+suite.ID+"/"+row.ID, "catalog.read"); errors.Is(err, authorization.ErrForbidden) {
				continue
			} else if err != nil {
				return result, err
			}
			var test struct {
				State      string   `json:"state"`
				Parameters []string `json:"requiredParameters"`
				Bindings   []string `json:"requiredBindings"`
				Check      *struct {
					Endpoint string `json:"endpoint"`
				} `json:"check"`
			}
			if err := json.Unmarshal(row.Data, &test); err != nil {
				return result, err
			}
			if test.State != "enabled" {
				continue
			}
			readableTest = true
			for _, name := range test.Parameters {
				add(name, "parameter")
			}
			for _, name := range test.Bindings {
				add(name, "secret")
			}
			if test.Check != nil {
				add(test.Check.Endpoint, "endpoint")
			}
		}
		if readableTest && suite.ActiveRelease != "" {
			release, err := s.Store.Get(ctx, "releases/"+project+"/"+suite.ID, suite.ActiveRelease)
			if err != nil {
				return result, err
			}
			var value struct {
				Catalog struct {
					Artifact struct {
						Browser *struct {
							Endpoint string `json:"endpoint"`
						} `json:"browser"`
					} `json:"artifact"`
				} `json:"catalog"`
			}
			if err := json.Unmarshal(release.Data, &value); err != nil {
				return result, err
			}
			if value.Catalog.Artifact.Browser != nil {
				add(value.Catalog.Artifact.Browser.Endpoint, "endpoint")
			}
		}
	}
	rows, err := s.Store.All(ctx, Bucket("environments", project), project)
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		environment, err := storage.Decode[Environment](row)
		if err != nil {
			return result, err
		}
		if environment.Application != application {
			continue
		}
		if err := s.Auth.CheckApplication(ctx, user, project, application, "environment/"+row.ID, "environments.read"); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		result.Environments = append(result.Environments, EnvironmentConfiguration{Record: row, Readiness: CheckConfiguration(ctx, s.Secrets, project, app, environment, result.Requirements, false)})
	}
	return result, nil
}

func (s *Service) SaveRequirements(ctx context.Context, user, project, application string, requirements []Requirement, revision int64) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		if err := auth.CheckApplication(ctx, user, project, application, "application/"+application, "applications.manage"); err != nil {
			return err
		}
		row, err := auth.Directory.Store.Get(ctx, Bucket("applications", project), application)
		if err != nil {
			return err
		}
		value, err := storage.Decode[Application](row)
		if err != nil {
			return err
		}
		if err := ValidateRequirements(requirements); err != nil {
			return err
		}
		value.Requirements = requirements
		return (&Service{Store: auth.Directory.Store, Auth: auth}).save(ctx, user, Bucket("applications", project), application, project, revision, value)
	})
}

func (s *Service) ApplicationCredentialReferences(ctx context.Context, user, project, application string) ([]secrets.Reference, error) {
	if err := s.Auth.CheckApplication(ctx, user, project, application, "application/"+application, "environments.manage"); err != nil {
		return nil, err
	}
	if s.Secrets == nil {
		return []secrets.Reference{}, nil
	}
	references, err := s.Secrets.References(ctx, project, secrets.APICredential)
	if err != nil {
		return nil, errors.New("API credential references are unavailable")
	}
	if err := s.Auth.Check(ctx, user, project, "", "environments.manage"); err == nil {
		return references, nil
	} else if !errors.Is(err, authorization.ErrForbidden) {
		return nil, err
	}
	environments, err := s.Store.All(ctx, Bucket("environments", project), project)
	if err != nil {
		return nil, err
	}
	origins := map[string]bool{}
	for _, row := range environments {
		e, err := storage.Decode[Environment](row)
		if err != nil {
			return nil, err
		}
		if e.Application != application {
			continue
		}
		if err := s.Auth.CheckApplication(ctx, user, project, application, "environment/"+e.ID, "environments.manage"); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return nil, err
		}
		for _, address := range e.Endpoints {
			u, err := url.Parse(address)
			if err != nil {
				continue
			}
			origin, err := browserpolicy.Origin(u.Scheme + "://" + u.Host)
			if err == nil {
				origins[origin] = true
			}
		}
	}
	result := []secrets.Reference{}
	for _, reference := range references {
		if slices.ContainsFunc(reference.Origins, func(origin string) bool { return origins[origin] }) {
			result = append(result, reference)
		}
	}
	return result, nil
}
