package authorization

import (
	"context"
	"errors"
	"slices"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type scopedResource struct {
	Application    string   `json:"applicationId"`
	Applications   []string `json:"applications"`
	ApplicationIDs []string `json:"applicationIds"`
	Request        struct {
		Application string `json:"applicationId"`
		Environment string `json:"environmentId"`
		Suite       string `json:"suiteId"`
	} `json:"request"`
	Environment *struct {
		Application string `json:"applicationId"`
	} `json:"environment"`
	Bindings []struct {
		Suite string `json:"suiteId"`
	} `json:"bindings"`
	Configuration struct {
		Bindings []struct {
			Suite string `json:"suiteId"`
		} `json:"bindings"`
	} `json:"configuration"`
	Tests []struct {
		Application string `json:"applicationId"`
	} `json:"tests"`
	Release struct {
		Catalog struct {
			Tests []struct {
				Application string `json:"applicationId"`
			} `json:"tests"`
		} `json:"catalog"`
	} `json:"release"`
}

type applicationScopeKey struct{}

type ApplicationScope struct {
	Project     string
	Application string
}

func WithApplication(ctx context.Context, project, application string) context.Context {
	return context.WithValue(ctx, applicationScopeKey{}, ApplicationScope{Project: project, Application: application})
}

func ScopedApplication(ctx context.Context, project string) string {
	scope, _ := ctx.Value(applicationScopeKey{}).(ApplicationScope)
	if scope.Project == project {
		return scope.Application
	}
	return ""
}

type applicationStore struct{ permissions.PermissionStore }

func (s *applicationStore) RoleAssignmentsForPrincipal(ctx context.Context, principal permissions.PrincipalRef) ([]permissions.RoleAssignment, error) {
	rows, err := s.PermissionStore.RoleAssignmentsForPrincipal(ctx, principal)
	if err != nil {
		return nil, err
	}
	result := make([]permissions.RoleAssignment, 0, len(rows))
	for _, row := range rows {
		application, scoped := row.BindingValues["application"]
		if scoped {
			project, ok := row.BindingValues["team"].(string)
			if !ok || application == "" || application != ScopedApplication(ctx, project) || !StandardRole(row.RoleID) {
				continue
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func (s *Service) CheckApplication(ctx context.Context, user, project, application, object, name string) error {
	if !domain.ValidID(project) || !domain.ValidID(application) {
		return ErrForbidden
	}
	row, err := s.Directory.Store.Get(ctx, "applications/"+project, application)
	if err != nil {
		return err
	}
	if row.Project != project {
		return storage.ErrNotFound
	}
	return s.Check(WithApplication(ctx, project, application), user, project, object, name)
}

func (s *Service) CheckProjectObject(ctx context.Context, user, project, object, name string) error {
	profile, err := s.Directory.User(ctx, user)
	if err != nil {
		return err
	}
	if profile.Disabled {
		return ErrForbidden
	}
	engine, err := s.CurrentEngine(ctx)
	if err != nil {
		return err
	}
	ok, err := engine.HasPermission(WithApplication(ctx, project, ""), RequestFor(user, project, object, name))
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) CheckCollection(ctx context.Context, user, project, name string) error {
	if err := s.Check(ctx, user, project, "", name); !errors.Is(err, ErrForbidden) {
		return err
	}
	rows, err := s.Directory.Store.All(ctx, "applications/"+project, project)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.CheckApplication(ctx, user, project, row.ID, "", name); err == nil {
			return nil
		} else if !errors.Is(err, ErrForbidden) {
			return err
		}
	}
	return ErrForbidden
}

func (s *Service) CheckSuiteTestList(ctx context.Context, user, project, suite string) error {
	if err := s.Check(ctx, user, project, "suite/"+suite, "catalog.read"); !errors.Is(err, ErrForbidden) {
		return err
	}
	row, err := s.Directory.Store.Get(ctx, "suites/"+project, suite)
	if err != nil {
		return err
	}
	value, err := storage.Decode[scopedResource](row)
	if err != nil {
		return err
	}
	if row.Project != project {
		return storage.ErrNotFound
	}
	engine, err := s.CurrentEngine(ctx)
	if err != nil {
		return err
	}
	profile, err := s.Directory.User(ctx, user)
	if err != nil {
		return err
	}
	if profile.Disabled {
		return ErrForbidden
	}
	for _, app := range value.Applications {
		ok, err := engine.HasPermission(WithApplication(ctx, project, app), RequestFor(user, project, "suite/"+suite, "catalog.read"))
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return ErrForbidden
}

func (s *Service) ApplicationsForObject(ctx context.Context, project, object, permission string) ([]string, error) {
	parts := strings.Split(object, "/")
	if len(parts) < 2 {
		return nil, nil
	}
	kind, id := parts[0], parts[1]
	bucket := map[string]string{"application": "applications/", "environment": "environments/", "suite": "suites/", "run": "runs/", "schedule": "schedules/", "notification": "notification-channels/", "source": "repository-sources/"}[kind]
	if kind == "test" && len(parts) == 3 {
		bucket, id = "tests/"+project+"/"+id, parts[2]
	} else if bucket != "" {
		bucket += project
		if kind == "run" {
			bucket = "runs"
		}
	} else {
		return nil, nil
	}
	row, err := s.Directory.Store.Get(ctx, bucket, id)
	if errors.Is(err, storage.ErrNotFound) {
		if kind == "test" {
			return s.ApplicationsForObject(ctx, project, "suite/"+parts[1], permission)
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Project != project {
		return nil, storage.ErrNotFound
	}
	data, err := storage.Decode[scopedResource](row)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "application":
		return []string{id}, nil
	case "environment":
		if data.Application != "" {
			return []string{data.Application}, nil
		}
	case "test":
		if data.Application != "" {
			return []string{data.Application}, nil
		}
		if permission == "catalog.read" || permission == "teststudio.catalog.read" {
			row, err := s.Directory.Store.Get(ctx, "suites/"+project, parts[1])
			if err != nil {
				return nil, err
			}
			suite, err := storage.Decode[scopedResource](row)
			if err != nil {
				return nil, err
			}
			return suite.Applications, nil
		}
		return s.ApplicationsForObject(ctx, project, "suite/"+parts[1], permission)
	case "suite":
		if len(data.Applications) == 1 {
			return data.Applications, nil
		}
	case "run":
		app := data.Request.Application
		if data.Environment != nil {
			if app != "" && app != data.Environment.Application {
				return nil, nil
			}
			app = data.Environment.Application
		}
		if app == "" && len(data.ApplicationIDs) == 1 {
			app = data.ApplicationIDs[0]
		}
		if app == "" {
			return nil, nil
		}
		for _, test := range data.Tests {
			if test.Application != "" && test.Application != app {
				return nil, nil
			}
		}
		for _, test := range data.Release.Catalog.Tests {
			if test.Application != "" && test.Application != app {
				return nil, nil
			}
		}
		return []string{app}, nil
	case "schedule":
		app := data.Request.Application
		if data.Request.Environment != "" {
			apps, err := s.ApplicationsForObject(ctx, project, "environment/"+data.Request.Environment, permission)
			if err != nil || len(apps) != 1 {
				return nil, err
			}
			if app != "" && app != apps[0] {
				return nil, nil
			}
			app = apps[0]
		}
		if app != "" {
			return []string{app}, nil
		}
	case "source":
		var app string
		bindings := data.Configuration.Bindings
		for _, binding := range bindings {
			apps, err := s.ApplicationsForObject(ctx, project, "suite/"+binding.Suite, permission)
			if err != nil || len(apps) != 1 {
				return nil, err
			}
			if app != "" && app != apps[0] {
				return nil, nil
			}
			app = apps[0]
		}
		if app != "" {
			return []string{app}, nil
		}
	case "notification":
		if data.Application != "" {
			return []string{data.Application}, nil
		}
	}
	return nil, nil
}

func (s *Service) PermissionContexts(ctx context.Context, project, object, permission string) ([]context.Context, error) {
	if object == "" || strings.HasPrefix(object, "role/") || strings.HasPrefix(object, "user/") || strings.HasPrefix(object, "group/") {
		return []context.Context{ctx}, nil
	}
	apps, err := s.ApplicationsForObject(ctx, project, object, permission)
	if err != nil {
		return nil, err
	}
	app := ScopedApplication(ctx, project)
	if app != "" && slices.Contains(apps, app) {
		return []context.Context{WithApplication(ctx, project, app)}, nil
	}
	if app == "" && len(apps) > 0 {
		contexts := []context.Context{}
		for _, application := range apps {
			contexts = append(contexts, WithApplication(ctx, project, application))
		}
		return contexts, nil
	}
	return []context.Context{WithApplication(ctx, project, "")}, nil
}
