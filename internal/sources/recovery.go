package sources

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) RecoverPreparations(ctx context.Context, worker string) error {
	page, err := s.Store.List(ctx, leaseBucket(s.Location), "", "", 100)
	if err != nil {
		return err
	}
	for _, index := range page.Items {
		item, err := storage.Decode[jobIndex](index)
		if err != nil {
			return err
		}
		var claimed Job
		recovered := false
		err = s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
			row, err := store.Get(ctx, JobBucket(index.Project, item.Suite), item.Job)
			if err != nil {
				return err
			}
			job, err := storage.Decode[Job](row)
			if err != nil {
				return err
			}
			if job.LeaseUntil == nil || time.Now().Before(*job.LeaseUntil) {
				return nil
			}
			driver := s.Inspectors[job.Runner]
			if driver == nil {
				driver = s.RecoveryInspectors[job.Runner]
			}
			if driver == nil || job.Reference == nil {
				return nil
			}
			lease := time.Now().UTC().Add(30 * time.Second)
			job.Worker, job.LeaseUntil, job.Phase = worker, &lease, "recovering"
			saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			if err != nil {
				return err
			}
			if err := store.Commit(ctx, saved); err != nil {
				return err
			}
			claimed = job
			recovered = true
			return nil
		})
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		if err != nil {
			return err
		}
		if !recovered {
			continue
		}
		driver := s.Inspectors[claimed.Runner]
		if driver == nil {
			driver = s.RecoveryInspectors[claimed.Runner]
		}
		cleanupContext, cancel := context.WithTimeout(ctx, 15*time.Second)
		cleanupErr := driver.CleanupInspection(cleanupContext, preparation.InspectionRequest{Image: claimed.Image, Suite: claimed.Suite, Reference: *claimed.Reference})
		cancel()
		err = s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
			row, err := store.Get(ctx, JobBucket(claimed.Project, claimed.Suite), claimed.ID)
			if err != nil {
				return err
			}
			job, err := storage.Decode[Job](row)
			if err != nil {
				return err
			}
			if job.Worker != worker || job.Phase != "recovering" {
				return storage.ErrConflict
			}
			now := time.Now().UTC()
			job.State, job.Phase, job.Error = "failed", "finished", "Preparation was interrupted. No tests were rerun; prepare the image again explicitly."
			job.FinishedAt = &now
			job.LeaseUntil = nil
			job.CleanupPending = cleanupErr != nil
			if job.CancelRequested {
				job.State, job.Error = "cancelled", ""
			}
			changes := []storage.Change{}
			if cleanupErr != nil {
				retry := now.Add(10 * time.Second)
				job.LeaseUntil = &retry
				job.Phase = "cleanup-pending"
			} else {
				changes = append(changes, storage.Change{Record: index, Delete: true})
			}
			saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			if err != nil {
				return err
			}
			return store.Commit(ctx, append(changes, saved)...)
		})
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return nil
}
