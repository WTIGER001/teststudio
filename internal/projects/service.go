package projects

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
	"slices"
)

type Service struct {
	Secrets secrets.Provider
	Store   *storage.Store
	Auth    *authorization.Service
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
	if revision == 0 {
		return s.createProject(ctx, user, p)
	}
	if err := s.Auth.Check(ctx, user, p.ID, "", "projects.manage"); err != nil {
		return err
	}
	if err := capture.Validate(capture.Resolve(p.Capture, nil), capture.ResolveLimits(p.CaptureLimits)); err != nil {
		return err
	}
	if err := timing.Validate(timing.Resolve(p.Timing, nil)); err != nil {
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
	if len(e.Credentials) > 16 {
		return errors.New("an environment supports at most 16 API credential bindings")
	}
	if len(e.Credentials) > 0 {
		refs, err := s.CredentialReferences(ctx, user, project)
		if err != nil {
			return err
		}
		for binding, id := range e.Credentials {
			if !domain.ValidID(binding) || !domain.ValidID(id) || !slices.ContainsFunc(refs, func(ref secrets.Reference) bool { return ref.ID == id }) {
				return errors.New("credential bindings require valid names and registered project API credential references")
			}
			ref := refs[slices.IndexFunc(refs, func(ref secrets.Reference) bool { return ref.ID == id })]
			approved := false
			for _, address := range e.Endpoints {
				endpoint, _ := url.Parse(address)
				origin, err := browserpolicy.Origin(endpoint.Scheme + "://" + endpoint.Host)
				if err == nil && slices.Contains(ref.Origins, origin) {
					approved = true
				}
			}
			if !approved {
				return errors.New("the API credential reference is not approved for an environment endpoint origin")
			}
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

func (s *Service) CredentialReferences(ctx context.Context, user, project string) ([]secrets.Reference, error) {
	if err := s.Auth.Check(ctx, user, project, "", "environments.manage"); err != nil {
		return nil, err
	}
	if s.Secrets == nil {
		return []secrets.Reference{}, nil
	}
	refs, err := s.Secrets.References(ctx, project, secrets.APICredential)
	if err != nil {
		return nil, errors.New("API credential references are unavailable")
	}
	return refs, nil
}

func (s *Service) SaveSuite(ctx context.Context, user, project string, value Suite, revision int64) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		return (&Service{Store: auth.Directory.Store, Auth: auth}).saveSuite(ctx, user, project, value, revision)
	})
}

func (s *Service) saveSuite(ctx context.Context, user, project string, v Suite, revision int64) error {
	for _, permission := range []string{"suites.read", "suites.manage"} {
		if err := s.Auth.Check(ctx, user, project, "suite/"+v.ID, permission); err != nil {
			return err
		}
	}
	if err := validSuiteSettings(v); err != nil {
		return err
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
	if err := timing.Validate(timing.Resolve(p.Timing, v.Timing)); err != nil {
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
