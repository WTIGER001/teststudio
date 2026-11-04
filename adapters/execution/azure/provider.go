package azureworker

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) Ensure(ctx context.Context, job hostedexecution.Job) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if job.ExchangeOrigin != p.config.ExchangeOrigin {
		return hostedexecution.ErrOwnership
	}
	old, _, _, lookup := p.load(ctx, job.Reference)
	if lookup == nil {
		if !sameJob(old.Proof, job) {
			return hostedexecution.ErrOwnership
		}
		if old.Removed || old.CreateRejected || old.CreationBlocked || old.StartRejected || old.Stopping {
			return hostedexecution.ErrRejected
		}
		return nil
	}
	if !errors.Is(lookup, storage.ErrNotFound) {
		return lookup
	}
	plan, err := NewPlan(p.config, job, p.now())
	if err != nil {
		return err
	}
	_, winning, err := p.reserve(ctx, plan)
	if err != nil || !winning {
		return err
	}
	image, err := p.InspectImage(ctx, job.Image)
	if err != nil || image.Runner != job.Runner || image.Project != job.Reference.Project || job.Operation == "discover" && image.Discovery != "worker-exchange/v1" {
		return p.rejectCreation(ctx, job.Reference, false)
	}
	if err := p.network.Admit(ctx, p.config); err != nil {
		return p.rejectCreation(ctx, job.Reference, false)
	}
	if _, err := p.jobs.Get(ctx, plan.Name()); err == nil {
		return p.rejectCreation(ctx, job.Reference, true)
	} else if !errors.Is(err, hostedexecution.ErrNotFound) {
		return hostedexecution.ErrUnavailable
	}
	if _, err := p.update(ctx, job.Reference, true, func(d *dispatch) error {
		if d.CreateAttempted || d.CreateRejected || d.CreationBlocked {
			return hostedexecution.ErrUnavailable
		}
		d.CreateAttempted = true
		return nil
	}); err != nil {
		return err
	}
	if err := p.jobs.Create(ctx, plan.Name(), plan.Definition()); err != nil {
		if errors.Is(err, hostedexecution.ErrRejected) {
			return p.rejectCreation(ctx, job.Reference, false)
		}
		return nil
	}
	return nil
}

func (p *Provider) rejectCreation(ctx context.Context, ref execution.Reference, blocked bool) error {
	_, err := p.update(ctx, ref, false, func(d *dispatch) error {
		d.CreationBlocked, d.CreateRejected = blocked, !blocked
		return nil
	})
	if err != nil {
		return err
	}
	if blocked {
		return hostedexecution.ErrOwnership
	}
	return hostedexecution.ErrRejected
}

func (p *Provider) prepareStart(ctx context.Context, d dispatch, plan *Plan) error {
	if !d.DeadlineAttempted {
		if _, err := p.update(ctx, plan.job.Reference, true, func(current *dispatch) error {
			if current.DeadlineAttempted {
				return hostedexecution.ErrUnavailable
			}
			current.DeadlineAttempted = true
			return nil
		}); err != nil {
			return err
		}
		if err := p.deadlines.Ensure(ctx, plan.deadlineSpec()); err != nil {
			return hostedexecution.ErrUnavailable
		}
	}
	if err := p.deadlines.Verify(ctx, plan.deadlineSpec()); err != nil {
		return hostedexecution.ErrUnavailable
	}
	ticket, err := p.resumeDispatch(ctx, plan.job.Reference)
	if err != nil || ticket.Execution != plan.job.Reference.ID || !ticket.Deadline.Equal(plan.job.Ticket.Deadline) || !token.MatchString(ticket.Token) || fingerprint(ticket.Token) != plan.ticketHash {
		return hostedexecution.ErrUnavailable
	}
	if err := p.network.Admit(ctx, p.config); err != nil {
		return err
	}
	prior, err := p.jobs.Executions(ctx, plan.Name())
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if len(prior) != 0 {
		latest, _, _, err := p.load(ctx, plan.job.Reference)
		if err != nil || latest.StartAttempted {
			return hostedexecution.ErrUnavailable
		}
		return hostedexecution.ErrOwnership
	}
	plan.job.Ticket = ticket
	start, err := plan.Start(p.now())
	if err != nil {
		return err
	}
	if _, err := p.update(ctx, plan.job.Reference, true, func(current *dispatch) error {
		if current.StartAttempted {
			return hostedexecution.ErrUnavailable
		}
		current.DeadlineVerified, current.StartAttempted = true, true
		return nil
	}); err != nil {
		return err
	}
	receipt, err := p.jobs.Start(ctx, plan.Name(), start)
	if errors.Is(err, hostedexecution.ErrRejected) {
		_, saveErr := p.update(ctx, plan.job.Reference, false, func(current *dispatch) error { current.StartRejected = true; return nil })
		if saveErr != nil {
			return saveErr
		}
		return hostedexecution.ErrRejected
	}
	if err != nil || receipt == nil {
		return hostedexecution.ErrUnavailable
	}
	return p.saveReceipt(ctx, plan, *receipt)
}
