package sources

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (s *Service) StartPreparation(ctx context.Context) <-chan struct{} {
	return s.startPreparation(ctx, false)
}

func (s *Service) StartBuildPreparation(ctx context.Context) <-chan struct{} {
	return s.startPreparation(ctx, true)
}

func (s *Service) startPreparation(ctx context.Context, builds bool) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker := domain.NewID()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			if err := s.RecoverPreparations(ctx, worker); err != nil && ctx.Err() == nil {
				slog.Error("preparation recovery failed")
			}
			var err error
			if builds {
				if _, publishErr := s.ProcessRepositoryPublication(ctx); publishErr != nil && ctx.Err() == nil {
					slog.Error("repository publication could not persist its outcome")
				}
				_, err = s.ProcessBuildNext(ctx, worker)
			} else {
				_, err = s.ProcessNext(ctx, worker)
			}
			if err != nil && ctx.Err() == nil {
				slog.Error("source preparation failed to persist its result")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}

func (s *Service) ProcessNext(ctx context.Context, worker string) (bool, error) {
	return s.processQueue(ctx, worker, queueBucket(s.Location))
}

func (s *Service) ProcessBuildNext(ctx context.Context, worker string) (bool, error) {
	if len(s.Builders) == 0 {
		return false, nil
	}
	return s.processQueue(ctx, worker, buildQueueBucket(s.Location))
}

func (s *Service) processQueue(ctx context.Context, worker, bucket string) (bool, error) {
	page, err := s.Store.List(ctx, bucket, "", "", 25)
	if err != nil {
		return false, err
	}
	for _, index := range page.Items {
		job, claimed, err := s.claimPreparation(ctx, index, worker)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		if err != nil {
			return false, err
		}
		if !claimed {
			continue
		}
		inspectionContext, cancel := context.WithCancel(ctx)
		heartbeatDone := make(chan struct{})
		go func() { defer close(heartbeatDone); s.preparationHeartbeat(inspectionContext, cancel, job, worker) }()
		result, inspectErr := s.inspectJob(inspectionContext, job, worker)
		cancel()
		<-heartbeatDone
		finishContext, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		err = s.finishPreparation(finishContext, job, worker, result, inspectErr)
		stop()
		return true, err
	}
	return false, nil
}

func (s *Service) claimPreparation(ctx context.Context, index storage.Record, worker string) (job Job, claimed bool, resultErr error) {
	item, err := storage.Decode[jobIndex](index)
	if err != nil {
		return job, false, err
	}
	installation, err := (workerstate.Ledger{Store: s.Store}).Installation(ctx, true)
	if err != nil {
		return job, false, err
	}
	resultErr = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		store := auth.Directory.Store
		row, err := store.Get(ctx, JobBucket(index.Project, item.Suite), item.Job)
		if err != nil {
			return err
		}
		job, err = storage.Decode[Job](row)
		if err != nil {
			return err
		}
		if job.State != "received" {
			return store.Commit(ctx, storage.Change{Record: index, Delete: true})
		}
		driver := s.Inspectors[job.Runner]
		denied := false
		bound := &Service{Store: store, Auth: auth}
		for _, permission := range []string{"sources.manage", "catalog.read", "catalog.import"} {
			err := bound.Auth.Check(ctx, job.Actor, job.Project, "suite/"+job.Suite, permission)
			if errors.Is(err, authorization.ErrForbidden) || errors.Is(err, storage.ErrNotFound) {
				denied = true
				break
			}
			if err != nil {
				return err
			}
		}
		if err := bound.checkRepositoryJob(ctx, job, false); err != nil {
			denied = true
		}
		now := time.Now().UTC()
		job.StartedAt = &now
		changes := []storage.Change{{Record: index, Delete: true}}
		if driver == nil || (job.Bundle != "" && s.Builders[job.Runner] == nil) || job.CancelRequested || denied {
			job.State, job.Phase = "failed", "finished"
			job.Error = "Preparation runner is no longer enabled."
			if denied {
				job.Error = "Current access no longer permits source preparation."
			}
			if job.CancelRequested {
				job.State, job.Error = "cancelled", ""
			}
			job.FinishedAt = &now
		} else {
			ref := driver.Reference(execution.Reference{ID: domain.NewID(), Run: job.ID, Installation: "preparation-" + installation, Project: job.Project, Test: "catalog", Attempt: 1, Worker: worker})
			deadline := now.Add(30 * time.Second)
			job.Reference, job.Worker, job.LeaseUntil = &ref, worker, &deadline
			job.State, job.Phase = "discovering", "discovering"
			if job.Bundle != "" {
				job.State, job.Phase = "preparing", "preparing"
			}
			lease, err := storage.Put(leaseBucket(job.Location), job.ID, job.Project, 0, jobIndex{Job: job.ID, Suite: job.Suite, Location: job.Location})
			if err != nil {
				return err
			}
			changes = append(changes, lease)
			claimed = true
		}
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
		if err != nil {
			return err
		}
		return store.Commit(ctx, append(changes, saved)...)
	})
	return job, claimed && resultErr == nil, resultErr
}

