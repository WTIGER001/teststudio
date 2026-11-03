package sources

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) checkRepository(ctx context.Context, user, project, id, action string) error {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return errors.New("valid project and source identifiers are required")
	}
	return s.Auth.Check(ctx, user, project, "source/"+id, "sources."+action)
}
func (s *Service) ReadRepository(ctx context.Context, user, project, id string) (storage.Record, error) {
	if err := s.checkRepository(ctx, user, project, id, "read"); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, repositoryBucket(project), id)
}
func (s *Service) ListRepositories(ctx context.Context, user, project, cursor string, limit int) (storage.Page, error) {
	if err := s.Auth.Check(ctx, user, project, "", "sources.read"); err != nil {
		return storage.Page{}, err
	}
	page, err := s.Store.List(ctx, repositoryBucket(project), project, cursor, limit)
	if err != nil {
		return page, err
	}
	allowed := []storage.Record{}
	for _, row := range page.Items {
		if err := s.checkRepository(ctx, user, project, row.ID, "read"); err == nil {
			allowed = append(allowed, row)
		} else if !errors.Is(err, authorization.ErrForbidden) {
			return storage.Page{}, err
		}
	}
	page.Items = allowed
	return page, nil
}
func (s *Service) SaveRepository(ctx context.Context, user, project, id string, input RepositoryConfig, revision int64) error {
	if err := s.checkRepository(ctx, user, project, id, "manage"); err != nil {
		return err
	}
	provider := s.Providers[input.Provider]
	if provider == nil {
		return errors.New("source provider is not enabled")
	}
	if !utf8.ValidString(input.Name) || strings.TrimSpace(input.Name) != input.Name || len(input.Name) < 1 || len(input.Name) > 200 || strings.ContainsFunc(input.Name, unicode.IsControl) || len(input.Bindings) < 1 || len(input.Bindings) > 20 {
		return errors.New("repository source requires a name and 1 through 20 suite bindings")
	}
	if err := provider.Validate(sourceprovider.Access{Project: project, Repository: input.Repository, Credential: input.Credential}, input.Reference); err != nil {
		return err
	}
	if input.Credential != "" {
		if s.Secrets == nil {
			return errors.New("source credential reference is unavailable")
		}
		refs, err := s.Secrets.References(ctx, project, sourceprovider.CredentialPurpose)
		if err != nil {
			return err
		}
		found := false
		for _, ref := range refs {
			found = found || ref.ID == input.Credential
		}
		if !found {
			return errors.New("source credential reference is not approved for this project")
		}
	}
	if err := s.validateRepositoryBindings(ctx, user, project, input); err != nil {
		return err
	}
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, BundleRunners: s.BundleRunners}
		if err := bound.checkRepository(ctx, user, project, id, "manage"); err != nil {
			return err
		}
		if err := bound.validateRepositoryBindings(ctx, user, project, input); err != nil {
			return err
		}
		source := RepositorySource{ID: id, Project: project, Configuration: input, Actor: user, Generation: 1, LastState: "not synchronized"}
		var before any
		if revision > 0 {
			row, err := bound.Store.Get(ctx, repositoryBucket(project), id)
			if err != nil {
				return err
			}
			previous, err := storage.Decode[RepositorySource](row)
			if err != nil {
				return err
			}
			before = previous
			source = previous
			source.Configuration = input
			source.Actor = user
			source.Generation++
			source.LastState = "configuration changed"
			source.LastError = ""
			if !input.Enabled {
				source.LastState = "disabled"
			}
			if previous.Configuration.Provider != input.Provider || !strings.EqualFold(previous.Configuration.Repository, input.Repository) {
				source.RepositoryID = ""
				source.LastCommit = ""
				source.LastSync = ""
				source.LastState = ""
				source.LastError = ""
			}
		}
		row, err := storage.Put(repositoryBucket(project), id, project, revision, source)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.repository.save", id, before, source)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, row, audit)
	})
}
func (s *Service) validateRepositoryBindings(ctx context.Context, user, project string, input RepositoryConfig) error {
	seen := map[string]bool{}
	for _, binding := range input.Bindings {
		if !domain.ValidID(binding.Suite) || !sourcebundle.SafePath(binding.Manifest) || seen[binding.Suite] || !slices.Contains(s.BundleRunners, binding.Runner) || binding.Profile != "go-module" || binding.Runner != "go" {
			return errors.New("each source binding requires a distinct suite, manifest path, and enabled build profile")
		}
		seen[binding.Suite] = true
		for _, permission := range []string{"sources.manage", "catalog.read", "catalog.import"} {
			if err := s.Auth.Check(ctx, user, project, "suite/"+binding.Suite, permission); err != nil {
				return err
			}
		}
		row, err := s.Store.Get(ctx, projects.Bucket("suites", project), binding.Suite)
		if err != nil {
			return err
		}
		suite, err := storage.Decode[projects.Suite](row)
		if err != nil {
			return err
		}
		if suite.Archived || suite.RequiresTarget {
			return errors.New("current repository build profiles require an active targetless suite")
		}
		if input.AutoActivate {
			for _, permission := range []string{"catalog.publish", "suites.manage"} {
				if err := s.Auth.Check(ctx, user, project, "suite/"+binding.Suite, permission); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type RepositoryOptions struct {
	Providers   []string            `json:"providers"`
	Credentials []secrets.Reference `json:"credentials"`
}

func (s *Service) RepositoryOptions(ctx context.Context, user, project string) (RepositoryOptions, error) {
	result := RepositoryOptions{Providers: []string{}, Credentials: []secrets.Reference{}}
	if err := s.Auth.Check(ctx, user, project, "", "sources.read"); err != nil {
		return result, err
	}
	for provider := range s.Providers {
		result.Providers = append(result.Providers, provider)
	}
	slices.Sort(result.Providers)
	if err := s.Auth.Check(ctx, user, project, "", "sources.manage"); err == nil && s.Secrets != nil {
		refs, err := s.Secrets.References(ctx, project, sourceprovider.CredentialPurpose)
		if err != nil {
			return result, err
		}
		result.Credentials = refs
	} else if err != nil && !errors.Is(err, authorization.ErrForbidden) {
		return result, err
	}
	return result, nil
}
