package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) EnqueueRepositorySync(ctx context.Context, user, project, id, key string, input SyncInput) (result storage.Record, err error) {
	if err := s.checkRepository(ctx, user, project, id, "manage"); err != nil {
		return result, err
	}
	if len(key) < 1 || len(key) > 128 || !domain.ValidID(s.Location) {
		return result, errors.New("repository sync requires an idempotency key and configured location")
	}
	syncID := digest("repository-sync\x00" + project + "\x00" + id + "\x00" + user + "\x00" + key)[7:]
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, BundleRunners: s.BundleRunners}
		if err := bound.checkRepository(ctx, user, project, id, "manage"); err != nil {
			return err
		}
		sourceRow, err := bound.Store.Get(ctx, repositoryBucket(project), id)
		if err != nil {
			return err
		}
		source, err := storage.Decode[RepositorySource](sourceRow)
		if err != nil {
			return err
		}
		if !source.Configuration.Enabled {
			return errors.New("repository synchronization is disabled")
		}
		if s.Providers[source.Configuration.Provider] == nil {
			return errors.New("repository source provider is not enabled")
		}
		if err := bound.validateRepositoryBindings(ctx, user, project, source.Configuration); err != nil {
			return err
		}
		if err := bound.checkRepository(ctx, source.Actor, project, id, "manage"); err != nil {
			return err
		}
		if err := bound.validateRepositoryBindings(ctx, source.Actor, project, source.Configuration); err != nil {
			return err
		}
		if existing, err := bound.Store.Get(ctx, syncBucket(project, id), syncID); err == nil {
			sync, err := storage.Decode[RepositorySync](existing)
			if err != nil {
				return err
			}
			if sync.Force != input.Force || sync.SourceGeneration != source.Generation {
				return errors.New("sync key cannot be reused after changing its source or retry policy")
			}
			result = existing
			return nil
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		source.Sequence++
		now := time.Now().UTC()
		sync := RepositorySync{ID: syncID, Project: project, Source: id, Actor: source.Actor, RequestedBy: user, SourceGeneration: source.Generation, Sequence: source.Sequence, Configuration: source.Configuration, RepositoryID: source.RepositoryID, Force: input.Force, State: "received", CreatedAt: now, Preparations: map[string]string{}}
		row, err := storage.Put(syncBucket(project, id), syncID, project, 0, sync)
		if err != nil {
			return err
		}
		queued, err := storage.Put(syncQueue(s.Location), now.Format("20060102T150405.000000000")+"-"+syncID, project, 0, syncIndex{Source: id, Sync: syncID})
		if err != nil {
			return err
		}
		source.LastSync, source.LastState, source.LastError = syncID, "received", ""
		updated, err := storage.Put(sourceRow.Bucket, sourceRow.ID, project, sourceRow.Revision, source)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.repository.sync", id, nil, map[string]any{"syncId": syncID, "force": input.Force})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, row, queued, updated, audit); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, row.Bucket, row.ID)
		return err
	})
	if errors.Is(err, storage.ErrConflict) {
		if row, readErr := s.ReadRepositorySync(ctx, user, project, id, syncID); readErr == nil {
			sync, _ := storage.Decode[RepositorySync](row)
			if sync.Force == input.Force {
				return row, nil
			}
		}
	}
	return result, err
}
func (s *Service) ReadRepositorySync(ctx context.Context, user, project, source, id string) (storage.Record, error) {
	if err := s.checkRepository(ctx, user, project, source, "read"); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, syncBucket(project, source), id)
}
func (s *Service) ListRepositorySyncs(ctx context.Context, user, project, source, cursor string, limit int) (storage.Page, error) {
	if err := s.checkRepository(ctx, user, project, source, "read"); err != nil {
		return storage.Page{}, err
	}
	return s.Store.ListNewest(ctx, syncBucket(project, source), project, cursor, limit)
}
