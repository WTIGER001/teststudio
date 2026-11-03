package sources

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) SweepBundles(ctx context.Context, cursor string) (string, error) {
	page, err := s.Store.List(ctx, "source-bundle-staging", "", cursor, 100)
	if err != nil {
		return cursor, err
	}
	for _, indexRow := range page.Items {
		index, err := storage.Decode[bundleIndex](indexRow)
		if err != nil {
			return cursor, err
		}
		if time.Now().Before(index.ExpiresAt) {
			continue
		}
		err = s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
			row, err := store.Get(ctx, JobBucket(indexRow.Project, index.Suite), index.Job)
			if err != nil {
				return err
			}
			job, err := storage.Decode[Job](row)
			if err != nil {
				return err
			}
			if job.State == "published" {
				return store.Commit(ctx, storage.Change{Record: indexRow, Delete: true})
			}
			source, err := store.Get(ctx, bundleBucket(job.Project), job.Bundle)
			if err != nil {
				return err
			}
			bundle, err := storage.Decode[Bundle](source)
			if err != nil {
				return err
			}
			if time.Now().Before(bundle.ExpiresAt) {
				return nil
			}
			changes := []storage.Change{}
			if bundle.Snapshot != "" && !activeJob(job.State) {
				referenced, err := (&Service{Store: store}).referenceRepositorySnapshot(ctx, job.Project, bundle.Snapshot, -1, false, time.Time{})
				if err != nil {
					return err
				}
				changes = append(changes, referenced...)
			}
			job.CancelRequested = true
			if activeJob(job.State) {
				saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
				if err != nil {
					return err
				}
				return store.Commit(ctx, saved)
			}
			job.SourceExpired = true
			if job.State != "failed" && job.State != "cancelled" {
				job.State, job.Phase, job.Error = "expired", "finished", "Source bundle staging expired. Upload a new bundle explicitly."
			}
			now := time.Now().UTC()
			if job.FinishedAt == nil {
				job.FinishedAt = &now
			}
			if job.Reference != nil {
				job.CleanupPending = true
				job.LeaseUntil = &now
				job.Phase = "cleanup-pending"
				if _, err := store.Get(ctx, leaseBucket(job.Location), job.ID); errors.Is(err, storage.ErrNotFound) {
					lease, err := storage.Put(leaseBucket(job.Location), job.ID, job.Project, 0, jobIndex{Job: job.ID, Suite: job.Suite, Location: job.Location})
					if err != nil {
						return err
					}
					changes = append(changes, lease)
				} else if err != nil {
					return err
				}
			}
			if queued, err := store.Get(ctx, jobQueue(job), queueID(job)); err == nil {
				changes = append(changes, storage.Change{Record: queued, Delete: true})
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			if err != nil {
				return err
			}
			return store.Commit(ctx, append(changes, saved, storage.Change{Record: source, Delete: true}, storage.Change{Record: indexRow, Delete: true})...)
		})
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			return cursor, err
		}
	}
	return page.Cursor, nil
}
