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
	return dispatchGuards(ctx, m.exchange, proof, job.Ticket.Token, "", m.now())
}
