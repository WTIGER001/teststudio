package ecsworker

import (
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type Proof struct {
	Environment *GatewayConfig      `json:"environment,omitempty"`
	Gateway     *GatewayConfig      `json:"gateway,omitempty"`
	Reference   execution.Reference `json:"reference"`
	Image       string              `json:"image"`
	Runner      string              `json:"runner"`
	Operation   string              `json:"operation,omitempty"`
	Deadline    time.Time           `json:"deadline"`
	TicketHash  string              `json:"ticketHash"`
	Fingerprint string              `json:"fingerprint"`
}

func (p *Plan) Proof() Proof {
	value := Proof{Reference: p.job.Reference, Image: p.job.Image, Runner: p.job.Runner, Operation: p.job.Operation, Deadline: p.job.Ticket.Deadline, TicketHash: p.ticketHash, Fingerprint: p.fingerprint}
	if p.gateway != nil {
		selected := canonicalGateway(*p.gateway)
		value.Gateway = &selected
	}
	if p.environment != nil {
		selected := canonicalGateway(*p.environment)
		value.Environment = &selected
	}
	return value
}

func Restore(config Config, proof Proof) (*Plan, error) {
	job := hostedexecution.Job{Reference: proof.Reference, Image: proof.Image, Runner: proof.Runner, Operation: proof.Operation, ExchangeOrigin: config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: proof.Reference.ID, Deadline: proof.Deadline}}
	if proof.Environment != nil && proof.Gateway != nil {
		return nil, hostedexecution.ErrOwnership
	}
	plan, err := newRolePlan(config, job, proof.TicketHash, proof.Gateway, proof.Environment)
	if err != nil || plan.fingerprint != proof.Fingerprint {
		return nil, hostedexecution.ErrOwnership
	}
	return plan, nil
}
