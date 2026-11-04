package runs

import (
	"context"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
	"reflect"
	"time"
)

func (s *Service) reconciliationLoop(ctx context.Context) {
	interval := s.RecoveryInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	cursor := ""
	inventoryAt := time.Time{}
	for {
		if ctx.Err() != nil {
			return
		}
		if !time.Now().Before(inventoryAt) {
			_ = s.discoverExecutions(ctx)
			inventoryAt = time.Now().Add(30 * time.Second)
		}
		cursor, _ = s.reconcilePage(ctx, cursor)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) discoverExecutions(ctx context.Context) error {
	ledger := workerstate.Ledger{Store: s.Store}
	installation, err := ledger.Installation(ctx, false)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var problems []error
	for _, driver := range s.managedDrivers() {
		problems = append(problems, s.discoverDriver(ctx, installation, driver))
	}
	return errors.Join(problems...)
}

func (s *Service) discoverDriver(ctx context.Context, installation string, driver execution.ManagedExecutor) error {
	ledger := workerstate.Ledger{Store: s.Store}
	scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	refs, err := driver.Inventory(scanCtx, installation)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		run, runRow, err := s.Get(ctx, ref.Project, ref.Run)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if !Terminal(run.State) || run.Worker != ref.Worker {
			continue
		}
		valid := false
		for _, test := range run.Tests {
			if test.ID == ref.Test && (test.Runner == "playwright" || test.Runner == "go") {
				valid = true
			}
		}
		if !valid || ref.Profile != driver.Profile() || ref.Installation != installation {
			continue
		}
		v, row, err := ledger.Get(ctx, ref.Run, ref.ID)
		if errors.Is(err, storage.ErrNotFound) {
			_, changes, err := ledger.Begin(ref)
			if err != nil {
				return err
			}
			changes = append(changes, storage.Change{Record: runRow, Assert: true})
			if err := s.Store.Commit(ctx, changes...); err != nil && !errors.Is(err, storage.ErrConflict) {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(v.Reference, ref) {
			continue
		}
		if _, err := s.Store.Get(ctx, "execution-work", ref.ID); err == nil {
			continue
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		v.State = "cleanup-pending"
		v.ResultSaved = false
		v.EvidenceComplete = false
		v.EvidenceAbandoned = false
		v.NextCheck = time.Now().UTC()
		v.UpdatedAt = v.NextCheck
		c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
		if err != nil {
			return err
		}
		job, err := storage.Put("execution-work", ref.ID, ref.Project, 0, workerstate.Job{Run: ref.Run, Execution: ref.ID})
		if err != nil {
			return err
		}
		if err := s.Store.Commit(ctx, c, job, storage.Change{Record: runRow, Assert: true}); err != nil && !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return nil
}

func (s *Service) reconcilePage(ctx context.Context, cursor string) (string, error) {
	s.recoverExpired(ctx)
	page, err := s.Store.List(ctx, "execution-work", "", cursor, 20)
	if err != nil {
		return cursor, err
	}
	for _, row := range page.Items {
		if ctx.Err() != nil {
			return cursor, ctx.Err()
		}
		if err := s.reconcileExecution(ctx, row); err != nil && !errors.Is(err, storage.ErrConflict) {
			continue
		}
	}
	return page.Cursor, nil
}

func (s *Service) reconcileExecution(parent context.Context, jobRow storage.Record) error {
	job, err := storage.Decode[workerstate.Job](jobRow)
	if err != nil {
		return err
	}
	ledger := workerstate.Ledger{Store: s.Store}
	installation, err := ledger.Installation(parent, false)
	if err != nil {
		return err
	}
	record, row, err := ledger.Get(parent, job.Run, job.Execution)
	if err != nil {
		return err
	}
	ref := record.Reference
	driver := s.managedDriver(ref.Profile)
	if driver == nil {
		return nil
	}
	if ref.Installation != installation || ref.Project != jobRow.Project || ref.Profile != driver.Profile() {
		return errors.New("execution recovery ownership does not match this installation")
	}
	now := time.Now().UTC()
	if now.Before(record.NextCheck) || (record.ClaimUntil != nil && now.Before(*record.ClaimUntil)) {
		return nil
	}
	run, runRow, err := s.Get(parent, ref.Project, ref.Run)
	if err != nil {
		return err
	}
	if !Terminal(run.State) || run.Worker != ref.Worker {
		return nil
	}
	record.State = "recovering"
	record.ReconcileAttempts++
	record.Claim = domain.NewID()
	until := now.Add(time.Minute)
	record.ClaimUntil = &until
	record.UpdatedAt = now
	claimed, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
	if err != nil {
		return err
	}
	if err := s.Store.Commit(parent, claimed, storage.Change{Record: runRow, Assert: true}, storage.Change{Record: jobRow, Assert: true}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	var recoveryErr error
	if !record.ResultSaved || (!record.EvidenceComplete && !record.EvidenceAbandoned && record.ReconcileAttempts <= 3) {
		input, err := recoveryInput(run, ref)
		if err != nil {
			recoveryErr = err
		} else {
			report := driver.Recover(ctx, ref, input)
			if report.RecoveryError != nil {
				recoveryErr = report.RecoveryError
			} else {
				result := assessResult(run, report.Result)
				result.ExecutionID = ref.ID
				result.Recovered = true
				result.Attempt = ref.Attempt
				problems := []string{}
				retryable := s.captureCodedReport(ctx, run, ref.Worker, result, true, report, &problems)
				recoveryErr = s.checkpointRecovered(ctx, run, ref, record.Claim, result, problems, !retryable)
				if recoveryErr == nil && len(problems) > 0 && retryable && record.ReconcileAttempts < 3 {
					recoveryErr = errors.New("execution evidence remains incomplete")
				}
			}
		}
	}
	if recoveryErr == nil {
		recoveryErr = driver.Cleanup(ctx, ref)
	}
	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancelFinish()
	return ledger.Finish(finishCtx, ref.Run, ref.ID, record.Claim, recoveryErr)
}

func recoveryInput(run Run, ref execution.Reference) (execution.CodedInput, error) {
	for _, test := range run.Tests {
		if test.ID == ref.Test {
			input := execution.CodedInput{Reference: &ref, RunID: run.ID, Test: test, Artifact: run.Release.Draft.Artifact, Capture: run.TestCapture[test.ID], TimeoutMs: test.TimeoutSeconds * 1000, FailedBefore: ref.Attempt > 1}
			if p, ok := run.Timing.Tests[test.ID]; ok {
				input.TimeoutMs = p.EffectiveTimeoutMs
				input.SlowThresholdMs = p.ThresholdMs
				input.TimingMode = p.Policy.Mode
			}
			return input, nil
		}
	}
	return execution.CodedInput{}, fmt.Errorf("execution test %s is absent from the immutable run", ref.Test)
}

func (s *Service) checkpointRecovered(ctx context.Context, run Run, ref execution.Reference, claim string, result execution.Result, problems []string, abandoned bool) error {
	ledger := workerstate.Ledger{Store: s.Store}
	for range 8 {
		record, row, err := ledger.Get(ctx, ref.Run, ref.ID)
		if err != nil {
			return err
		}
		if record.Claim != claim || record.ClaimUntil == nil || !time.Now().Before(*record.ClaimUntil) {
			return ErrLeaseLost
		}
		current, runRow, err := s.Get(ctx, ref.Project, ref.Run)
		if err != nil {
			return err
		}
		if !Terminal(current.State) {
			return ErrLeaseLost
		}
		if record.ResultSaved && record.EvidenceComplete {
			return nil
		}
		record.RecoveredResult = &result
		record.EvidenceAbandoned = abandoned && len(problems) > 0
		change, err := ledger.Checkpoint(row, record, problems, true)
		if err != nil {
			return err
		}
		changes := []storage.Change{change, {Record: runRow, Assert: true}}
		id := fmt.Sprintf("%s.%04d", ref.Test, ref.Attempt)
		if _, err := s.Store.Get(ctx, "attempts/"+run.ID, id); errors.Is(err, storage.ErrNotFound) {
			attempt, err := storage.Put("attempts/"+run.ID, id, run.Project, 0, result)
			if err != nil {
				return err
			}
			changes = append(changes, attempt)
		} else if err != nil {
			return err
		}
		err = s.Store.Commit(ctx, changes...)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		return err
	}
	return storage.ErrConflict
}
