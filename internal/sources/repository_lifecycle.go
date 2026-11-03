package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/suitespec"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) pinRepositoryRevision(ctx context.Context, sync RepositorySync) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, BundleRunners: s.BundleRunners}
		if err := bound.checkSync(ctx, sync); err != nil {
			return err
		}
		if err := bound.checkSyncLease(ctx, sync); err != nil {
			return err
		}
		sourceRow, err := bound.Store.Get(ctx, repositoryBucket(sync.Project), sync.Source)
		if err != nil {
			return err
		}
		source, err := storage.Decode[RepositorySource](sourceRow)
		if err != nil {
			return err
		}
		if source.RepositoryID != "" && source.RepositoryID != sync.Revision.RepositoryID {
			return errors.New("repository identity changed; register a new source explicitly")
		}
		source.RepositoryID = sync.Revision.RepositoryID
		sourceChange, err := storage.Put(sourceRow.Bucket, sourceRow.ID, sync.Project, sourceRow.Revision, source)
		if err != nil {
			return err
		}
		row, err := bound.Store.Get(ctx, syncBucket(sync.Project, sync.Source), sync.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[RepositorySync](row)
		if err != nil {
			return err
		}
		if current.Worker != sync.Worker || current.State != "fetching" {
			return storage.ErrConflict
		}
		current.Revision = sync.Revision
		updated, err := storage.Put(row.Bucket, row.ID, sync.Project, row.Revision, current)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, sourceChange, updated)
	})
}
func (s *Service) adoptRepositoryPreparation(ctx context.Context, sync RepositorySync, job Job) error {
	if job.State == "published" || job.SourceSequence >= sync.Sequence {
		return nil
	}
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, BundleRunners: s.BundleRunners}
		if err := bound.checkSync(ctx, sync); err != nil {
			return err
		}
		if err := bound.checkSyncLease(ctx, sync); err != nil {
			return err
		}
		row, err := bound.Store.Get(ctx, JobBucket(job.Project, job.Suite), job.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Job](row)
		if err != nil {
			return err
		}
		if current.SourceGeneration != sync.SourceGeneration || current.RepositorySource != sync.Source || current.Origin == nil || current.Origin.Commit != sync.Revision.Commit {
			return errors.New("reused preparation does not match its source generation")
		}
		if current.State == "published" || current.SourceSequence >= sync.Sequence {
			return nil
		}
		current.SourceSequence = sync.Sequence
		current.RepositorySync = sync.ID
		changes := []storage.Change{}
		if current.State == "ready" {
			content, err := bound.Content(ctx, current.Actor, current.Project, current.Suite, current.Upload)
			if err != nil {
				return err
			}
			var discovery suitespec.Discovery
			if json.Unmarshal([]byte(content.Content), &discovery) != nil {
				return errors.New("retained repository discovery cannot be read")
			}
			upload, err := bound.stagePrepared(ctx, current, preparation.Inspection{Project: current.Project, Image: current.Image, ImageID: current.ImageID, Runtime: current.Runtime, Discovery: discovery})
			if err != nil {
				return err
			}
			current.Upload = upload.ID
			current.Error = ""
			if upload.State == "failed" {
				current.State, current.Error = "failed", "Current catalog validation failed. Review the retained source diagnostics."
				now := time.Now().UTC()
				current.LeaseUntil = &now
				current.CleanupPending = true
				current.Phase = "cleanup-pending"
				lease, err := storage.Put(leaseBucket(current.Location), current.ID, current.Project, 0, jobIndex{Job: current.ID, Suite: current.Suite, Location: current.Location})
				if err != nil {
					return err
				}
				changes = append(changes, lease)
			}
		}
		queued, err := bound.queueRepositoryPublication(ctx, current)
		if err != nil {
			return err
		}
		changes = append(changes, queued...)

		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, current)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, append(changes, saved)...)
	})
}
func (s *Service) syncHeartbeat(ctx context.Context, cancel context.CancelFunc, claimed RepositorySync, worker string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			row, err := s.Store.Get(ctx, syncBucket(claimed.Project, claimed.Source), claimed.ID)
			if err != nil {
				cancel()
				return
			}
			sync, err := storage.Decode[RepositorySync](row)
			if err != nil || sync.Worker != worker || sync.State != "fetching" {
				cancel()
				return
			}
			if err := s.checkSync(ctx, sync); err != nil {
				cancel()
				return
			}
			lease := time.Now().UTC().Add(30 * time.Second)
			sync.LeaseUntil = &lease
			change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, sync)
			if err != nil {
				cancel()
				return
			}
			if err := s.Store.Commit(ctx, change); err != nil && !errors.Is(err, storage.ErrConflict) {
				cancel()
				return
			}
		}
	}
}
func (s *Service) finishRepositorySync(ctx context.Context, claimed RepositorySync, worker string, jobs map[string]string, failure error) error {
	return s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
		row, err := store.Get(ctx, syncBucket(claimed.Project, claimed.Source), claimed.ID)
		if err != nil {
			return err
		}
		sync, err := storage.Decode[RepositorySync](row)
		if err != nil {
			return err
		}
		if sync.Worker != worker || sync.State != "fetching" {
			return storage.ErrConflict
		}
		now := time.Now().UTC()
		sync.State, sync.Error = "completed", ""
		sync.FinishedAt = &now
		sync.LeaseUntil = nil
		sync.Preparations = jobs
		if failure != nil {
			sync.State, sync.Error = "failed", failure.Error()
		}
		updated, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, sync)
		if err != nil {
			return err
		}
		lease, err := store.Get(ctx, syncLeases(s.Location), sync.ID)
		if err != nil {
			return err
		}
		changes := []storage.Change{updated, {Record: lease, Delete: true}}
		sourceRow, err := store.Get(ctx, repositoryBucket(sync.Project), sync.Source)
		if err != nil {
			return err
		}
		source, err := storage.Decode[RepositorySource](sourceRow)
		if err != nil {
			return err
		}
		if source.Generation == sync.SourceGeneration && source.LastSync == sync.ID {
			source.LastState, source.LastError, source.LastSyncAt = sync.State, sync.Error, &now
			if sync.Revision != nil {
				source.LastCommit = sync.Revision.Commit
			}
			saved, err := storage.Put(sourceRow.Bucket, sourceRow.ID, sourceRow.Project, sourceRow.Revision, source)
			if err != nil {
				return err
			}
			changes = append(changes, saved)
		}
		return store.Commit(ctx, changes...)
	})
}
func (s *Service) RecoverRepositorySyncs(ctx context.Context) error {
	page, err := s.Store.List(ctx, syncLeases(s.Location), "", "", 100)
	if err != nil {
		return err
	}
	for _, index := range page.Items {
		item, err := storage.Decode[syncIndex](index)
		if err != nil {
			return err
		}
		err = s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
			row, err := store.Get(ctx, syncBucket(index.Project, item.Source), item.Sync)
			if err != nil {
				return err
			}
			sync, err := storage.Decode[RepositorySync](row)
			if err != nil {
				return err
			}
			if sync.LeaseUntil == nil || time.Now().Before(*sync.LeaseUntil) {
				return nil
			}
			if sync.Revision != nil {
				for _, binding := range sync.Configuration.Bindings {
					id := bundleJobID(sync.Actor, sync.Project, binding.Suite, preparationKey(sync, binding))
					if _, err := store.Get(ctx, JobBucket(sync.Project, binding.Suite), id); err == nil {
						sync.Preparations[binding.Suite] = id
					} else if !errors.Is(err, storage.ErrNotFound) {
						return err
					}
				}
			}
			now := time.Now().UTC()
			sync.State = "failed"
			sync.Error = "Repository sync was interrupted. Retained preparations are listed; no builds or tests were repeated."
			sync.FinishedAt = &now
			sync.LeaseUntil = nil
			saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, sync)
			if err != nil {
				return err
			}
			return store.Commit(ctx, saved, storage.Change{Record: index, Delete: true})
		})
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return nil
}
