package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/storage"
)

type repositoryPreparation struct {
	MetadataFiles map[string]string
	Sync          string
	Worker        string
	Source        string
	Generation    int64
	Sequence      int64
	Snapshot      string
	Origin        catalog.Provenance
	Root          string
}
type repositoryContent struct {
	Content []byte `json:"content"`
}

func repositoryContentBucket(project string) string { return "repository-snapshot-content/" + project }

func (s *Service) checkRepositoryJob(ctx context.Context, job Job, activate bool) error {
	if job.RepositorySource == "" {
		return nil
	}
	if err := s.checkRepository(ctx, job.Actor, job.Project, job.RepositorySource, "manage"); err != nil {
		return err
	}
	row, err := s.Store.Get(ctx, repositoryBucket(job.Project), job.RepositorySource)
	if err != nil {
		return err
	}
	source, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return err
	}
	if !source.Configuration.Enabled || source.Generation != job.SourceGeneration {
		return errors.New("repository source is disabled or its configuration changed")
	}
	if activate && source.Sequence != job.SourceSequence {
		return storage.ErrConflict
	}
	return nil
}
func (s *Service) loadRepositorySnapshot(ctx context.Context, project, id, source string) (RepositorySnapshot, error) {
	row, err := s.Store.Get(ctx, snapshotBucket(project), id)
	if err != nil {
		return RepositorySnapshot{}, err
	}
	snapshot, err := storage.Decode[RepositorySnapshot](row)
	if err != nil {
		return snapshot, err
	}
	if snapshot.Project != project || snapshot.Source != source || snapshot.ID != id {
		return snapshot, errors.New("repository snapshot is outside its registered source")
	}
	dataRow, err := s.Store.Get(ctx, repositoryContentBucket(project), id)
	if err != nil {
		return snapshot, err
	}
	data, err := storage.Decode[repositoryContent](dataRow)
	if err != nil {
		return snapshot, err
	}
	if digest(string(data.Content)) != snapshot.Digest {
		return snapshot, errors.New("repository snapshot digest does not match stored content")
	}
	snapshot.Content = data.Content
	return snapshot, nil
}
func (s *Service) referenceRepositorySnapshot(ctx context.Context, project, id string, delta int, publish bool, expiry time.Time) ([]storage.Change, error) {
	row, err := s.Store.Get(ctx, snapshotBucket(project), id)
	if err != nil {
		return nil, err
	}
	snapshot, err := storage.Decode[RepositorySnapshot](row)
	if err != nil {
		return nil, err
	}
	snapshot.References += delta
	if snapshot.References < 0 {
		return nil, errors.New("repository snapshot reference count cannot be negative")
	}
	snapshot.Published = snapshot.Published || publish
	if expiry.After(snapshot.ExpiresAt) {
		snapshot.ExpiresAt = expiry
	}
	saved, err := storage.Put(row.Bucket, row.ID, project, row.Revision, snapshot)
	if err != nil {
		return nil, err
	}
	return []storage.Change{saved}, nil
}

func (s *Service) SweepRepositorySnapshots(ctx context.Context, cursor string) (string, error) {
	page, err := s.Store.List(ctx, "repository-snapshot-staging", "", cursor, 100)
	if err != nil {
		return cursor, err
	}
	for _, index := range page.Items {
		row, err := s.Store.Get(ctx, snapshotBucket(index.Project), index.ID)
		if err != nil {
			return cursor, err
		}
		snapshot, err := storage.Decode[RepositorySnapshot](row)
		if err != nil {
			return cursor, err
		}
		changes := []storage.Change{{Record: index, Delete: true}}
		if !snapshot.Published {
			if snapshot.References > 0 || time.Now().Before(snapshot.ExpiresAt) {
				continue
			}
			content, err := s.Store.Get(ctx, repositoryContentBucket(index.Project), index.ID)
			if err != nil {
				return cursor, err
			}
			changes = append(changes, storage.Change{Record: row, Delete: true}, storage.Change{Record: content, Delete: true})
		} else {
			changes = append(changes, storage.Change{Record: row, Assert: true})
		}
		if err := s.Store.Commit(ctx, changes...); err != nil && !errors.Is(err, storage.ErrConflict) {
			return cursor, err
		}
	}
	return page.Cursor, nil
}
