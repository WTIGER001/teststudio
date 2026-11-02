package runs

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (s *Service) managedDriver() execution.ManagedExecutor {
	if s.RecoveryDriver != nil {
		return s.RecoveryDriver
	}
	driver, _ := s.Playwright.(execution.ManagedExecutor)
	return driver
}
func (s *Service) beginExecution(ctx context.Context, run Run, worker string, test catalog.Test, attempt int, driver execution.ManagedExecutor) (execution.Reference, error) {
	ledger := workerstate.Ledger{Store: s.Store}
	installation, err := ledger.Installation(ctx, true)
	if err != nil {
		return execution.Reference{}, err
	}
	ref := driver.Reference(execution.Reference{ID: fingerprint([]any{installation, run.ID, test.ID, attempt}), Installation: installation, Project: run.Project, Run: run.ID, Test: test.ID, Attempt: attempt, Worker: worker})
	_, changes, err := ledger.Begin(ref)
	if err != nil {
		return ref, err
	}
	_, err = s.mutate(ctx, run.Project, run.ID, func(v *Run) error { return owned(v, worker) }, changes...)
	return ref, err
}
func (s *Service) checkpointExecution(ctx context.Context, run Run, worker string, ref execution.Reference, result execution.Result, problems []string, abandoned bool) error {
	ledger := workerstate.Ledger{Store: s.Store}
	for range 8 {
		v, row, err := ledger.Get(ctx, run.ID, ref.ID)
		if err != nil {
			return err
		}
		if v.Claim != "" {
			return ErrLeaseLost
		}
		if v.ResultSaved {
			return nil
		}
		v.EvidenceAbandoned = abandoned && len(problems) > 0
		change, err := ledger.Checkpoint(row, v, problems, false)
		if err != nil {
			return err
		}
		err = s.attemptResult(ctx, run, worker, result, change)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		return err
	}
	return storage.ErrConflict
}
