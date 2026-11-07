package schedules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		interval := s.PollInterval
		if interval <= 0 {
			interval = time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		cursor, sourceCursor := "", ""
		for {
			if ctx.Err() != nil {
				return
			}
			cursor, _ = s.Tick(ctx, cursor)
			if s.RepositorySources != nil {
				sourceCursor, _ = s.RepositorySources.TickRepositorySchedules(ctx, sourceCursor, s.now())
			}
			if cursor != "" || sourceCursor != "" {
				continue
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
func (s *Service) Tick(ctx context.Context, cursor string) (string, error) {
	page, err := s.Store.List(ctx, "schedule-work", "", cursor, 50)
	if err != nil {
		return "", err
	}
	var problems []error
	for _, row := range page.Items {
		if ctx.Err() != nil {
			return cursor, ctx.Err()
		}
		job, err := storage.Decode[work](row)
		if err == nil {
			err = s.process(ctx, job, row)
		}
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			problems = append(problems, err)
		}
	}
	return page.Cursor, errors.Join(problems...)
}
func (s *Service) process(parent context.Context, job work, index storage.Record) error {
	v, row, err := s.get(parent, job.Project, job.Schedule)
	if err != nil {
		return err
	}
	now := s.now()
	if v.Archived || !v.Enabled || v.NextAt == nil || now.Before(*v.NextAt) || (v.ClaimUntil != nil && now.Before(*v.ClaimUntil)) {
		return nil
	}
	if index.Project != v.Project {
		return errors.New("schedule work ownership does not match its project")
	}
	v.Claim = domain.NewID()
	until := now.Add(30 * time.Second)
	v.ClaimUntil = &until
	claim, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
	if err != nil {
		return err
	}
	if err = s.Store.Commit(parent, claim, storage.Change{Record: index, Assert: true}); err != nil {
		return err
	}
	row = claim.Record
	row.Revision++
	next, coalesced, err := advance(v.Definition, *v.NextAt, now)
	if err != nil {
		return err
	}
	firing := Firing{ID: fmt.Sprintf("%020d.%s", v.Generation, v.NextAt.UTC().Format("20060102T150405.000000000")), Schedule: v.ID, Generation: v.Generation, Owner: v.Owner, DueAt: *v.NextAt, At: now, State: "triggered", Coalesced: coalesced}
	complete := func(run runs.Run, state, message string) ([]storage.Change, error) {
		if v.ClaimUntil == nil || !s.now().Before(*v.ClaimUntil) {
			return nil, storage.ErrConflict
		}
		updated := v
		updated.NextAt = &next
		updated.LastAt = &now
		updated.LastState = state
		updated.LastError = message
		updated.Claim = ""
		updated.ClaimUntil = nil
		if run.ID != "" {
			updated.LastRun = run.ID
		}
		firing.State = state
		firing.Error = message
		firing.Run = run.ID
		c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, updated)
		if err != nil {
			return nil, err
		}
		f, err := storage.Put(Fires(v.Project, v.ID), firing.ID, v.Project, 0, firing)
		if err != nil {
			return nil, err
		}
		audit, err := domain.Audit(v.Owner, v.Project, "schedule.fire", v.ID, nil, firing)
		if err != nil {
			return nil, err
		}
		return []storage.Change{c, f, audit}, nil
	}
	finish := func(state, message string) error {
		changes, err := complete(runs.Run{}, state, message)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
		defer cancel()
		return s.Store.Commit(ctx, changes...)
	}
	if _, err := s.Runs.AuthorizeRequest(parent, v.Owner, v.Project, v.Request, "runs.create"); err != nil {
		if parent.Err() != nil {
			return parent.Err()
		}
		return finish("blocked", err.Error())
	}
	if v.Overlap == "skip" && v.LastRun != "" {
		last, _, err := s.Runs.Get(parent, v.Project, v.LastRun)
		if err != nil {
			return finish("blocked", "Previous run state is unavailable: "+err.Error())
		}
		if !runs.Terminal(last.State) {
			return finish("skipped", "The previous scheduled run remains active.")
		}
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	_, err = s.Runs.CreateScheduled(ctx, v.Owner, v.Project, v.Request, key(v.Project+"/"+v.ID+"/"+firing.ID), runs.ScheduleTrigger{ID: v.ID, Firing: firing.ID, DueAt: firing.DueAt}, func(run runs.Run) ([]storage.Change, error) { return complete(run, "triggered", "") })
	if err == nil || errors.Is(err, storage.ErrConflict) {
		return err
	}
	if parent.Err() != nil {
		return parent.Err()
	}
	return finish("blocked", err.Error())
}
