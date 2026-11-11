package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) CheckRunSource(ctx context.Context, user, project, suite string, release catalog.Release) (*time.Time, []storage.Record, error) {
	sourceID := release.Draft.Artifact.RepositorySource
	if sourceID == "" {
		if release.Draft.Artifact.Image == "" {
			return nil, nil, nil
		}
		var err error
		sourceID, err = s.SourceForSuite(ctx, project, suite)
		if err != nil {
			return nil, nil, err
		}
		if sourceID == "" {
			return nil, nil, nil
		}
	}
	row, err := s.ReadRepository(ctx, user, project, sourceID)
	if err != nil {
		return nil, nil, err
	}
	source, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return nil, nil, err
	}
	if !source.Configuration.Enabled {
		return nil, nil, errors.New("repository synchronization is disabled; explicitly choose a retained version to run it")
	}
	provider := s.Providers[source.Configuration.Provider]
	if provider == nil {
		return nil, nil, sourceprovider.ErrUnavailable
	}
	access := sourceprovider.Access{Project: project, Repository: source.Configuration.Repository, RepositoryID: source.RepositoryID, Credential: source.Configuration.Credential}
	revision, err := provider.Resolve(ctx, access, source.Configuration.Reference)
	if err != nil {
		return nil, nil, errors.New("Repository refresh failed. No tests started and no older source version was substituted.")
	}
	if revision.RepositoryID != source.RepositoryID || revision.Reference != source.Configuration.Reference || !sourceprovider.ValidCommit(revision.Commit) {
		return nil, nil, errors.New("repository refresh returned a changed identity or invalid commit")
	}
	checked := time.Now().UTC()
	artifact := release.Draft.Artifact
	metadataChanged := (source.Configuration.Mode == "read-write") != artifact.GitMetadata
	guards := []storage.Record{row}
	metadataRow, readErr := s.Store.Get(ctx, metadataBucket(project, sourceID), suite)
	if readErr == nil {
		m, err := storage.Decode[RepositoryMetadata](metadataRow)
		if err != nil {
			return nil, nil, err
		}
		if source.Configuration.Mode == "read-only" || source.Configuration.Mode == "" {
			metadataChanged = metadataChanged || metadataContentDigest(m) != artifact.MetadataDigest
		}
		guards = append(guards, metadataRow)
	} else if !errors.Is(readErr, storage.ErrNotFound) {
		return nil, nil, readErr
	}
	if artifact.Provenance != nil && artifact.Provenance.Commit == revision.Commit && !metadataChanged {
		return &checked, guards, nil
	}
	if err = s.checkRepository(ctx, source.Actor, project, sourceID, "manage"); err != nil {
		return nil, nil, err
	}
	_, err = s.enqueueRepositorySync(ctx, source.Actor, project, sourceID, "run-source/"+digest(fmt.Sprintf("%d/%s/%s", source.Generation, revision.Commit, artifact.MetadataDigest))[7:], SyncInput{}, repositoryTrigger{Kind: "run", Generation: source.Generation, Revision: &revision})
	if err != nil {
		return nil, nil, err
	}
	return nil, nil, errors.New("Repository source changed. Setup has been queued for the resolved commit. Wait for setup, then refresh run review. No older source version will run.")
}
func metadataContentDigest(m RepositoryMetadata) string {
	tests := make([]catalog.Test, len(m.Catalog.Tests))
	copy(tests, m.Catalog.Tests)
	for i := range tests {
		tests[i].DefinitionSource = ""
		tests[i].RepositorySource = ""
	}
	document := m.Catalog
	document.Tests = tests
	data, _ := json.Marshal([]any{m.Manifest, document})
	return digest(string(data))
}

func (s *Service) SourceForSuite(ctx context.Context, project, suite string) (string, error) {
	cursor := ""
	found := ""
	for {
		page, err := s.Store.List(ctx, repositoryBucket(project), project, cursor, 100)
		if err != nil {
			return "", err
		}
		for _, row := range page.Items {
			source, err := storage.Decode[RepositorySource](row)
			if err != nil {
				return "", err
			}
			for _, binding := range source.Configuration.Bindings {
				if binding.Suite == suite {
					if found != "" {
						return "", errors.New("suite has multiple repository bindings; select one authoritative source")
					}
					found = source.ID
				}
			}
		}
		if page.Cursor == "" {
			return found, nil
		}
		cursor = page.Cursor
	}
}
