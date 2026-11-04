package runs

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (s *Service) AuthorizeWorkerAttempt(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
	run, row, err := s.Get(ctx, ref.Project, ref.Run)
	if err != nil {
		return workerstate.Access{}, err
	}
	if row.Project != ref.Project || run.State != "running" || run.CancelRequested || run.Worker != ref.Worker || run.CurrentTest != ref.Test || run.CurrentAttempt != ref.Attempt || run.LeaseUntil == nil || !time.Now().Before(*run.LeaseUntil) || !time.Now().Before(run.Deadline) {
		return workerstate.Access{}, ErrLeaseLost
	}
	entry, executionRow, err := (workerstate.Ledger{Store: s.Store}).Get(ctx, ref.Run, ref.ID)
	if err != nil {
		return workerstate.Access{}, err
	}
	if executionRow.Project != ref.Project || entry.Reference != ref || entry.State != "prepared" || entry.ResultSaved || entry.Claim != "" {
		return workerstate.Access{}, ErrLeaseLost
	}
	if err := s.admitAttempt(ctx, run); err != nil {
		return workerstate.Access{}, err
	}
	projectRow, err := s.Store.Get(ctx, "projects", run.Project)
	if err != nil {
		return workerstate.Access{}, err
	}
	project, err := storage.Decode[projects.Project](projectRow)
	if err != nil || project.Archived {
		return workerstate.Access{}, ErrLeaseLost
	}
	limits := capture.ResolveLimits(project.CaptureLimits)
	return workerstate.Access{Deadline: run.Deadline, ProjectBytes: limits.ProjectBytes, RunBytes: capture.Normalize(run.Capture).MaxBytes, Records: []storage.Record{row, executionRow, projectRow}}, nil
}
