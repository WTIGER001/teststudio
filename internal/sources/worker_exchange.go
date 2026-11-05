package sources

import (
	"context"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (s *Service) discoveryJob(ctx context.Context, ref execution.Reference) (Job, storage.Record, error) {
	if ref.Test != "catalog" || ref.Attempt != 1 || !strings.HasPrefix(ref.Installation, "preparation-") || !strings.HasSuffix(ref.Profile, "-discovery-v1") {
		return Job{}, storage.Record{}, storage.ErrNotFound
	}
	index, err := s.Store.Get(ctx, leaseBucket(s.Location), ref.Run)
	if err != nil || index.Project != ref.Project {
		return Job{}, storage.Record{}, storage.ErrNotFound
	}
	item, err := storage.Decode[jobIndex](index)
	if err != nil || item.Job != ref.Run || item.Location != s.Location {
		return Job{}, storage.Record{}, storage.ErrNotFound
	}
	row, err := s.Store.Get(ctx, JobBucket(ref.Project, item.Suite), ref.Run)
	if err != nil {
		return Job{}, row, err
	}
	job, err := storage.Decode[Job](row)
	if err != nil || row.Project != ref.Project || job.Project != ref.Project || job.ID != ref.Run || job.Reference == nil || *job.Reference != ref {
		return Job{}, row, storage.ErrNotFound
	}
	return job, row, nil
}

func (s *Service) AuthorizeDiscovery(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
	job, row, err := s.discoveryJob(ctx, ref)
	if err != nil {
		return workerstate.Access{}, err
	}
	if job.State != "discovering" || job.Phase == "recovering" || job.Worker != ref.Worker || job.CancelRequested || job.LeaseUntil == nil || !time.Now().Before(*job.LeaseUntil) || job.StartedAt == nil {
		return workerstate.Access{}, preparation.ErrCleanupIncomplete
	}
	for _, permission := range []string{"sources.manage", "catalog.read", "catalog.import"} {
		if err := s.Auth.Check(ctx, job.Actor, job.Project, "suite/"+job.Suite, permission); err != nil {
			return workerstate.Access{}, err
		}
	}
	if err := s.checkRepositoryJob(ctx, job, false); err != nil {
		return workerstate.Access{}, err
	}
	projectRow, err := s.Store.Get(ctx, "projects", job.Project)
	if err != nil {
		return workerstate.Access{}, err
	}
	project, err := storage.Decode[projects.Project](projectRow)
	if err != nil || project.Archived {
		return workerstate.Access{}, storage.ErrConflict
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", job.Project), job.Suite)
	if err != nil {
		return workerstate.Access{}, err
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil || suite.Archived {
		return workerstate.Access{}, storage.ErrConflict
	}
	deadline := job.StartedAt.Add(5 * time.Minute)
	if !time.Now().Before(deadline) {
		return workerstate.Access{}, context.DeadlineExceeded
	}
	return workerstate.Access{Deadline: deadline, Records: []storage.Record{row, projectRow, suiteRow}}, nil
}

func (s *Service) AuthorizeDiscoveryCleanup(ctx context.Context, ref execution.Reference) error {
	job, _, err := s.discoveryJob(ctx, ref)
	if err != nil {
		return err
	}
	if job.Phase == "recovering" && job.Worker != ref.Worker && job.LeaseUntil != nil && time.Now().Before(*job.LeaseUntil) {
		return nil
	}
	if activeJob(job.State) || !job.CleanupPending {
		return storage.ErrConflict
	}
	return nil
}

func (s *Service) deferredInspection(job Job) bool {
	driver := s.Inspectors[job.Runner]
	if driver == nil {
		driver = s.RecoveryInspectors[job.Runner]
	}
	deferred, ok := driver.(preparation.DeferredInspector)
	return ok && deferred.DeferredCleanup()
}
