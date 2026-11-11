package sources

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/preparation"
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
	if err := s.Auth.CheckCollection(ctx, user, project, "sources.read"); err != nil {
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
	if err := s.repositoryMutationAuthority(ctx, user, project, id, input, revision); err != nil {
		return err
	}

	if input.Mode == "" {
		input.Mode = "read-only"
	}
	if input.Mode != "read-only" && input.Mode != "read-write" {
		return errors.New("repository access mode must be read-only or read-write")
	}
	if input.Mode == "read-write" && (!strings.HasPrefix(input.Reference, "heads/") || input.Credential == "") {
		return errors.New("read/write metadata requires a branch and an approved write credential")
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
	if input.IntervalSeconds != 0 && (input.IntervalSeconds < 60 || input.IntervalSeconds > 86400 || !domain.ValidID(s.Location)) {
		return errors.New("periodic source sync requires 60 through 86400 seconds, or zero to disable it")
	}
	if input.Mode == "read-write" {
		if _, ok := provider.(sourceprovider.Writer); !ok {
			return errors.New("provider does not support metadata commits")
		}
	}
	if revision > 0 {
		previousRow, err := s.ReadRepository(ctx, user, project, id)
		if err != nil {
			return err
		}
		previous, err := storage.Decode[RepositorySource](previousRow)
		if err != nil {
			return err
		}
		previousMode := previous.Configuration.Mode
		if previousMode == "" {
			previousMode = "read-only"
		}
		if previousMode != input.Mode && !input.ConfirmModeChange {
			return errors.New("review and confirm the change of metadata authority before switching modes")
		}
		if previous.Configuration.Credential != input.Credential || previousMode != input.Mode {
			if _, err := provider.Resolve(ctx, sourceprovider.Access{Project: project, Repository: input.Repository, RepositoryID: previous.RepositoryID, Credential: input.Credential, Write: input.Mode == "read-write"}, input.Reference); err != nil {
				return errors.New("validate replacement repository access before saving the connection")
			}
		}
	}
	if input.Push {
		if _, ok := provider.(sourceprovider.WebhookVerifier); !ok || input.WebhookCredential == "" || sourceprovider.ValidCommit(input.Reference) {
			return errors.New("push synchronization requires a supported provider, webhook credential reference, and watched branch or tag")
		}
	}
	for purpose, credential := range map[string]string{sourceprovider.CredentialPurpose: input.Credential, sourceprovider.WebhookPurpose: input.WebhookCredential} {
		if credential == "" {
			continue
		}
		if s.Secrets == nil {
			return errors.New("source credential reference is unavailable")
		}
		refs, err := s.Secrets.References(ctx, project, purpose)
		if err != nil {
			return err
		}
		found := false
		for _, ref := range refs {
			found = found || ref.ID == credential
		}
		if !found {
			return errors.New("source credential reference is not approved for this project and purpose")
		}
	}
	if err := s.validateRepositoryBindings(ctx, user, project, input); err != nil {
		return err
	}
	input.ConfirmModeChange = false
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, BundleRunners: s.BundleRunners, Location: s.Location}
		if err := bound.repositoryMutationAuthority(ctx, user, project, id, input, revision); err != nil {
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

		timerChanges, err := bound.configureRepositoryTimer(ctx, &source, time.Now().UTC())
		if err != nil {
			return err
		}
		row, err := storage.Put(repositoryBucket(project), id, project, revision, source)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.repository.save", id, before, source)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, append(timerChanges, row, audit)...)
	})
}
func (s *Service) validateRepositoryBindings(ctx context.Context, user, project string, input RepositoryConfig) error {
	seen := map[string]bool{}
	for _, binding := range input.Bindings {
		if !domain.ValidID(binding.Suite) || !sourcebundle.SafePath(binding.Manifest) || seen[binding.Suite] || !slices.Contains(s.BundleRunners, binding.Runner) || !preparation.LocalSourceProfile(binding.Runner, binding.Profile) {
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
		if binding.Discover && (binding.Runner != "playwright" || !sourcebundle.SafePath(binding.Config) || binding.ModuleRoot != "." && !sourcebundle.SafePath(binding.ModuleRoot) || len(binding.Project) > 200 || strings.ContainsAny(binding.Project, "[]\r\n›>\x00") || len(suite.Applications) != 1) {
			return errors.New("discovery requires a Playwright config, source root, exact project, and one application owner")
		}
		if suite.Archived || suite.RequiresTarget && binding.Endpoint == "" {
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
	Providers          []string            `json:"providers"`
	Credentials        []secrets.Reference `json:"credentials"`
	WebhookCredentials []secrets.Reference `json:"webhookCredentials"`
}

func (s *Service) RepositoryOptions(ctx context.Context, user, project string) (RepositoryOptions, error) {
	result := RepositoryOptions{Providers: []string{}, Credentials: []secrets.Reference{}, WebhookCredentials: []secrets.Reference{}}
	if err := s.Auth.CheckCollection(ctx, user, project, "sources.read"); err != nil {
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
		result.WebhookCredentials, err = s.Secrets.References(ctx, project, sourceprovider.WebhookPurpose)
		if err != nil {
			return result, err
		}
	} else if err != nil && !errors.Is(err, authorization.ErrForbidden) {
		return result, err
	}
	return result, nil
}
