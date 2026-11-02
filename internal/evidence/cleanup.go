package evidence

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Sweep(ctx context.Context, limit int) error {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()
	page, err := s.Store.List(ctx, "artifact-expirations", "", s.cleanupCursor, limit)
	if err != nil {
		return err
	}
	s.cleanupCursor = page.Cursor
	var failures []error
	for _, row := range page.Items {
		e, err := storage.Decode[Expiration](row)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		now := time.Now().UTC()
		if now.Before(e.NextAttempt) || (e.LeaseUntil != nil && now.Before(*e.LeaseUntil)) {
			continue
		}
		a, token, claimed, err := s.claimDeletion(ctx, row, e, now)
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			failures = append(failures, err)
		}
		if err != nil || !claimed {
			continue
		}
		deletion, cancel := context.WithTimeout(ctx, 10*time.Second)
		deleteErr := s.Blobs.Delete(deletion, blobKey(a))
		cancel()
		if deleteErr != nil {
			failures = append(failures, deleteErr)
		}
		finish, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		if err := s.finishDeletion(finish, a, token, deleteErr); err != nil {
			failures = append(failures, err)
		}
		cancelFinish()
	}
	return errors.Join(failures...)
}

func (s *Service) claimDeletion(ctx context.Context, indexRow storage.Record, e Expiration, now time.Time) (Artifact, string, bool, error) {
	runRow, err := s.Store.Get(ctx, "runs", e.Run)
	if err != nil {
		return Artifact{}, "", false, err
	}
	run, err := storage.Decode[runOutcome](runRow)
	if err != nil {
		return Artifact{}, "", false, err
	}
	if run.Project != e.Project || run.FinishedAt == nil {
		return Artifact{}, "", false, nil
	}
	if _, err := s.Store.Get(ctx, "evidence-retention-jobs", e.Run); err == nil {
		return Artifact{}, "", false, nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return Artifact{}, "", false, err
	}
	r, err := s.Store.Get(ctx, Bucket(e.Run), e.Artifact)
	if err != nil {
		return Artifact{}, "", false, err
	}
	a, err := storage.Decode[Artifact](r)
	if err != nil {
		return a, "", false, err
	}
	if a.Project != e.Project || a.Run != e.Run {
		return a, "", false, storage.ErrNotFound
	}
	if a.State == "expired" || a.State == "deleted" {
		return a, "", false, s.Store.Commit(ctx, storage.Change{Record: indexRow, Delete: true})
	}
	if a.State == "pending" && a.UploadUntil != nil && now.Before(*a.UploadUntil) {
		return a, "", false, nil
	}
	if e.Target == "expired" && now.Before(a.ExpiresAt) {
		return a, "", false, nil
	}
	p, pinRow, err := s.ReadPin(ctx, e.Project, e.Run)
	if err != nil {
		return a, "", false, err
	}
	if p.Active(now) {
		return a, "", false, nil
	}
	if e.Token == "" {
		p.InFlight++
	} else if p.InFlight < 1 {
		return a, "", false, errors.New("deletion lease has no run reservation")
	}
	e.Token = domain.NewID()
	until := now.Add(30 * time.Second)
	e.LeaseUntil = &until
	e.Attempts++
	a.State = "deleting"
	a.Error = ""
	metadata, err := storage.Put(r.Bucket, r.ID, e.Project, r.Revision, a)
	if err != nil {
		return a, "", false, err
	}
	index, err := storage.Put(indexRow.Bucket, indexRow.ID, e.Project, indexRow.Revision, e)
	if err != nil {
		return a, "", false, err
	}
	reservation, err := storage.Put("evidence-pins", e.Run, e.Project, pinRow.Revision, p)
	if err != nil {
		return a, "", false, err
	}
	err = s.Store.Commit(ctx, metadata, index, reservation, storage.Change{Record: runRow, Assert: true})
	return a, e.Token, err == nil, err
}

func (s *Service) finishDeletion(ctx context.Context, a Artifact, token string, deleteErr error) error {
	for range 8 {
		indexRow, err := s.Store.Get(ctx, "artifact-expirations", expirationID(a))
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		e, err := storage.Decode[Expiration](indexRow)
		if err != nil {
			return err
		}
		if e.Token != token {
			return storage.ErrConflict
		}
		r, err := s.Store.Get(ctx, Bucket(a.Run), a.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Artifact](r)
		if err != nil {
			return err
		}
		p, pinRow, err := s.ReadPin(ctx, a.Project, a.Run)
		if err != nil {
			return err
		}
		if p.InFlight < 1 {
			return errors.New("deletion reservation is missing")
		}
		p.InFlight--
		pinChange, err := storage.Put("evidence-pins", a.Run, a.Project, pinRow.Revision, p)
		if err != nil {
			return err
		}
		changes := []storage.Change{pinChange}
		if deleteErr != nil {
			current.State = "cleanup-failed"
			current.Error = "Artifact deletion failed. A retry is scheduled."
			e.Token = ""
			e.LeaseUntil = nil
			e.Error = deleteErr.Error()
			e.NextAttempt = time.Now().UTC().Add(time.Minute)
			index, err := storage.Put(indexRow.Bucket, indexRow.ID, a.Project, indexRow.Revision, e)
			if err != nil {
				return err
			}
			changes = append(changes, index)
		} else {
			current.State = e.Target
			current.Error = ""
			now := time.Now().UTC()
			current.DeletedAt = &now
			changes = append(changes, storage.Change{Record: indexRow, Delete: true})
			scopes := []struct{ bucket, id string }{{"artifact-quotas", a.Run}}
			if current.QuotaVersion >= 1 {
				scopes = append(scopes, struct{ bucket, id string }{"artifact-project-quotas", a.Project})
				if a.Test != "" {
					scopes = append(scopes, struct{ bucket, id string }{"artifact-test-quotas", testQuotaKey(a)})
				}
			}
			for _, scope := range scopes {
				change, err := quotaDelta(ctx, s.Store, scope.bucket, scope.id, a.Project, -a.Size, 0, true)
				if err != nil {
					return err
				}
				changes = append(changes, change)
			}
		}
		metadata, err := storage.Put(r.Bucket, r.ID, a.Project, r.Revision, current)
		if err != nil {
			return err
		}
		if err := s.Store.Commit(ctx, append(changes, metadata)...); errors.Is(err, storage.ErrConflict) {
			continue
		} else {
			return err
		}
	}
	return storage.ErrConflict
}
