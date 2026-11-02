package evidence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) RequestDelete(ctx context.Context, actor, project, run, id, reason string, revision int64) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return errors.New("evidence deletion requires a reason of up to 500 characters")
	}
	runRow, err := s.Store.Get(ctx, "runs", run)
	if err != nil {
		return err
	}
	result, err := storage.Decode[runOutcome](runRow)
	if err != nil {
		return err
	}
	if result.Project != project {
		return storage.ErrNotFound
	}
	if result.FinishedAt == nil {
		return errors.New("finish or cancel the run before deleting its evidence")
	}
	r, err := s.Store.Get(ctx, Bucket(run), id)
	if err != nil {
		return err
	}
	a, err := storage.Decode[Artifact](r)
	if err != nil {
		return err
	}
	if a.Project != project || a.Run != run {
		return storage.ErrNotFound
	}
	if r.Revision != revision {
		return storage.ErrConflict
	}
	if a.State == "expired" || a.State == "deleted" {
		return ErrUnavailable
	}
	if a.UploadUntil != nil && time.Now().Before(*a.UploadUntil) {
		return errors.New("an artifact upload is in progress; wait for it to finish before deletion")
	}
	p, pinRow, err := s.ReadPin(ctx, project, run)
	if err != nil {
		return err
	}
	if p.Active(time.Now()) || p.InFlight > 0 {
		return errors.New("unpin the run and wait for active deletions before deleting evidence")
	}
	index, err := s.expirationChange(ctx, a)
	if err != nil {
		return err
	}
	e, err := storage.Decode[Expiration](index.Record)
	if err != nil {
		return err
	}
	if e.Token != "" {
		return storage.ErrConflict
	}
	e.Target = "deleted"
	e.NextAttempt = time.Now().UTC()
	e.Error = ""
	index, err = storage.Put(index.Bucket, index.ID, project, index.Revision, e)
	if err != nil {
		return err
	}
	a.State = "delete-pending"
	a.Error = ""
	metadata, err := storage.Put(r.Bucket, r.ID, project, r.Revision, a)
	if err != nil {
		return err
	}
	guard, err := storage.Put("evidence-pins", run, project, pinRow.Revision, p)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(actor, project, "evidence.delete", run+"/"+id, nil, map[string]string{"reason": strings.TrimSpace(reason)})
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, metadata, index, guard, audit, storage.Change{Record: runRow, Assert: true})
}
