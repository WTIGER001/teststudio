package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func dispatchGuards(ctx context.Context, exchange *workerexchange.Service, proof Proof, token, inputHash string, now time.Time) ([]storage.Change, error) {
	c := Controller{exchange: exchange, store: exchange.Store}
	r, row, err := c.transport(ctx, dispatch{Proof: proof, InputHash: inputHash})
	if err != nil {
		return nil, err
	}
	if r.Revoked || r.DispatchReady || !now.Before(r.Deadline) {
		return nil, hostedexecution.ErrRejected
	}
	ticket, err := exchange.ResumeDispatch(ctx, proof.Reference)
	if err != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	if ticket.Execution != proof.Reference.ID || hash(ticket.Token) != hash(token) || !ticket.Deadline.Equal(proof.Deadline) {
		return nil, hostedexecution.ErrOwnership
	}
	access, err := exchange.Authorize(ctx, proof.Reference)
	if err != nil || access.Deadline.Before(r.Deadline) {
		return nil, hostedexecution.ErrUnavailable
	}
	guards := []storage.Change{{Record: row, Assert: true}}
	for _, guard := range access.Records {
		guards = append(guards, storage.Change{Record: guard, Assert: true})
	}
	return guards, nil
}
