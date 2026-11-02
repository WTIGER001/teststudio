package evidence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrUnavailable = errors.New("artifact content is unavailable or its retention has ended")

func (s *Service) ReadPin(ctx context.Context, project, run string) (Pin, storage.Record, error) {
	r, err := s.Store.Get(ctx, "evidence-pins", run)
	if errors.Is(err, storage.ErrNotFound) {
		return Pin{Project: project, Run: run}, r, nil
	}
	if err != nil {
		return Pin{}, r, err
	}
	p, err := storage.Decode[Pin](r)
	if err == nil && (p.Project != project || p.Run != run) {
		err = storage.ErrNotFound
	}
	return p, r, err
}

func (s *Service) SavePin(ctx context.Context, actor, project, run string, pinned bool, until *time.Time, reason string, revision int64) error {
	p, r, err := s.ReadPin(ctx, project, run)
	if err != nil {
		return err
	}
	if r.Revision != revision {
		return storage.ErrConflict
	}
	if len(strings.TrimSpace(reason)) > 500 || (pinned && strings.TrimSpace(reason) == "") {
		return errors.New("a retention pin requires a reason of up to 500 characters")
	}
	if pinned && p.InFlight > 0 {
		return errors.New("evidence deletion is in progress; wait for it to finish before pinning")
	}
	if pinned && until != nil && (!until.After(time.Now()) || until.After(time.Now().Add(3650*24*time.Hour))) {
		return errors.New("pin expiry must be in the next 3650 days")
	}
	before := p
	p.Pinned = pinned
	p.Until = until
	p.Reason = strings.TrimSpace(reason)
	p.Actor = actor
	p.UpdatedAt = time.Now().UTC()
	if !pinned {
		p.Until = nil
	}
	change, err := storage.Put("evidence-pins", run, project, r.Revision, p)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(actor, project, "evidence.pin", run, before, p)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, change, audit)
}

func (s *Service) Available(ctx context.Context, a Artifact, now time.Time) (bool, error) {
	if a.State != "available" {
		return false, nil
	}
	p, _, err := s.ReadPin(ctx, a.Project, a.Run)
	if err != nil {
		return false, err
	}
	return p.Active(now) || now.Before(a.ExpiresAt), nil
}
