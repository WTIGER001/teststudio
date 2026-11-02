package runs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	concurrency := s.Concurrency
	if concurrency < 1 || concurrency > 32 {
		concurrency = 2
	}
	var workers sync.WaitGroup
	for range concurrency {
		workers.Go(func() { s.loop(ctx, domain.NewID()) })
	}
	go func() { workers.Wait(); close(done) }()
	return done
}

func (s *Service) loop(ctx context.Context, worker string) {
	interval := s.PollInterval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		s.recoverExpired(ctx)
		queued, err := s.Store.List(ctx, "run-queue", "", "", 25)
		if err == nil {
			for _, entry := range queued.Items {
				if ctx.Err() != nil {
					return
				}
				v, claimed, err := s.claim(ctx, entry, worker)
				if err == nil && claimed {
					s.execute(ctx, v, worker)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) claim(ctx context.Context, entry storage.Record, worker string) (Run, bool, error) {
	v, r, err := s.Get(ctx, entry.Project, entry.ID)
	if err != nil {
		return v, false, err
	}
	if v.State != "requested" {
		return v, false, errors.New("run is not queued")
	}
	now := time.Now().UTC()
	changes := []storage.Change{{Record: entry, Delete: true}}
	if v.CancelRequested || !now.Before(v.Deadline) {
		v.State = "cancelled"
		if !v.CancelRequested {
			v.State = "timed-out"
			v.Error = "Run deadline expired before execution."
		}
		v.Phase = "finished"
		v.FinishedAt = &now
		v.EvidenceState = "complete"
	} else {
		v.State = "running"
		v.Phase = "execution"
		v.Worker = worker
		v.StartedAt = &now
		lease := now.Add(30 * time.Second)
		v.LeaseUntil = &lease
		index, err := storage.Put("run-leases", v.ID, v.Project, 0, v.ID)
		if err != nil {
			return v, false, err
		}
		changes = append(changes, index)
	}
	v.UpdatedAt = now
	change, err := storage.Put(r.Bucket, r.ID, v.Project, r.Revision, v)
	if err != nil {
		return v, false, err
	}
	changes = append(changes, change)
	err = s.Store.Commit(ctx, changes...)
	return v, err == nil && v.State == "running", err
}

func (s *Service) recoverExpired(ctx context.Context) {
	entries, err := s.Store.All(ctx, "run-leases", "")
	if err != nil {
		return
	}
	for _, entry := range entries {
		_, _ = s.mutate(ctx, entry.Project, entry.ID, func(v *Run) error {
			if v.State != "running" || v.LeaseUntil == nil || time.Now().Before(*v.LeaseUntil) {
				return ErrLeaseLost
			}
			now := time.Now().UTC()
			v.State = "incomplete"
			v.Outcome = "unknown"
			v.Phase = "finished"
			v.FinishedAt = &now
			v.Error = "Execution worker stopped reporting before completion."
			v.EvidenceState = "incomplete"
			return nil
		}, storage.Change{Record: entry, Delete: true})
	}
}

func owned(v *Run, worker string) error {
	if v.State != "running" || v.Worker != worker || v.LeaseUntil == nil || !time.Now().Before(*v.LeaseUntil) {
		return ErrLeaseLost
	}
	return nil
}

func (s *Service) heartbeat(ctx context.Context, run Run, worker string, cancel context.CancelFunc, done chan<- struct{}) {
	defer close(done)
	interval := s.HeartbeatInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := s.mutate(ctx, run.Project, run.ID, func(v *Run) error {
				if err := owned(v, worker); err != nil {
					return err
				}
				if v.CancelRequested {
					cancel()
					return nil
				}
				until := time.Now().UTC().Add(30 * time.Second)
				v.LeaseUntil = &until
				return nil
			})
			if err != nil {
				cancel()
				return
			}
		}
	}
}

func (s *Service) execute(parent context.Context, run Run, worker string) {
	ctx, cancel := context.WithDeadline(parent, run.Deadline)
	heartbeat := make(chan struct{})
	go s.heartbeat(ctx, run, worker, cancel, heartbeat)
	evidenceErrors := []string{}
	sequence := int64(0)
	capture := func(kind, test string, attempt int, message string) {
		captureCtx, releaseCapture := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
		defer releaseCapture()
		if s.Evidence == nil {
			if len(evidenceErrors) < 20 {
				evidenceErrors = append(evidenceErrors, "Evidence storage is not configured.")
			}
			return
		}
		if err := s.Evidence.Event(captureCtx, evidence.Event{Run: run.ID, Project: run.Project, Kind: kind, Test: test, Attempt: attempt, Message: message}); err != nil && len(evidenceErrors) < 20 {
			evidenceErrors = append(evidenceErrors, err.Error())
		}
		if run.Capture.Logs {
			sequence++
			_, err := s.Evidence.Log(captureCtx, run.Project, run.ID, worker, "stdout", test, attempt, sequence, time.Now().UTC().Format(time.RFC3339Nano)+" "+message+"\n", run.Capture.MaxBytes)
			if err != nil && len(evidenceErrors) < 20 {
				evidenceErrors = append(evidenceErrors, err.Error())
			}
		}
	}
	capture("run.started", "", 0, "Run started.")
	completed := map[string]string{}
	executionError := ""
	for _, test := range run.Tests {
		current, _, err := s.Get(ctx, run.Project, run.ID)
		if err != nil {
			executionError = err.Error()
			break
		}
		if current.CancelRequested || ctx.Err() != nil {
			break
		}
		if err := s.Auth.Check(ctx, run.Actor, run.Project, "", "runs.create"); err != nil {
			executionError = "Execution permission was revoked."
			break
		}
		protected := run.Environment != nil && run.Environment.Protected
		if protected {
			if err := s.Auth.Check(ctx, run.Actor, run.Project, "environment/"+run.Request.Environment, "runs.execute-protected"); err != nil {
				executionError = "Protected execution permission was revoked."
				break
			}
		}
		if err := s.Catalog.CheckCurrentState(ctx, run.Project, run.Request.Suite, run.Tests, protected); err != nil {
			executionError = err.Error()
			break
		}
		skip := false
		for _, id := range test.Dependencies {
			if completed[id] != "passed" && completed[id] != "flaky" {
				skip = true
			}
		}
		if skip {
			now := time.Now().UTC()
			result := execution.Result{TestID: test.ID, State: "skipped", Error: "A dependency did not pass.", StartedAt: now, FinishedAt: now}
			if err := s.finalResult(ctx, run, worker, result); err != nil {
				executionError = err.Error()
				break
			}
			completed[test.ID] = "skipped"
			capture("test.skipped", test.ID, 0, "Test skipped because a dependency did not pass.")
			continue
		}
		var result execution.Result
		failedBefore := false
		for attempt := 1; attempt <= run.Request.Retries+1; attempt++ {
			if ctx.Err() != nil {
				break
			}
			if err := s.admitAttempt(ctx, run); err != nil {
				executionError = err.Error()
				break
			}
			_, err := s.mutate(ctx, run.Project, run.ID, func(v *Run) error {
				if err := owned(v, worker); err != nil {
					return err
				}
				v.CurrentTest = test.ID
				return nil
			})
			if err != nil {
				executionError = err.Error()
				break
			}
			capture("test.started", test.ID, attempt, fmt.Sprintf("Starting %s, attempt %d.", test.ID, attempt))
			if run.Environment == nil || s.HTTP == nil {
				executionError = "No compatible executor is configured."
				break
			}
			result = s.HTTP.Execute(ctx, test, *run.Environment)
			result.Attempt = attempt
			recordCtx, releaseRecord := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
			recordErr := s.attemptResult(recordCtx, run, worker, result)
			releaseRecord()
			if recordErr != nil {
				executionError = recordErr.Error()
				break
			}
			capture("test.finished", test.ID, attempt, fmt.Sprintf("%s: %s.", test.ID, result.State))
			if result.State == "passed" {
				if failedBefore {
					result.State = "flaky"
				}
				break
			}
			failedBefore = true
		}
		if executionError != "" || result.TestID == "" || ctx.Err() != nil {
			break
		}
		if err := s.finalResult(ctx, run, worker, result); err != nil {
			executionError = err.Error()
			break
		}
		completed[test.ID] = result.State
	}
	ctxErr := ctx.Err()
	capture("run.finished", "", 0, "Execution finished.")
	cancel()
	<-heartbeat
	finish, cancelFinish := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancelFinish()
	lease, err := s.Store.Get(finish, "run-leases", run.ID)
	if err != nil {
		return
	}
	_, _ = s.mutate(finish, run.Project, run.ID, func(v *Run) error {
		if err := owned(v, worker); err != nil {
			return err
		}
		now := time.Now().UTC()
		v.FinishedAt = &now
		v.Phase = "finished"
		v.CurrentTest = ""
		v.EvidenceErrors = evidenceErrors
		v.EvidenceState = "complete"
		if len(evidenceErrors) > 0 {
			v.EvidenceState = "incomplete"
		}
		v.State = "completed"
		v.Outcome = "passed"
		if v.Counts.Failed > 0 {
			v.Outcome = "failed"
		} else if v.Counts.Flaky > 0 {
			v.Outcome = "flaky"
		}
		if v.CancelRequested {
			v.State = "cancelled"
			v.Outcome = "unknown"
		} else if errors.Is(ctxErr, context.DeadlineExceeded) {
			v.State = "timed-out"
			v.Outcome = "unknown"
		} else if parent.Err() != nil {
			v.State = "incomplete"
			v.Outcome = "unknown"
			v.Error = "Service stopped during execution."
		} else if executionError != "" {
			v.State = "blocked"
			v.Outcome = "unknown"
			v.Error = executionError
		} else if v.Counts.Completed != v.Counts.Total || v.Counts.Passed+v.Counts.Failed+v.Counts.Flaky == 0 {
			v.State = "incomplete"
			v.Outcome = "unknown"
			v.Error = "Selected checks did not produce complete results."
		}
		return nil
	}, storage.Change{Record: lease, Delete: true})
}

func (s *Service) attemptResult(ctx context.Context, run Run, worker string, result execution.Result) error {
	id := fmt.Sprintf("%s.%04d", result.TestID, result.Attempt)
	change, err := storage.Put("attempts/"+run.ID, id, run.Project, 0, result)
	if err != nil {
		return err
	}
	_, err = s.mutate(ctx, run.Project, run.ID, func(v *Run) error { return owned(v, worker) }, change)
	return err
}

func (s *Service) finalResult(ctx context.Context, run Run, worker string, result execution.Result) error {
	_, err := s.mutate(ctx, run.Project, run.ID, func(v *Run) error {
		if err := owned(v, worker); err != nil {
			return err
		}
		for _, old := range v.Results {
			if old.TestID == result.TestID {
				return errors.New("test result already exists")
			}
		}
		v.Results = append(v.Results, result)
		v.Counts = Count(len(v.Tests), v.Results)
		return nil
	})
	return err
}

func (s *Service) admitAttempt(ctx context.Context, run Run) error {
	current, _, err := s.Get(ctx, run.Project, run.ID)
	if err != nil {
		return err
	}
	if current.CancelRequested {
		return errors.New("run cancellation was requested")
	}
	if err := s.Auth.Check(ctx, run.Actor, run.Project, "", "runs.create"); err != nil {
		return errors.New("execution permission was revoked")
	}
	protected := run.Environment != nil && run.Environment.Protected
	if protected {
		if err := s.Auth.Check(ctx, run.Actor, run.Project, "environment/"+run.Request.Environment, "runs.execute-protected"); err != nil {
			return errors.New("protected execution permission was revoked")
		}
	}
	return s.Catalog.CheckCurrentState(ctx, run.Project, run.Request.Suite, run.Tests, protected)
}
