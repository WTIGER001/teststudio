package evidence

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func testQuotaKey(a Artifact) string { return a.Run + "/" + a.Test }

func quotaReservation(ctx context.Context, store *storage.Store, bucket, id, project string, bytes, limit int64, stable bool) (storage.Change, error) {
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
		if stable && q.Limit != limit {
			return storage.Change{}, errors.New("capture quota differs from the run snapshot")
		}
	}
	q.Limit = limit
	if q.Bytes+bytes > limit {
		return storage.Change{}, ErrQuota
	}
	q.Bytes += bytes
	return storage.Put(bucket, id, project, r.Revision, q)
}

func (s *Service) reservations(ctx context.Context, a Artifact, budget Budget) ([]storage.Change, error) {
	projectLimit := budget.ProjectBytes
	changes := []storage.Change{}
	r, err := s.Store.Get(ctx, "projects", a.Project)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		p, err := storage.Decode[projects.Project](r)
		if err != nil {
			return nil, err
		}
		limits := capture.ResolveLimits(p.CaptureLimits)
		projectLimit = min(projectLimit, limits.ProjectBytes)
		if a.Size > limits.FileBytes {
			return nil, ErrQuota
		}
		changes = append(changes, storage.Change{Record: r, Assert: true})
	}
	for _, scope := range []struct {
		bucket, id string
		limit      int64
		stable     bool
	}{
		{"artifact-quotas", a.Run, budget.RunBytes, true},
		{"artifact-project-quotas", a.Project, projectLimit, false},
		{"artifact-test-quotas", testQuotaKey(a), budget.TestBytes, true},
	} {
		if scope.bucket == "artifact-test-quotas" && a.Test == "" {
			continue
		}
		change, err := quotaReservation(ctx, s.Store, scope.bucket, scope.id, a.Project, a.Size, scope.limit, scope.stable)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
