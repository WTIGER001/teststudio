package workerexchange

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ResumeDispatch(ctx context.Context, ref execution.Reference) (Ticket, error) {
	record, row, err := s.owned(ctx, ref)
	if err != nil || s.Authorize == nil || !record.DispatchGated || record.DispatchReady || record.Revoked || !time.Now().Before(record.Deadline) {
		return Ticket{}, ErrDenied
	}
	access, err := s.Authorize(ctx, ref)
	if err != nil || access.Deadline.Before(record.Deadline) || !time.Now().Before(access.Deadline) {
		return Ticket{}, ErrDenied
	}
	token, err := s.token(ctx, record)
	if err != nil {
		return Ticket{}, ErrUnavailable
	}
	guards := []storage.Change{{Record: row, Assert: true}}
	for _, guard := range access.Records {
		guards = append(guards, storage.Change{Record: guard, Assert: true})
	}
	if err := s.Store.Commit(ctx, guards...); err != nil {
		return Ticket{}, ErrUnavailable
	}
	return Ticket{Execution: ref.ID, Token: token, Deadline: record.Deadline}, nil
}
