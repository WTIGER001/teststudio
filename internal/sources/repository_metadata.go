package sources

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type RepositoryMetadata struct {
	CatalogRevision int64                    `json:"catalogRevision,omitempty"`
	Pending         bool                     `json:"pending"`
	Manifest        suitespec.Manifest       `json:"manifest"`
	Catalog         catalog.Document         `json:"catalog"`
	Base            *sourceprovider.Revision `json:"base,omitempty"`
	Actor           string                   `json:"actor"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

func metadataBucket(project, source string) string {
	return "repository-metadata/" + project + "/" + source
}
func metadataFiles(binding RepositoryBinding, m RepositoryMetadata) (map[string]string, error) {
	if binding.Manifest == m.Manifest.Catalog || !slices.Contains([]string{".json", ".yaml", ".yml"}, path.Ext(binding.Manifest)) || !slices.Contains([]string{".json", ".yaml", ".yml"}, path.Ext(m.Manifest.Catalog)) {
		return nil, errors.New("managed Git metadata requires JSON manifest and catalog paths")
	}
	manifest, err := json.MarshalIndent(m.Manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	document := m.Catalog
	document.Tests = append([]catalog.Test(nil), m.Catalog.Tests...)
	for i := range document.Tests {
		document.Tests[i].DefinitionSource = ""
		document.Tests[i].RepositorySource = ""
	}
	catalog, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string]string{binding.Manifest: string(manifest) + "\n", m.Manifest.Catalog: string(catalog) + "\n"}, nil
}
func (s *Service) repositoryBinding(ctx context.Context, user, project, source, suite, action string) (RepositorySource, RepositoryBinding, storage.Record, error) {
	row, err := s.ReadRepository(ctx, user, project, source)
	if err != nil {
		return RepositorySource{}, RepositoryBinding{}, row, err
	}
	registered, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return registered, RepositoryBinding{}, row, err
	}
	if action != "read" {
		if err = s.checkRepository(ctx, user, project, source, action); err != nil {
			return registered, RepositoryBinding{}, row, err
		}
	}
	for _, binding := range registered.Configuration.Bindings {
		if binding.Suite == suite {
			if err = s.check(ctx, user, project, suite, action); err != nil {
				return registered, binding, row, err
			}
			return registered, binding, row, nil
		}
	}
	return registered, RepositoryBinding{}, row, errors.New("suite is not bound to this repository")
}
func (s *Service) ReadRepositoryMetadata(ctx context.Context, user, project, source, suite string) (storage.Record, error) {
	registered, _, _, err := s.repositoryBinding(ctx, user, project, source, suite, "read")
	if err != nil {
		return storage.Record{}, err
	}
	row, err := s.Store.Get(ctx, metadataBucket(project, source), suite)
	if err != nil {
		return row, err
	}
	m, err := storage.Decode[RepositoryMetadata](row)
	if err != nil {
		return row, err
	}
	for i, test := range m.Catalog.Tests {
		if err = s.Auth.Check(ctx, user, project, "test/"+suite+"/"+test.ID, "catalog.read"); err != nil {
			return storage.Record{}, err
		}
		if !m.Pending && registered.Configuration.Mode != "read-write" {
			current, err := s.Store.Get(ctx, catalog.Bucket("tests", project, suite), test.ID)
			if err == nil {
				actual, err := storage.Decode[catalog.Test](current)
				if err != nil {
					return row, err
				}
				if actual.RepositorySource == source {
					m.Catalog.Tests[i] = catalog.MergeManagedMetadata(actual, test)
				}
			} else if !errors.Is(err, storage.ErrNotFound) {
				return row, err
			}
		}
	}
	head, err := s.Store.Get(ctx, "catalog-heads/"+project, suite)
	if err == nil {
		m.CatalogRevision = head.Revision
	} else if !errors.Is(err, storage.ErrNotFound) {
		return row, err
	}
	row.Data, err = json.Marshal(m)
	return row, err
}

func (s *Service) SaveRepositoryMetadata(ctx context.Context, user, project, source, suite string, m RepositoryMetadata, revision int64) error {
	registered, binding, sourceRow, err := s.repositoryBinding(ctx, user, project, source, suite, "manage")
	if err != nil {
		return err
	}
	head, headErr := s.Store.Get(ctx, "catalog-heads/"+project, suite)
	if headErr != nil && !errors.Is(headErr, storage.ErrNotFound) {
		return headErr
	}
	if revision > 0 && m.CatalogRevision != head.Revision {
		return storage.ErrConflict
	}
	if err = s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
		return err
	}
	if m.Manifest.Suite != suite || m.Manifest.Runner != binding.Runner || m.Manifest.BuildProfile != binding.Profile {
		return errors.New("metadata must match its bound suite and profile")
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return err
	}
	owner, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return err
	}
	if len(m.Manifest.Applications) != 1 || len(owner.Applications) != 1 || m.Manifest.Applications[0] != owner.Applications[0] {
		return errors.New("metadata must retain the suite application owner")
	}
	if err = suitespec.ValidateCatalog(m.Manifest, m.Catalog); err != nil {
		return err
	}
	for _, test := range m.Catalog.Tests {
		if err = s.Auth.Check(ctx, user, project, "test/"+suite+"/"+test.ID, "catalog.read"); err != nil {
			return err
		}
		if err = catalog.ValidateTest(test); err != nil {
			return err
		}
	}
	if _, err = metadataFiles(binding, m); err != nil {
		return err
	}
	m.Actor = user
	m.UpdatedAt = time.Now().UTC()
	m.Pending = true
	if revision > 0 {
		row, err := s.Store.Get(ctx, metadataBucket(project, source), suite)
		if err != nil {
			return err
		}
		prior, err := storage.Decode[RepositoryMetadata](row)
		if err != nil {
			return err
		}
		if m.Manifest.Catalog != prior.Manifest.Catalog {
			return errors.New("preserve the established catalog file path when editing metadata")
		}
		requestedBase := m.Base
		m.Base = prior.Base
		if registered.Configuration.Mode == "read-write" && requestedBase != nil && prior.Base != nil && requestedBase.Commit != prior.Base.Commit {
			provider := s.Providers[registered.Configuration.Provider]
			if provider == nil {
				return sourceprovider.ErrUnavailable
			}
			current, err := provider.Resolve(ctx, sourceprovider.Access{Project: project, Repository: registered.Configuration.Repository, RepositoryID: registered.RepositoryID, Credential: registered.Configuration.Credential}, registered.Configuration.Reference)
			if err != nil {
				return errors.New("latest Git base could not be validated")
			}
			if requestedBase.Commit != current.Commit || requestedBase.RepositoryID != current.RepositoryID || requestedBase.URL != current.URL || requestedBase.Reference != current.Reference {
				return storage.ErrConflict
			}
			m.Base = &current
		}
	} else {
		m.Base = nil
	}
	change, err := storage.Put(metadataBucket(project, source), suite, project, revision, m)
	if err != nil {
		return err
	}
	registered.Generation++
	registered.Actor = user
	registered.LastState = "metadata changed"
	registered.LastError = ""
	_, err = storage.Put(sourceRow.Bucket, sourceRow.ID, project, sourceRow.Revision, registered)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "sources.repository.metadata.save", source+"/"+suite, nil, map[string]any{"mode": registered.Configuration.Mode, "digest": digest(string(change.Data))})
	if err != nil {
		return err
	}
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if _, _, _, err := bound.repositoryBinding(ctx, user, project, source, suite, "manage"); err != nil {
			return err
		}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
			return err
		}
		timers, err := bound.configureRepositoryTimer(ctx, &registered, time.Now().UTC())
		if err != nil {
			return err
		}
		updated, err := storage.Put(sourceRow.Bucket, sourceRow.ID, project, sourceRow.Revision, registered)
		if err != nil {
			return err
		}
		history, err := metadataHistoryChange(project, source, suite, change)
		if err != nil {
			return err
		}
		changes := append(timers, change, updated, audit, history, storage.Change{Record: suiteRow, Assert: true})
		if head.Revision > 0 {
			changes = append(changes, storage.Change{Record: head, Assert: true})
		}
		return bound.Store.Commit(ctx, changes...)
	})
}
func (s *Service) repositoryOverlay(ctx context.Context, sync RepositorySync, binding RepositoryBinding, archive sourceprovider.Archive) (map[string]string, error) {
	var m RepositoryMetadata
	row, err := s.Store.Get(ctx, metadataBucket(sync.Project, sync.Source), binding.Suite)
	if err == nil {
		m, err = storage.Decode[RepositoryMetadata](row)
		if err != nil {
			return nil, err
		}
		if sync.Configuration.Mode == "read-write" {
			return nil, nil
		}
		current, err := s.ReadRepositoryMetadata(ctx, sync.Actor, sync.Project, sync.Source, binding.Suite)
		if err != nil {
			return nil, err
		}
		m, err = storage.Decode[RepositoryMetadata](current)
		if err != nil {
			return nil, err
		}
		return metadataFiles(binding, m)
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if !binding.Discover {
		return nil, nil
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", sync.Project), binding.Suite)
	if err != nil {
		return nil, err
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return nil, err
	}
	m.Manifest = suitespec.Manifest{Schema: 1, Suite: binding.Suite, Runner: binding.Runner, Type: "browser", Applications: suite.Applications, ModuleRoot: binding.ModuleRoot, Tests: suitespec.Tests{Config: binding.Config, Project: binding.Project}, Catalog: strings.TrimSuffix(binding.Manifest, ".json") + ".catalog.json", BuildProfile: binding.Profile, Discover: true}
	if binding.Endpoint != "" {
		m.Manifest.Browser = &browserpolicy.Spec{Endpoint: binding.Endpoint}
	}
	m.Catalog = catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{}}
	raw, err := sourcebundle.ReadRoot(archive.Content, archive.Root)
	if err != nil {
		return nil, err
	}
	if sync.Configuration.Mode == "read-write" {
		if _, exists := raw.Files[binding.Manifest]; exists {
			return nil, nil
		}
	}
	for name, body := range raw.Files {
		if name == ".gitmodules" || strings.HasPrefix(string(body), "version https://git-lfs.github.com/spec/v1\n") {
			return nil, errors.New("repository requires submodule or large-file assets; supply complete prepared source")
		}
	}
	return metadataFiles(binding, m)
}

func (s *Service) discoveredMetadata(ctx context.Context, job Job, discovery suitespec.Discovery) (*storage.Change, error) {
	sourceRow, err := s.Store.Get(ctx, repositoryBucket(job.Project), job.RepositorySource)
	if err != nil {
		return nil, err
	}
	source, err := storage.Decode[RepositorySource](sourceRow)
	if err != nil {
		return nil, err
	}
	row, err := s.Store.Get(ctx, metadataBucket(job.Project, job.RepositorySource), job.Suite)
	revision := int64(0)
	if err == nil {
		revision = row.Revision
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	pending := false
	if source.Configuration.Mode == "read-write" {
		bundleRow, err := s.Store.Get(ctx, bundleBucket(job.Project), job.Bundle)
		if err != nil {
			return nil, err
		}
		bundle, err := storage.Decode[Bundle](bundleRow)
		if err != nil {
			return nil, err
		}
		pending = len(bundle.MetadataFiles) > 0
		if pending && revision > 0 {
			return nil, nil
		}
	}
	m := RepositoryMetadata{Manifest: discovery.Manifest, Catalog: discovery.Catalog, Actor: job.Actor, UpdatedAt: time.Now().UTC(), Pending: pending}
	if job.Origin != nil {
		m.Base = &sourceprovider.Revision{RepositoryID: source.RepositoryID, Repository: source.Configuration.Repository, URL: job.Origin.Repository, Reference: job.Origin.Reference, Commit: job.Origin.Commit}
	}
	if revision > 0 && source.Configuration.Mode == "read-write" {
		prior, err := storage.Decode[RepositoryMetadata](row)
		if err != nil {
			return nil, err
		}
		if prior.Pending && metadataContentDigest(prior) != metadataContentDigest(m) {
			return nil, nil
		}
	}
	change, err := storage.Put(metadataBucket(job.Project, job.RepositorySource), job.Suite, job.Project, revision, m)
	return &change, err
}

func (s *Service) RemoteRepositoryMetadata(ctx context.Context, user, project, source, suite string) (RepositoryMetadata, error) {
	registered, binding, _, err := s.repositoryBinding(ctx, user, project, source, suite, "manage")
	if err != nil {
		return RepositoryMetadata{}, err
	}
	provider := s.Providers[registered.Configuration.Provider]
	if provider == nil {
		return RepositoryMetadata{}, sourceprovider.ErrUnavailable
	}
	access := sourceprovider.Access{Project: project, Repository: registered.Configuration.Repository, RepositoryID: registered.RepositoryID, Credential: registered.Configuration.Credential}
	revision, err := provider.Resolve(ctx, access, registered.Configuration.Reference)
	if err != nil {
		return RepositoryMetadata{}, errors.New("latest Git metadata could not be resolved")
	}
	archive, err := provider.Archive(ctx, access, revision)
	if err != nil {
		return RepositoryMetadata{}, errors.New("latest Git metadata could not be downloaded")
	}
	snapshot, err := sourcebundle.ValidateRoot(archive.Content, binding.Manifest, suite, archive.Root)
	if err != nil {
		return RepositoryMetadata{}, err
	}
	return RepositoryMetadata{Manifest: snapshot.Manifest, Catalog: snapshot.Catalog, Base: &revision, Actor: user, UpdatedAt: time.Now().UTC()}, nil
}
