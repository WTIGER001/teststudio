package azureworker

import (
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type Proof struct {
	Reference      execution.Reference `json:"reference"`
	Image          string              `json:"image"`
	Runner         string              `json:"runner"`
	Operation      string              `json:"operation,omitempty"`
	Deadline       time.Time           `json:"deadline"`
	TimeoutSeconds int32               `json:"timeoutSeconds"`
	TicketHash     string              `json:"ticketHash"`
	Fingerprint    string              `json:"fingerprint"`
}

func (p *Plan) Proof() Proof {
	return Proof{Reference: p.job.Reference, Image: p.job.Image, Runner: p.job.Runner, Operation: p.job.Operation, Deadline: p.job.Ticket.Deadline, TimeoutSeconds: p.timeout, TicketHash: p.ticketHash, Fingerprint: p.fingerprint}
}

func Restore(config Config, proof Proof) (*Plan, error) {
	job := hostedexecution.Job{Reference: proof.Reference, Image: proof.Image, Runner: proof.Runner, Operation: proof.Operation, ExchangeOrigin: config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: proof.Reference.ID, Deadline: proof.Deadline}}
	p, err := newPlan(config, job, proof.TicketHash, proof.TimeoutSeconds)
	if err != nil || p.fingerprint != proof.Fingerprint {
		return nil, hostedexecution.ErrOwnership
	}
	return p, nil
}
