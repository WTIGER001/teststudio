package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store *storage.Store
	Auth  *authorization.Service
}

func Bucket(kind, project, suite string) string { return kind + "/" + project + "/" + suite }

func (s *Service) suite(ctx context.Context, project, id string) (projects.Suite, storage.Record, error) {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return projects.Suite{}, storage.Record{}, errors.New("invalid project or suite identifier")
	}
	r, err := s.Store.Get(ctx, projects.Bucket("suites", project), id)
	if err != nil {
		return projects.Suite{}, r, err
	}
	v, err := storage.Decode[projects.Suite](r)
	if err == nil && v.Archived {
		err = errors.New("suite is archived")
	}
	return v, r, err
}

func (s *Service) SaveTest(ctx context.Context, user, project, suite string, t Test, revision int64) error {
	permission := "catalog.update"
	if revision == 0 {
		permission = "catalog.create"
	}
	if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+t.ID, permission); err != nil {
		return err
	}
	v, _, err := s.suite(ctx, project, suite)
	if err != nil {
		return err
	}
	if err := ValidateTest(t); err != nil {
		return err
	}
	if t.Capture != nil {
		_, limits, err := s.captureSettings(ctx, project, suite)
		if err != nil {
			return err
		}
		if err := capture.Validate(*t.Capture, limits); err != nil {
			return err
		}
	}
	if t.Application != "" {
		found := false
		for _, id := range v.Applications {
			found = found || id == t.Application
		}
		if !found {
			return errors.New("test application is outside the suite")
		}
	}
	bucket := Bucket("tests", project, suite)
	var before any
	if revision > 0 {
		r, err := s.Store.Get(ctx, bucket, t.ID)
		if err != nil {
			return err
		}
		old, err := storage.Decode[Test](r)
		if err != nil {
			return err
		}
		if old.Safety != t.Safety {
			if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+t.ID, "catalog.publish"); err != nil {
				return err
			}
		}
		if old.State != t.State {
			return errors.New("use an explicit lifecycle action to change test state")
		}
		before = old
	}
	c, err := storage.Put(bucket, t.ID, project, revision, t)
	if err != nil {
		return err
	}
	history, err := storage.Put(Bucket("test-revisions", project, suite)+"/"+t.ID, fmt.Sprintf("%020d", revision+1), project, 0, Revision{Test: t, Actor: user, At: time.Now().UTC()})
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "catalog.save", suite+"/"+t.ID, before, t)
	if err != nil {
		return err
	}
	return s.commitTests(ctx, project, suite, c, history, audit)
}

func (s *Service) ChangeState(ctx context.Context, user, project, suite, id, state string, revision int64) error {
	permission := "catalog.update"
	if state == "archived" {
		permission = "catalog.archive"
	}
	if state == "enabled" {
		permission = "catalog.restore"
	}
	if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+id, permission); err != nil {
		return err
	}
	if state != "enabled" && state != "disabled" && state != "archived" {
		return errors.New("unsupported lifecycle action")
	}
	r, err := s.Store.Get(ctx, Bucket("tests", project, suite), id)
	if err != nil {
		return err
	}
	t, err := storage.Decode[Test](r)
	if err != nil {
		return err
	}
	before := t
	t.State = state
	c, err := storage.Put(r.Bucket, r.ID, project, revision, t)
	if err != nil {
		return err
	}
	history, err := storage.Put(Bucket("test-revisions", project, suite)+"/"+id, fmt.Sprintf("%020d", revision+1), project, 0, Revision{Test: t, Actor: user, At: time.Now().UTC()})
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "catalog."+state, suite+"/"+id, before, t)
	if err != nil {
		return err
	}
	return s.commitTests(ctx, project, suite, c, history, audit)
}

func (s *Service) Tests(ctx context.Context, user, project, suite string) ([]storage.Record, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return nil, err
	}
	if _, _, err := s.suite(ctx, project, suite); err != nil {
		return nil, err
	}
	return s.Store.All(ctx, Bucket("tests", project, suite), project)
}

func (s *Service) CheckCurrentState(ctx context.Context, project, suite string, tests []Test, protected bool) error {
	for _, t := range tests {
		if protected && t.Safety != "safe" {
			return fmt.Errorf("test %s is not labeled safe in the selected release", t.ID)
		}
		r, err := s.Store.Get(ctx, Bucket("tests", project, suite), t.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Test](r)
		if err != nil {
			return err
		}
		if current.State != "enabled" {
			return fmt.Errorf("test %s is currently %s", t.ID, current.State)
		}
		if protected && current.Safety != "safe" {
			return fmt.Errorf("test %s is no longer labeled safe", t.ID)
		}
	}
	return nil
}
