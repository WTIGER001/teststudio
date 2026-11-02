package storage

import (
	"context"
	"errors"
	"time"

	"modernc.org/sqlite"
)

func retrySQLiteBusy(ctx context.Context, operation func() error) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := operation()
		var failure *sqlite.Error
		if !errors.As(err, &failure) || (failure.Code()&255 != 5 && failure.Code()&255 != 6) || time.Now().After(deadline) {
			return err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
