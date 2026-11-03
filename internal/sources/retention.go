package sources

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Sweep(ctx context.Context, cursor string) (string, error) {
	page, err := s.Store.List(ctx, "source-staging", "", cursor, 100)
	if err != nil {
		return cursor, err
	}
	for _, staged := range page.Items {
		upload, err := storage.Decode[Upload](staged)
		if err != nil {
			return cursor, err
		}
		if time.Now().Before(upload.ExpiresAt) {
			continue
		}
		row, err := s.Store.Get(ctx, Bucket(upload.Project, upload.Suite), upload.ID)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return cursor, err
		}
		current, err := storage.Decode[Upload](row)
		if err != nil {
			return cursor, err
		}
		if current.State == "published" {
			continue
		}
		content, err := s.Store.Get(ctx, "source-content/"+upload.Project, upload.ID)
		if err != nil {
			return cursor, err
		}
		current.State, current.PlanID, current.Issues = "expired", "", []catalog.Issue{}
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, current)
		if err != nil {
			return cursor, err
		}
		changes := []storage.Change{saved, {Record: staged, Delete: true}, {Record: content, Delete: true}}
		if upload.PlanID != "" {
			plan, err := s.Store.Get(ctx, catalog.Bucket("imports", upload.Project, upload.Suite), upload.PlanID)
			if err != nil {
				return cursor, err
			}
			changes = append(changes, storage.Change{Record: plan, Delete: true})
		}
		if err := s.Store.Commit(ctx, changes...); err != nil && !errors.Is(err, storage.ErrConflict) {
			return cursor, err
		}
	}
	return page.Cursor, nil
}

func (s *Service) StartMaintenance(ctx context.Context, interval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		cursor := ""
		bundleCursor := ""
		repositoryCursor := ""
		for {
			next, err := s.Sweep(ctx, cursor)
			if err != nil && ctx.Err() == nil {
				slog.Error("source staging cleanup failed")
			} else {
				cursor = next
			}
			nextBundle, err := s.SweepBundles(ctx, bundleCursor)
			if err != nil && ctx.Err() == nil {
				slog.Error("source bundle staging cleanup failed")
			} else {
				bundleCursor = nextBundle
			}
			nextRepository, err := s.SweepRepositorySnapshots(ctx, repositoryCursor)
			if err != nil && ctx.Err() == nil {
				slog.Error("repository snapshot cleanup failed")
			} else {
				repositoryCursor = nextRepository
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
