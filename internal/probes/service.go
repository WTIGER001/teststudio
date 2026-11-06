package probes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store     *storage.Store
	Auth      *authorization.Service
	Catalog   *catalog.Service
	Runs      *runs.Service
	Schedules *schedules.Service
}

func Bucket(project string) string { return "probes/" + project }
func identifier(v Binding) string {
	raw, _ := json.Marshal([]string{v.Suite, v.Test, v.Environment, v.Endpoint, v.Location})
	sum := sha256.Sum256(raw)
	return "probe-" + hex.EncodeToString(sum[:16])
}
func valid(project string, v Binding) bool {
	return domain.ValidID(project) && domain.ValidID(v.Suite) && domain.ValidID(v.Test) && domain.ValidID(v.Environment) && domain.ValidID(v.Endpoint) && domain.ValidID(v.Location)
}
func (s *Service) authority(ctx context.Context, user, project string, v Binding, mutation bool) error {
	permission := "catalog.read"
	if mutation {
		permission = "catalog.update"
	}
	for _, obj := range []string{"suite/" + v.Suite, "test/" + v.Suite + "/" + v.Test} {
		if err := s.Auth.Check(ctx, user, project, obj, permission); err != nil {
			return err
		}
	}
	return s.Auth.Check(ctx, user, project, "environment/"+v.Environment, "environments.read")
}
func (s *Service) Save(ctx context.Context, user, project string, v Binding, revision int64) (storage.Record, error) {
	if v.Location == "" {
		v.Location = s.Runs.Location
	}
	if !valid(project, v) || strings.TrimSpace(v.Name) == "" || len(v.Name) > 256 {
		return storage.Record{}, errors.New("valid probe references and a name within 256 bytes are required")
	}
	id := identifier(v)
	if v.ID != "" && v.ID != id {
		return storage.Record{}, errors.New("probe target references are permanent; add another binding")
	}
	v.ID = id
	if err := s.authority(ctx, user, project, v, true); err != nil {
		return storage.Record{}, err
	}
	before, err := s.Store.Get(ctx, Bucket(project), id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Record{}, err
	}
	if revision > 0 && err != nil {
		return storage.Record{}, err
	}
	if revision == 0 && err == nil {
		old, _ := storage.Decode[Binding](before)
		if old == v {
			return before, nil
		}
		return storage.Record{}, storage.ErrConflict
	}
	changes := []storage.Change{}
	if !v.Archived {
		retained, err := s.Catalog.RetainedTest(ctx, user, project, v.Suite, v.Test)
		if err != nil {
			return storage.Record{}, err
		}
		if !retained.Current || retained.SuiteArchived || retained.Test.Type != "https" || retained.Test.Runner != "http" || retained.Test.Check == nil || retained.Test.Check.Endpoint != v.Endpoint || retained.Test.State == "archived" {
			return storage.Record{}, errors.New("choose a current native HTTPS definition with this endpoint")
		}
		envRow, err := s.Store.Get(ctx, projects.Bucket("environments", project), v.Environment)
		if err != nil {
			return storage.Record{}, err
		}
		env, err := storage.Decode[projects.Environment](envRow)
		if err != nil {
			return storage.Record{}, err
		}
		suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), v.Suite)
		if err != nil {
			return storage.Record{}, err
		}
		suite, err := storage.Decode[projects.Suite](suiteRow)
		if err != nil {
			return storage.Record{}, err
		}
		if env.Endpoints[v.Endpoint] == "" || len(suite.Applications) > 0 && !slices.Contains(suite.Applications, env.Application) || retained.Test.Application != "" && retained.Test.Application != env.Application {
			return storage.Record{}, errors.New("the environment must belong to the selected test and suite application")
		}
		if len(env.Locations) > 0 && !slices.Contains(env.Locations, v.Location) {
			return storage.Record{}, errors.New("execution location is not enabled for this target")
		}
		testRow, err := s.Store.Get(ctx, catalog.Bucket("tests", project, v.Suite), v.Test)
		if err != nil {
			return storage.Record{}, err
		}
		changes = append(changes, storage.Change{Record: envRow, Assert: true}, storage.Change{Record: suiteRow, Assert: true}, storage.Change{Record: testRow, Assert: true})
	}
	c, err := storage.Put(Bucket(project), id, project, revision, v)
	if err != nil {
		return storage.Record{}, err
	}
	audit, err := domain.Audit(user, project, "probe.save", id, before.Data, v)
	if err != nil {
		return storage.Record{}, err
	}
	changes = append(changes, c, audit)
	if err = s.Store.Commit(ctx, changes...); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, Bucket(project), id)
}
func (s *Service) Read(ctx context.Context, user, project, id string) (Binding, storage.Record, error) {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return Binding{}, storage.Record{}, errors.New("valid project and probe identifiers are required")
	}
	row, err := s.Store.Get(ctx, Bucket(project), id)
	if err != nil {
		return Binding{}, row, err
	}
	v, err := storage.Decode[Binding](row)
	if err != nil {
		return v, row, err
	}
	if row.Project != project || v.ID != id || !valid(project, v) {
		return v, row, storage.ErrNotFound
	}
	return v, row, s.authority(ctx, user, project, v, false)
}
func (s *Service) references(ctx context.Context, user, project string, v Binding) (catalog.RetainedTest, projects.Environment, storage.Record, error) {
	t, err := s.Catalog.RetainedTest(ctx, user, project, v.Suite, v.Test)
	if err != nil {
		return t, projects.Environment{}, storage.Record{}, err
	}
	row, err := s.Store.Get(ctx, projects.Bucket("environments", project), v.Environment)
	if err != nil {
		return t, projects.Environment{}, row, err
	}
	e, err := storage.Decode[projects.Environment](row)
	if err != nil {
		return t, e, row, fmt.Errorf("probe target configuration is unavailable")
	}
	return t, e, row, nil
}
