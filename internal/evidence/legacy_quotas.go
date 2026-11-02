package evidence

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func quotaDelta(ctx context.Context, store *storage.Store, bucket, id, project string, bytes, limit int64, keepLimit bool) (storage.Change, error) {
	r, err := store.Get(ctx, bucket, id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Change{}, err
	}
	q := Quota{Limit: limit}
	if r.Revision > 0 {
		q, err = storage.Decode[Quota](r)
		if err != nil {
			return storage.Change{}, err
		}
	}
	if !keepLimit {
		q.Limit = limit
	}
	q.Bytes += bytes
	if q.Bytes < 0 {
		return storage.Change{}, errors.New("artifact quota accounting would become negative")
	}
	return storage.Put(bucket, id, project, r.Revision, q)
}

func (s *Service) legacyQuotaChanges(ctx context.Context, project, run string, sizes map[string]int64) ([]storage.Change, error) {
	if len(sizes) == 0 {
		return nil, nil
	}
	projectLimit := capture.DefaultLimits().ProjectBytes
	changes := []storage.Change{}
	r, err := s.Store.Get(ctx, "projects", project)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		p, err := storage.Decode[projects.Project](r)
		if err != nil {
			return nil, err
		}
		projectLimit = capture.ResolveLimits(p.CaptureLimits).ProjectBytes
		changes = append(changes, storage.Change{Record: r, Assert: true})
	}
	runQuotaRow, err := s.Store.Get(ctx, "artifact-quotas", run)
	if err != nil {
		return nil, err
	}
	runQuota, err := storage.Decode[Quota](runQuotaRow)
	if err != nil {
		return nil, err
	}
	var total int64
	for test, bytes := range sizes {
		total += bytes
		if test == "" {
			continue
		}
		change, err := quotaDelta(ctx, s.Store, "artifact-test-quotas", run+"/"+test, project, bytes, runQuota.Limit, true)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	change, err := quotaDelta(ctx, s.Store, "artifact-project-quotas", project, project, total, projectLimit, false)
	if err != nil {
		return nil, err
	}
	return append(changes, change), nil
}
