package projects

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store *storage.Store
	Auth  *authorization.Service
}

func Bucket(kind, project string) string {
	if kind == "projects" {
		return kind
	}
	return kind + "/" + project
}

func (s *Service) SaveProject(ctx context.Context, user string, p Project, revision int64) error {
	if !domain.ValidID(p.ID) || strings.TrimSpace(p.Name) == "" {
		return errors.New("valid project identifier and name are required")
	}
	permission, scope := "projects.manage", p.ID
	if revision == 0 {
		permission, scope = "projects.create", ""
	}
	if err := s.Auth.Check(ctx, user, scope, "", permission); err != nil {
		return err
	}
	if err := capture.Validate(capture.Resolve(p.Capture, nil), capture.ResolveLimits(p.CaptureLimits)); err != nil {
		return err
	}
	return s.save(ctx, user, "projects", p.ID, p.ID, revision, p)
}

func (s *Service) SaveApplication(ctx context.Context, user, project string, a Application, revision int64) error {
	if err := s.Auth.Check(ctx, user, project, "", "projects.manage"); err != nil {
		return err
	}
	if !domain.ValidID(a.ID) || strings.TrimSpace(a.Name) == "" {
		return errors.New("valid application identifier and name are required")
	}
	if _, err := s.Store.Get(ctx, "projects", project); err != nil {
		return err
	}
	return s.save(ctx, user, Bucket("applications", project), a.ID, project, revision, a)
}

func (s *Service) SaveEnvironment(ctx context.Context, user, project string, e Environment, revision int64) error {
	if err := s.Auth.Check(ctx, user, project, "", "environments.manage"); err != nil {
		return err
	}
	if !domain.ValidID(e.ID) || e.Name == "" || e.Stage == "" {
		return errors.New("environment identifier, name, and explicit stage are required")
	}
	if _, err := s.Store.Get(ctx, Bucket("applications", project), e.Application); err != nil {
		return err
	}
	for name, address := range e.Endpoints {
		u, err := url.Parse(address)
		if !domain.ValidID(name) || err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Fragment != "" {
			return errors.New("endpoints require named HTTP or HTTPS URLs without embedded credentials")
		}
	}
	if revision > 0 {
		r, err := s.Store.Get(ctx, Bucket("environments", project), e.ID)
		if err != nil {
			return err
		}
		old, err := storage.Decode[Environment](r)
		if err != nil {
			return err
		}
		if old.Application != e.Application {
			return errors.New("environment application is permanent")
		}
	}
	return s.save(ctx, user, Bucket("environments", project), e.ID, project, revision, e)
}

func (s *Service) SaveSuite(ctx context.Context, user, project string, v Suite, revision int64) error {
	if err := s.Auth.Check(ctx, user, project, "", "suites.manage"); err != nil {
		return err
	}
	if !domain.ValidID(v.ID) || v.Name == "" {
		return errors.New("valid suite identifier and name are required")
	}
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return err
	}
	p, err := storage.Decode[Project](projectRow)
	if err != nil {
		return err
	}
	if err := capture.Validate(capture.Resolve(p.Capture, v.Capture), capture.ResolveLimits(p.CaptureLimits)); err != nil {
		return err
	}
	for _, app := range v.Applications {
		if _, err := s.Store.Get(ctx, Bucket("applications", project), app); err != nil {
			return err
		}
	}
	if revision > 0 {
		r, err := s.Store.Get(ctx, Bucket("suites", project), v.ID)
		if err != nil {
			return err
		}
		old, err := storage.Decode[Suite](r)
		if err != nil {
			return err
		}
		v.ActiveRelease = old.ActiveRelease
	} else {
		v.ActiveRelease = ""
	}
	return s.save(ctx, user, Bucket("suites", project), v.ID, project, revision, v)
}

func (s *Service) save(ctx context.Context, user, bucket, id, project string, revision int64, value any) error {
	var before any
	if revision > 0 {
		r, err := s.Store.Get(ctx, bucket, id)
		if err != nil {
			return err
		}
		before = r.Data
	}
	c, err := storage.Put(bucket, id, project, revision, value)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "save", bucket+"/"+id, before, value)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, c, audit)
}
