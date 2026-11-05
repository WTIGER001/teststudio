package sources

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) StartPreparation(ctx context.Context) <-chan struct{} {
	return s.startPreparation(ctx, false, false)
}

func (s *Service) StartBuildPreparation(ctx context.Context) <-chan struct{} {
	return s.startPreparation(ctx, true, true)
}

func (s *Service) StartDiscoveryPreparation(ctx context.Context) <-chan struct{} {
	worker := *s
	worker.AssemblyOnly = false
	worker.Builders, worker.RecoveryBuilders = nil, nil
	return worker.startPreparation(ctx, false, true)
}

func (s *Service) startPreparation(ctx context.Context, builds, publish bool) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker := domain.NewID()
		recoveryCursor := ""
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			nextRecovery, recoveryErr := s.recoverPreparationPage(ctx, worker, recoveryCursor)
			if recoveryErr != nil {
				recoveryCursor = ""
				if ctx.Err() == nil {
					slog.Error("preparation recovery failed")
				}
			} else {
				recoveryCursor = nextRecovery
			}

			var err error
			if publish {
				if _, publishErr := s.ProcessRepositoryPublication(ctx); publishErr != nil && ctx.Err() == nil {
					slog.Error("repository publication could not persist its outcome")
				}
			}
			if builds {
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
	if s.AssemblyOnly {
		for _, builder := range s.Builders {
			if _, ok := builder.(preparation.AssemblyBuilder); !ok {
				return false, errAssemblyBuilder
			}
		}
	}
	return s.processQueue(ctx, worker, buildQueueBucket(s.Location))
}

func (s *Service) processQueue(ctx context.Context, worker, bucket string) (bool, error) {
	cursor := ""
	for {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		page, err := s.Store.List(ctx, bucket, "", cursor, 25)
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
			defer stop()
			if errors.Is(inspectErr, errAssemblyHandoff) {
				err = s.completeAssembly(finishContext, job, worker)
				if err != nil {
					err = s.failPreparation(finishContext, job, worker, result, err)
				}
			} else {
				err = s.finishPreparation(finishContext, job, worker, result, inspectErr)
			}
			return true, err
		}
		if page.Cursor == "" {
			return false, nil
		}
		cursor = page.Cursor
	}
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
			if err != nil || !samePreparationClaim(job, claimed, worker) || job.CancelRequested || !activeJob(job.State) {
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
		if !samePreparationClaim(job, claimed, worker) || job.State != "discovering" {
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
		if job.State == "failed" && requiresBuildCleanup(job) || s.deferredInspection(job) {
			job.CleanupPending = true
			retry := now.Add(5 * time.Second)
			job.LeaseUntil = &retry
			job.Phase = "cleanup-pending"
			saved, err = storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
			if err != nil {
				return err
			}
			queued, err := bound.queueRepositoryPublication(ctx, job)
			if err != nil {
				return err
			}
			return bound.Store.Commit(ctx, append(queued, saved)...)
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
		if !samePreparationClaim(job, claimed, worker) || !activeJob(job.State) {
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
		job.CleanupPending = errors.Is(failure, preparation.ErrCleanupIncomplete) || requiresBuildCleanup(job) || s.deferredInspection(job)
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
