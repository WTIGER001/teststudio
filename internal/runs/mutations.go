package runs

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) mutate(ctx context.Context, project, id string, change func(*Run) error, extra ...storage.Change) (Run, error) {
	for range 8 {
		v, r, err := s.Get(ctx, project, id)
		if err != nil {
			return v, err
		}
		wasTerminal := Terminal(v.State)
		if err := change(&v); err != nil {
			return v, err
		}
		v.UpdatedAt = time.Now().UTC()
		c, err := storage.Put(r.Bucket, r.ID, project, r.Revision, v)
		if err != nil {
			return v, err
		}
		changes := append([]storage.Change{c}, extra...)
		if !wasTerminal && Terminal(v.State) && s.Completion != nil {
			notifications, err := s.Completion.Complete(ctx, v)
			if err != nil {
				return v, err
			}
			changes = append(changes, notifications...)
		}
		if err := s.Store.Commit(ctx, changes...); errors.Is(err, storage.ErrConflict) {
			continue
		} else {
			return v, err
		}
	}
	return Run{}, storage.ErrConflict
}

func (s *Service) Cancel(ctx context.Context, user, project, id string) (Run, error) {
	if err := s.Auth.Check(ctx, user, project, "run/"+id, "runs.cancel"); err != nil {
		return Run{}, err
	}
	audit, err := domain.Audit(user, project, "run.cancel", id, nil, nil)
	if err != nil {
		return Run{}, err
	}
	return s.mutate(ctx, project, id, func(v *Run) error {
		if Terminal(v.State) {
			return errors.New("run is already finished")
		}
		v.CancelRequested = true
		return nil
	}, audit)
}

func (s *Service) RerunFailures(ctx context.Context, user, project, id, key string) (Run, error) {
	if err := s.Auth.Check(ctx, user, project, "run/"+id, "runs.read"); err != nil {
		return Run{}, err
	}
	parent, _, err := s.Get(ctx, project, id)
	if err != nil {
		return Run{}, err
	}
	if !Terminal(parent.State) {
		return Run{}, errors.New("run is still active")
	}
	ids := []string{}
	for _, result := range parent.Results {
		if result.State == "failed" || result.State == "timed-out" || result.State == "flaky" {
			ids = append(ids, result.TestID)
		}
	}
	if len(ids) == 0 {
		return Run{}, errors.New("run has no failed, timed-out, or flaky tests")
	}
	req := parent.Request
	req.SelectionPreview = ""
	req.IDs = ids
	req.Parent = id
	req.Release = parent.Release.ID
	return s.Create(ctx, user, project, req, key, "rerun")
}