func (s *Service) preparationHeartbeat(ctx context.Context, cancel context.CancelFunc, claimed Job, worker string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			row, err := s.Store.Get(ctx, JobBucket(claimed.Project, claimed.Suite), claimed.ID)
			if err != nil {
				cancel()
				return
			}
			job, err := storage.Decode[Job](row)
			if err != nil || job.Worker != worker || job.CancelRequested || !activeJob(job.State) {
				cancel()
				return
			}
			if err := s.check(ctx, job.Actor, job.Project, job.Suite, "manage"); err != nil {
				cancel()
				return
			}
			if err := s.checkRepositoryJob(ctx, job, false); err != nil {
				cancel()
				return
			}
			lease := time.Now().UTC().Add(30 * time.Second)
			job.LeaseUntil = &lease
			saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			if err != nil {
				cancel()
				return
			}
			if err := s.Store.Commit(ctx, saved); err != nil && !errors.Is(err, storage.ErrConflict) {
				cancel()
				return
			}
		}
	}
}

func (s *Service) finishPreparation(ctx context.Context, claimed Job, worker string, result preparation.Inspection, inspectErr error) error {
	if inspectErr != nil {
		return s.failPreparation(ctx, claimed, worker, result, inspectErr)
	}
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		row, err := bound.Store.Get(ctx, JobBucket(claimed.Project, claimed.Suite), claimed.ID)
		if err != nil {
			return err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return err
		}
		if job.Worker != worker || job.State != "discovering" {
			return storage.ErrConflict
		}
		if job.CancelRequested {
			return context.Canceled
		}
		upload, err := bound.stagePrepared(ctx, job, result)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		job.Upload = upload.ID
		job.State, job.Phase = "ready", "ready"
		job.FinishedAt = &now
		job.LeaseUntil = nil
		job.Log, job.LogTruncated = result.Log, result.LogTruncated
		job.ImageID, job.Runtime = result.ImageID, result.Runtime
		if upload.State == "failed" {
			job.State, job.Phase, job.Error = "failed", "finished", "Catalog validation failed. Inspect the retained source review."
		}
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
		if err != nil {
			return err
		}
		lease, err := bound.Store.Get(ctx, leaseBucket(job.Location), job.ID)
		if err != nil {
			return err
		}
		if job.State == "failed" && job.Bundle != "" {
			job.CleanupPending = true
			retry := now.Add(5 * time.Second)
			job.LeaseUntil = &retry
			job.Phase = "cleanup-pending"
			saved, err = storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			if err != nil {
				return err
			}
			return bound.Store.Commit(ctx, saved)
		}
		queued, err := bound.queueRepositoryPublication(ctx, job)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, append(queued, saved, storage.Change{Record: lease, Delete: true})...)
	})
	if err != nil {
		return s.failPreparation(ctx, claimed, worker, result, err)
	}
	return nil
}

func (s *Service) failPreparation(ctx context.Context, claimed Job, worker string, result preparation.Inspection, failure error) error {
	for range 8 {
		row, err := s.Store.Get(ctx, JobBucket(claimed.Project, claimed.Suite), claimed.ID)
		if err != nil {
			return err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return err
		}
		if job.Worker != worker || !activeJob(job.State) {
			return storage.ErrConflict
		}
		now := time.Now().UTC()
		job.State, job.Phase, job.Error = "failed", "finished", failure.Error()
		job.FinishedAt = &now
		job.Log, job.LogTruncated = result.Log, result.LogTruncated
		job.ImageID, job.Runtime = result.ImageID, result.Runtime
		if job.CancelRequested {
			job.State, job.Error = "cancelled", ""
		}
		job.CleanupPending = errors.Is(failure, preparation.ErrCleanupIncomplete) || job.Bundle != ""
		changes := []storage.Change{}
		if job.CleanupPending {
			lease := now.Add(5 * time.Second)
			job.LeaseUntil = &lease
			job.Phase = "cleanup-pending"
		} else {
			job.LeaseUntil = nil
			lease, err := s.Store.Get(ctx, leaseBucket(job.Location), job.ID)
			if err != nil {
				return err
			}
			changes = append(changes, storage.Change{Record: lease, Delete: true})
		}
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
		if err != nil {
			return err
		}
		if err := s.Store.Commit(ctx, append(changes, saved)...); errors.Is(err, storage.ErrConflict) {
			continue
		} else {
			return err
		}
	}
	return storage.ErrConflict
}
