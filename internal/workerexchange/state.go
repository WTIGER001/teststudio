package workerexchange

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

type State struct {
	DispatchPending bool
	Deadline        time.Time
	StartedAt       *time.Time
	Submitted       bool
	Uploaded        bool
	Revoked         bool
}

func (s *Service) State(ctx context.Context, ref execution.Reference) (State, error) {
	record, _, err := s.owned(ctx, ref)
	if err != nil {
		return State{}, err
	}
	state := State{DispatchPending: record.DispatchGated && !record.DispatchReady, Deadline: record.Deadline, StartedAt: record.StartedAt, Submitted: record.Submission != nil, Uploaded: record.Submission != nil, Revoked: record.Revoked}
	if record.Submission != nil {
		for _, file := range record.Submission.Files {
			state.Uploaded = state.Uploaded && record.Uploaded[file.Path]
		}
	}
	return state, nil
}
