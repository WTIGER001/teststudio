package recording

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) StartMaintenance(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTicker(5 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				s.maintain(ctx)
			}
		}
	}()
	return done
}

func (s *Service) maintain(ctx context.Context) {
	if s.Driver == nil {
		return
	}
	cursor := ""
	for {
		page, err := s.Store.List(ctx, Bucket(), "", cursor, 30)
		if err != nil {
			return
		}
		for _, row := range page.Items {
			value, err := storage.Decode[Session](row)
			if err != nil || !active(value.State) {
				continue
			}
			if time.Now().Before(value.DeadlineAt) && s.check(ctx, value.Actor, value.Project, value.Suite, true) == nil {
				continue
			}
			s.operation.Lock()
			current, err := s.Store.Get(ctx, row.Bucket, row.ID)
			if err == nil {
				value, err = storage.Decode[Session](current)
			}
			if err == nil && active(value.State) {
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
				snapshot, stopErr := s.Driver.Stop(cleanup, row.ID)
				if stopErr == nil {
					snapshot.State, snapshot.Error = "failed", "Recording stopped because its time limit or current source authority no longer permits it. Captured code remains private to its owner."
					_, _ = s.finish(cleanup, current, value, snapshot)
				}
				cancel()
			}
			s.operation.Unlock()
		}
		if page.Cursor == "" {
			return
		}
		cursor = page.Cursor
	}
}
