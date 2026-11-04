package workerexchange

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

func (s *Service) Activate(ctx context.Context, ref execution.Reference) error {
	for range 8 {
		record, row, err := s.owned(ctx, ref)
		if err != nil {
			return ErrUnavailable
		}
		if record.Revoked || !time.Now().Before(record.Deadline) || s.Authorize == nil {
			return ErrDenied
		}
		access, err := s.Authorize(ctx, ref)
		if err != nil || !time.Now().Before(access.Deadline) {
			return ErrDenied
		}
		if !record.DispatchGated || record.DispatchReady {
			return nil
		}
		record.DispatchReady = true
		if err := s.save(ctx, row, record, access.Records); err == ErrConflict {
			continue
		} else {
			return err
		}
	}
	return ErrConflict
}
