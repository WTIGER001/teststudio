package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *Controller) Stop(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.update(ctx, ref, false, func(d *dispatch) error { d.Stopping = true; return nil })
	if err != nil || d.Removed || !d.CreateAttempted || d.CreateRejected {
		return err
	}
	_, _, plan, err := c.load(ctx, ref)
	if err != nil {
		return err
	}
	actual, err := c.resolve(ctx, d, plan)
	if err != nil {
		return err
	}
	if quiescent(actual.State) || c.now().Before(d.StopAfter) {
		return nil
	}
	if _, err := c.update(ctx, ref, false, func(d *dispatch) error {
		if d.InstanceAbsent || d.Quiesced || d.Removed || c.now().Before(d.StopAfter) {
			return hostedexecution.ErrUnavailable
		}
		d.StopAfter = c.now().UTC().Add(30 * time.Second)
		return nil
	}); err != nil {
		return err
	}
	if err := bounded(ctx, func(ctx context.Context) error { return c.instances.Stop(ctx, actual.ID) }); err != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (c *Controller) authorizeCleanup(ctx context.Context, d dispatch) error {
	r, _, err := c.transport(ctx, d)
	if err != nil || !r.Revoked {
		return hostedexecution.ErrUnavailable
	}
	return bounded(ctx, func(ctx context.Context) error { return c.exchange.CheckCleanup(ctx, d.Proof.Reference) })
}

func (c *Controller) removeInstance(ctx context.Context, d dispatch, plan *Plan) error {
	actual, err := c.resolve(ctx, d, plan)
	if err == hostedexecution.ErrNotFound {
		if d.Receipt == "" || !d.Quiesced {
			return hostedexecution.ErrUnavailable
		}
		_, err = c.update(ctx, d.Proof.Reference, false, func(current *dispatch) error { current.InstanceAbsent = true; return nil })
		return err
	}
	if err != nil {
		return err
	}
	if !quiescent(actual.State) || c.now().Before(d.DeleteAfter) {
		return hostedexecution.ErrUnavailable
	}
	if _, err := c.update(ctx, d.Proof.Reference, false, func(current *dispatch) error {
		if !current.Quiesced || current.InstanceAbsent || current.Removed || c.now().Before(current.DeleteAfter) {
			return hostedexecution.ErrUnavailable
		}
		current.DeleteAttempted = true
		current.DeleteAfter = c.now().UTC().Add(30 * time.Second)
		return nil
	}); err != nil {
		return err
	}
	if err := bounded(ctx, func(ctx context.Context) error { return c.instances.Delete(ctx, actual.ID) }); err != nil {
		return hostedexecution.ErrUnavailable
	}
	var after Snapshot
	err = bounded(ctx, func(ctx context.Context) error {
		var err error
		after, err = c.instances.Read(ctx, actual.ID)
		return err
	})
	if err == nil && (after.ID != actual.ID || plan.Verify(after) != nil) {
		return hostedexecution.ErrOwnership
	}
	if err != hostedexecution.ErrNotFound {
		return hostedexecution.ErrUnavailable
	}
	_, err = c.update(ctx, d.Proof.Reference, false, func(current *dispatch) error { current.InstanceAbsent = true; return nil })
	return err
}

func (c *Controller) Remove(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, plan, err := c.load(ctx, ref)
	if err != nil || d.Removed {
		return err
	}
	if err := c.authorizeCleanup(ctx, d); err != nil {
		return err
	}
	d, err = c.update(ctx, ref, false, func(current *dispatch) error { current.Stopping = true; return nil })
	if err != nil {
		return err
	}
	if !settled(d) {
		if err := c.removeInstance(ctx, d, plan); err != nil {
			return err
		}
	}
	d, _, _, err = c.load(ctx, ref)
	if err != nil || !settled(d) {
		return hostedexecution.ErrUnavailable
	}
	if d.DeadlineAttempted && !d.DeadlineRemoved {
		if _, err := c.update(ctx, ref, false, func(current *dispatch) error {
			if !settled(*current) {
				return hostedexecution.ErrUnavailable
			}
			current.DeadlineRemoving = true
			return nil
		}); err != nil {
			return err
		}
		if err := bounded(ctx, func(ctx context.Context) error { return c.deadlines.Remove(ctx, c.deadlineSpec(plan)) }); err != nil {
			return hostedexecution.ErrUnavailable
		}
		if _, err := c.update(ctx, ref, false, func(current *dispatch) error { current.DeadlineRemoved = true; return nil }); err != nil {
			return err
		}
	}
	_, err = c.update(ctx, ref, false, func(current *dispatch) error {
		if !settled(*current) || current.DeadlineAttempted && !current.DeadlineRemoved {
			return hostedexecution.ErrUnavailable
		}
		current.Removed = true
		return nil
	})
	return err
}
