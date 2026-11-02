package evidence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RetentionJob struct {
	Project     string    `json:"projectId"`
	Run         string    `json:"runId"`
	Cursor      string    `json:"cursor,omitempty"`
	NextAttempt time.Time `json:"nextAttempt"`
	Attempts    int       `json:"attempts"`
}

type RetentionStatus struct {
	State        string    `json:"state"`
	Error        string    `json:"error,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
	IndexVersion int       `json:"indexVersion"`
}

type runOutcome struct {
	Project    string     `json:"projectId"`
	Outcome    string     `json:"outcome"`
	FinishedAt *time.Time `json:"finishedAt"`
	Results    []struct {
		Test  string `json:"testId"`
		State string `json:"state"`
	} `json:"results"`
}

func (s *Service) ProcessRetention(ctx context.Context, limit int) error {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()
	page, err := s.Store.List(ctx, "evidence-retention-jobs", "", s.retentionCursor, limit)
	if err != nil {
		return err
	}
	s.retentionCursor = page.Cursor
	var failures []error
	for _, row := range page.Items {
		job, err := storage.Decode[RetentionJob](row)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if time.Now().Before(job.NextAttempt) {
			continue
		}
		if err := s.retentionPage(ctx, row, job); err != nil && !errors.Is(err, storage.ErrConflict) {
			failures = append(failures, err)
			job.Attempts++
			job.NextAttempt = time.Now().UTC().Add(time.Minute)
			change, encodeErr := storage.Put(row.Bucket, row.ID, job.Project, row.Revision, job)
			status, statusErr := s.retentionStatusChange(ctx, job.Project, job.Run, "failed", "Retention processing failed. A retry is scheduled.")
			if encodeErr == nil && statusErr == nil {
				_ = s.Store.Commit(ctx, change, status)
			}
		}
	}
	return errors.Join(failures...)
}

func (s *Service) retentionPage(ctx context.Context, jobRow storage.Record, job RetentionJob) error {
	runRow, err := s.Store.Get(ctx, "runs", job.Run)
	if err != nil {
		return err
	}
	run, err := storage.Decode[runOutcome](runRow)
	if err != nil {
		return err
	}
	if run.Project != job.Project || run.FinishedAt == nil {
		return errors.New("retention requires a finished run in the assigned project")
	}
	page, err := s.Store.List(ctx, Bucket(job.Run), job.Project, job.Cursor, 20)
	if err != nil {
		return err
	}
	changes := []storage.Change{{Record: runRow, Assert: true}}
	legacySizes := map[string]int64{}
	for _, row := range page.Items {
		a, err := storage.Decode[Artifact](row)
		if err != nil {
			return err
		}
		if a.State == "expired" || a.State == "deleted" {
			continue
		}
		changed := false
		if a.QuotaVersion < 1 {
			legacySizes[a.Test] += a.Size
			a.QuotaVersion = 1
			changed = true
		}
		outcome := run.Outcome
		if a.Test != "" {
			for _, result := range run.Results {
				if result.Test == a.Test {
					outcome = result.State
					break
				}
			}
			if a.Attempt > 0 && outcome != "flaky" {
				attemptRow, err := s.Store.Get(ctx, "attempts/"+job.Run, fmt.Sprintf("%s.%04d", a.Test, a.Attempt))
				if err == nil {
					attempt, err := storage.Decode[struct {
						State string `json:"state"`
					}](attemptRow)
					if err != nil {
						return err
					}
					outcome = attempt.State
				} else if !errors.Is(err, storage.ErrNotFound) {
					return err
				}
			}
		}
		if a.Policy != nil {
			expires := capture.Expiry(a.CreatedAt, *a.Policy, a.Kind, outcome)
			if !expires.Equal(a.ExpiresAt) {
				a.ExpiresAt = expires
				changed = true
			}
		}
		if changed {
			change, err := storage.Put(row.Bucket, row.ID, job.Project, row.Revision, a)
			if err != nil {
				return err
			}
			changes = append(changes, change)
		}
		index, err := s.expirationChange(ctx, a)
		if err != nil {
			return err
		}
		changes = append(changes, index)
	}
	legacy, err := s.legacyQuotaChanges(ctx, job.Project, job.Run, legacySizes)
	if err != nil {
		return err
	}
	changes = append(changes, legacy...)
	state := "pending"
	if page.Cursor == "" {
		changes = append(changes, storage.Change{Record: jobRow, Delete: true})
		state = "ready"
	} else {
		job.Cursor = page.Cursor
		job.NextAttempt = time.Time{}
		change, err := storage.Put(jobRow.Bucket, jobRow.ID, job.Project, jobRow.Revision, job)
		if err != nil {
			return err
		}
		changes = append(changes, change)
	}
	status, err := s.retentionStatusChange(ctx, job.Project, job.Run, state, "")
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, append(changes, status)...)
}

func (s *Service) retentionStatusChange(ctx context.Context, project, run, state, message string) (storage.Change, error) {
	r, err := s.Store.Get(ctx, "evidence-retention-status", run)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Change{}, err
	}
	return storage.Put("evidence-retention-status", run, project, r.Revision, RetentionStatus{State: state, Error: message, UpdatedAt: time.Now().UTC(), IndexVersion: 1})
}

func (s *Service) StartMaintenance(ctx context.Context, interval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	if interval <= 0 {
		interval = 5 * time.Second
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			_ = s.ProcessRetention(ctx, 10)
			_ = s.EnsureRetentionIndex(ctx, 10)
			_ = s.Sweep(ctx, 30)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}
