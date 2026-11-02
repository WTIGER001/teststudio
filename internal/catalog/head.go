package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) head(ctx context.Context, project, suite string) (storage.Record, error) {
	r, err := s.Store.Get(ctx, "catalog-heads/"+project, suite)
	if errors.Is(err, storage.ErrNotFound) {
		return r, nil
	}
	return r, err
}

func headChange(project, suite string, revision int64) (storage.Change, error) {
	return storage.Put("catalog-heads/"+project, suite, project, revision, time.Now().UTC())
}

func (s *Service) commitTests(ctx context.Context, project, suite string, changes ...storage.Change) error {
	head, err := s.head(ctx, project, suite)
	if err != nil {
		return err
	}
	guard, err := headChange(project, suite, head.Revision)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, append(changes, guard)...)
}
