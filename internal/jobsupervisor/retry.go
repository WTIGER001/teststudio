package jobsupervisor

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func (s *Supervisor) retry(ctx context.Context, operation func() error) error {
	attempts := s.Attempts
	if attempts == 0 {
		attempts = 6
	}
	if attempts < 1 || attempts > 10 {
		return ErrDelivery
	}
	delay := s.RetryDelay
	if delay == 0 {
		delay = 150 * time.Millisecond
	}
	if delay < 0 || delay > 2*time.Second {
		return ErrDelivery
	}
	for index := 0; index < attempts; index++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := operation()
		if err == nil || !errors.Is(err, workerprotocol.ErrUnavailable) && !errors.Is(err, workerprotocol.ErrConflict) {
			return err
		}
		if index == attempts-1 {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, 2*time.Second)
	}
	return ErrDelivery
}
