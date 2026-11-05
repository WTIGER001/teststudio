package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (m *ImageManager) authorize(ctx context.Context, job hostedexecution.Job, plan *ImagePlan) ([]storage.Change, error) {
	if job.Operation != "discover" || !validReference(job.Reference, job.Runner, job.Operation) || job.ExchangeOrigin != m.config.ExchangeOrigin || job.Image != plan.proof.Image.Reference || job.Runner != plan.proof.Image.Runner || job.Reference.Project != plan.proof.Image.Project || !ticketToken.MatchString(job.Ticket.Token) || job.Ticket.Execution != job.Reference.ID || !job.Ticket.Deadline.After(m.now()) || job.Ticket.Deadline.Sub(m.now()) > 2*time.Hour {
		return nil, hostedexecution.ErrConfiguration
	}
	proof := Proof{Reference: job.Reference, Image: job.Image, SourceDigest: plan.proof.Image.SourceDigest, Runner: job.Runner, Operation: job.Operation, Deadline: job.Ticket.Deadline}
	c := Controller{exchange: m.exchange, store: m.exchange.Store}
	r, row, err := c.transport(ctx, dispatch{Proof: proof})
	if err != nil {
		return nil, err
	}
	if r.Revoked || r.DispatchReady {
		return nil, hostedexecution.ErrRejected
	}
	ticket, err := m.exchange.ResumeDispatch(ctx, job.Reference)
	if err != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	if ticket.Execution != job.Reference.ID || hash(ticket.Token) != hash(job.Ticket.Token) || !ticket.Deadline.Equal(job.Ticket.Deadline) {
		return nil, hostedexecution.ErrOwnership
	}
	access, err := m.exchange.Authorize(ctx, job.Reference)
	if err != nil || access.Deadline.Before(r.Deadline) {
		return nil, hostedexecution.ErrUnavailable
	}
	guards := []storage.Change{{Record: row, Assert: true}}
	for _, guard := range access.Records {
		guards = append(guards, storage.Change{Record: guard, Assert: true})
	}
	return guards, nil
}
