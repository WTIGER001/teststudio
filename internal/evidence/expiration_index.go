package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func expirationID(a Artifact) string {
	hash := sha256.Sum256([]byte(a.Project + "\x00" + a.Run + "\x00" + a.ID))
	return hex.EncodeToString(hash[:])
}

func (s *Service) expirationChange(ctx context.Context, a Artifact) (storage.Change, error) {
	r, err := s.Store.Get(ctx, "artifact-expirations", expirationID(a))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Change{}, err
	}
	e := Expiration{Project: a.Project, Run: a.Run, Artifact: a.ID, Target: "expired", NextAttempt: a.ExpiresAt}
	if r.Revision > 0 {
		e, err = storage.Decode[Expiration](r)
		if err != nil {
			return storage.Change{}, err
		}
		if e.Project != a.Project || e.Run != a.Run || e.Artifact != a.ID {
			return storage.Change{}, storage.ErrConflict
		}
		if e.Target == "expired" && e.Token == "" {
			e.NextAttempt = a.ExpiresAt
		}
	}
	return storage.Put("artifact-expirations", expirationID(a), a.Project, r.Revision, e)
}

func (s *Service) EnsureRetentionIndex(ctx context.Context, limit int) error {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()
	page, err := s.Store.List(ctx, "runs", "", s.indexCursor, limit)
	if err != nil {
		return err
	}
	s.indexCursor = page.Cursor
	for _, row := range page.Items {
		run, err := storage.Decode[runOutcome](row)
		if err != nil {
			return err
		}
		if run.FinishedAt == nil {
			continue
		}
		statusRow, err := s.Store.Get(ctx, "evidence-retention-status", row.ID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if err == nil {
			status, err := storage.Decode[RetentionStatus](statusRow)
			if err != nil {
				return err
			}
			if status.IndexVersion >= 2 {
				continue
			}
		}
		if _, err := s.Store.Get(ctx, "evidence-retention-jobs", row.ID); err == nil {
			continue
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		job, err := storage.Put("evidence-retention-jobs", row.ID, run.Project, 0, RetentionJob{Project: run.Project, Run: row.ID, NextAttempt: time.Now().UTC()})
		if err != nil {
			return err
		}
		if err := s.Store.Commit(ctx, job, storage.Change{Record: row, Assert: true}); err != nil && !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return nil
}
