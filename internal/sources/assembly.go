package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/storage"
)

var errAssemblyHandoff = errors.New("source assembly awaits isolated discovery")
var errAssemblyBuilder = errors.New("assembly-only preparation requires a builder that preserves its published artifact during handoff")

func assemblyJob(job Job) bool          { return job.AssemblyReference != nil && !job.AssemblyComplete }
func requiresBuildCleanup(job Job) bool { return job.Bundle != "" && !job.AssemblyComplete }
func confirmedAssembly(job Job) bool {
	return assemblyJob(job) && job.Reference != nil && *job.Reference == *job.AssemblyReference && job.Build != nil && immutableImage.MatchString(job.Image) && job.Image == job.Build.Image && job.SourceDigest == job.Build.SourceDigest
}

func (s *Service) completeAssembly(ctx context.Context, claimed Job, worker string) error {
	builder, ok := s.Builders[claimed.Runner].(preparation.AssemblyBuilder)
	if !ok {
		builder, ok = s.RecoveryBuilders[claimed.Runner].(preparation.AssemblyBuilder)
	}
	if !ok || claimed.Reference == nil || claimed.AssemblyReference == nil || *claimed.Reference != *claimed.AssemblyReference {
		return errAssemblyBuilder
	}
	row, err := s.Store.Get(ctx, JobBucket(claimed.Project, claimed.Suite), claimed.ID)
	if err != nil {
		return err
	}
	current, err := storage.Decode[Job](row)
	if err != nil {
		return err
	}
	if current.AssemblyComplete && current.AssemblyReference != nil && *current.AssemblyReference == *claimed.AssemblyReference {
		return nil
	}
	if !confirmedAssembly(current) || *current.AssemblyReference != *claimed.AssemblyReference || current.Worker != worker || current.State != "discovering" {
		return storage.ErrConflict
	}
	if err := builder.CompleteAssembly(ctx, *current.AssemblyReference); err != nil {
		return errors.Join(err, preparation.ErrCleanupIncomplete)
	}
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		row, err := bound.Store.Get(ctx, row.Bucket, row.ID)
		if err != nil {
			return err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return err
		}
		if job.AssemblyComplete && job.AssemblyReference != nil && *job.AssemblyReference == *claimed.AssemblyReference {
			return nil
		}
		if !confirmedAssembly(job) || *job.AssemblyReference != *claimed.AssemblyReference || job.Worker != worker || job.State != "discovering" || job.LeaseUntil == nil || !time.Now().Before(*job.LeaseUntil) {
			return storage.ErrConflict
		}
		if job.CancelRequested {
			return context.Canceled
		}
		for _, permission := range []string{"sources.manage", "catalog.read", "catalog.import"} {
			if err := auth.Check(ctx, job.Actor, job.Project, "suite/"+job.Suite, permission); err != nil {
				return err
			}
		}
		if err := bound.checkRepositoryJob(ctx, job, false); err != nil {
			return err
		}
		if _, err := bound.loadBundle(ctx, job); err != nil {
			return err
		}
		lease, err := bound.Store.Get(ctx, leaseBucket(job.Location), job.ID)
		if err != nil {
			return err
		}
		job.AssemblyComplete = true
		job.State, job.Phase = "received", "awaiting-discovery"
		job.Reference, job.LeaseUntil = nil, nil
		job.Worker = ""
		job.CleanupPending = false
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
		if err != nil {
			return err
		}
		queued, err := storage.Put(queueBucket(job.Location), queueID(job), job.Project, 0, jobIndex{Job: job.ID, Suite: job.Suite, Location: job.Location})
		if err != nil {
			return err
		}
		audit, err := domain.Audit(job.Actor, job.Project, "sources.prepare.assembled", job.Suite+"/"+job.ID, nil, map[string]string{"image": job.Image, "sourceDigest": job.SourceDigest})
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, saved, queued, audit, storage.Change{Record: lease, Delete: true})
	})
}
