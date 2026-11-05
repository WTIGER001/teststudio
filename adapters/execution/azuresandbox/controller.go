package azuresandbox

import (
	"context"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func (c *Controller) Ensure(ctx context.Context, plan *Plan) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if plan == nil || hash(plan.config) != hash(c.config) {
		return hostedexecution.ErrOwnership
	}
	old, _, _, lookup := c.load(ctx, plan.proof.Reference)
	if lookup == nil {
		if old.Proof.Fingerprint != plan.proof.Fingerprint {
			return hostedexecution.ErrOwnership
		}
		return c.Advance(ctx, plan.proof.Reference)
	}
	if lookup != hostedexecution.ErrNotFound {
		return lookup
	}
	if _, err := plan.launch(c.now()); err != nil {
		return err
	}
	d, err := c.reserve(ctx, plan)
	if err != nil {
		return err
	}
	if d.Proof.Fingerprint != plan.proof.Fingerprint {
		return hostedexecution.ErrOwnership
	}
	return c.Advance(ctx, plan.proof.Reference)
}

func (c *Controller) deadlineSpec(plan *Plan) DeadlineSpec {
	return DeadlineSpec{Config: c.config, Proof: plan.Proof()}
}

func (c *Controller) ensureDeadline(ctx context.Context, d dispatch, plan *Plan) error {
	ref := d.Proof.Reference
	spec := c.deadlineSpec(plan)
	if !d.DeadlineAttempted {
		if _, err := c.update(ctx, ref, true, func(current *dispatch) error {
			if current.DeadlineAttempted {
				return hostedexecution.ErrUnavailable
			}
			current.DeadlineAttempted = true
			return nil
		}); err != nil {
			return err
		}
		if err := bounded(ctx, func(ctx context.Context) error { return c.deadlines.Ensure(ctx, spec) }); err != nil {
			return hostedexecution.ErrUnavailable
		}
	}
	if err := bounded(ctx, func(ctx context.Context) error { return c.deadlines.Verify(ctx, spec) }); err != nil {
		return hostedexecution.ErrUnavailable
	}
	_, err := c.update(ctx, ref, true, func(current *dispatch) error { current.DeadlineVerified = true; return nil })
	return err
}

func (c *Controller) Advance(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, plan, err := c.load(ctx, ref)
	if err != nil {
		return err
	}
	if d.CreateRejected || d.CreationBlocked || d.Stopping || d.Removed {
		return hostedexecution.ErrRejected
	}
	if d.CreateAttempted {
		_, err := c.Observe(ctx, ref)
		return err
	}
	if err := c.ensureDeadline(ctx, d, plan); err != nil {
		return err
	}
	var ticket workerexchange.Ticket
	if err := bounded(ctx, func(ctx context.Context) error {
		var err error
		ticket, err = c.exchange.ResumeDispatch(ctx, ref)
		return err
	}); err != nil {
		return hostedexecution.ErrUnavailable
	}
	if ticket.Execution != ref.ID || !ticket.Deadline.Equal(plan.proof.Deadline) || hash(ticket.Token) != plan.proof.TicketHash {
		return hostedexecution.ErrOwnership
	}
	plan.token = ticket.Token
	if _, err := plan.launch(c.now()); err != nil {
		return err
	}
	if err := bounded(ctx, func(ctx context.Context) error { return c.admission.BeforeCreate(ctx, c.config) }); err != nil {
		return err
	}
	var prior []Snapshot
	if err := bounded(ctx, func(ctx context.Context) error {
		var err error
		prior, err = c.instances.Inventory(ctx)
		return err
	}); err != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, candidate := range prior {
		if plan.Verify(candidate) == nil {
			current, _, _, err := c.load(ctx, ref)
			if err != nil || current.CreateAttempted {
				return hostedexecution.ErrUnavailable
			}
			_, err = c.update(ctx, ref, true, func(current *dispatch) error {
				if current.CreateAttempted {
					return hostedexecution.ErrUnavailable
				}
				current.CreationBlocked = true
				return nil
			})
			if err != nil {
				return err
			}
			return hostedexecution.ErrOwnership
		}
	}
	if _, err := c.update(ctx, ref, true, func(current *dispatch) error {
		if current.CreateAttempted || !current.DeadlineVerified {
			return hostedexecution.ErrUnavailable
		}
		current.CreateAttempted = true
		return nil
	}); err != nil {
		return err
	}
	var actual Snapshot
	err = bounded(ctx, func(ctx context.Context) error {
		var err error
		actual, err = c.instances.Create(ctx, plan)
		return err
	})
	if err == hostedexecution.ErrRejected {
		_, saveErr := c.update(ctx, ref, false, func(current *dispatch) error {
			if current.Receipt != "" {
				return hostedexecution.ErrOwnership
			}
			current.CreateRejected = true
			return nil
		})
		if saveErr != nil {
			return saveErr
		}
		return err
	}
	if err != nil {
		return err
	}
	return c.remember(ctx, plan, actual)
}
