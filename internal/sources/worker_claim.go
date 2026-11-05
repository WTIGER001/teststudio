package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

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
		if s.AssemblyOnly && (job.Bundle == "" || job.AssemblyComplete) {
			return nil
		}
		if job.AssemblyComplete && s.Inspectors[job.Runner] == nil {
			return nil
		}
		if job.Bundle != "" && !job.AssemblyComplete && s.Builders[job.Runner] == nil {
			return nil
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
		if driver == nil && !s.AssemblyOnly || job.CancelRequested || denied {
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
			ref := execution.Reference{ID: domain.NewID(), Run: job.ID, Installation: "preparation-" + installation, Project: job.Project, Test: "catalog", Attempt: 1, Worker: worker}
			if s.AssemblyOnly {
				ref.Profile = job.Runner + "-assembly-v1"
				job.AssemblyReference = &ref
			} else {
				ref = driver.Reference(ref)
			}
			deadline := now.Add(30 * time.Second)
			job.Reference, job.Worker, job.LeaseUntil = &ref, worker, &deadline
			job.State, job.Phase = "discovering", "discovering"
			if job.Bundle != "" && !job.AssemblyComplete {
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

func samePreparationClaim(job, claimed Job, worker string) bool {
	return job.Worker == worker && job.Reference != nil && claimed.Reference != nil && *job.Reference == *claimed.Reference
}
