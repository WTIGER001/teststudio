package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKDeadlines) Remove(ctx context.Context, s DeadlineSpec) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.load(ctx, s)
	if err != nil || d.DeadlineRemoved {
		return err
	}
	if !settled(d) {
		return hostedexecution.ErrUnavailable
	}
	if err := c.control.authorizeCleanup(ctx, d); err != nil {
		return err
	}
	if d.Receipt != "" {
		err := bounded(ctx, func(ctx context.Context) error { _, err := c.control.instances.Read(ctx, d.Receipt); return err })
		if err != hostedexecution.ErrNotFound {
			return hostedexecution.ErrUnavailable
		}
	}
	d, err = c.control.update(ctx, s.Proof.Reference, false, func(current *dispatch) error {
		if !settled(*current) || current.TimerProfile != "" && current.TimerProfile != c.config.profile() {
			return hostedexecution.ErrOwnership
		}
		current.TimerProfile = c.config.profile()
		current.Stopping = true
		current.DeadlineRemoving = true
		return nil
	})
	if err != nil {
		return err
	}
	actual, err := c.get(ctx, s)
	if err == hostedexecution.ErrNotFound {
		if d.TimerCreateAttempted && !d.TimerCreateRejected && !d.TimerSeen {
			return hostedexecution.ErrUnavailable
		}
		return c.removed(ctx, s)
	}
	if err != nil {
		return err
	}
	if !d.TimerCreateAttempted || d.TimerCreateRejected || c.config.verify(s, actual, false) != nil {
		return hostedexecution.ErrOwnership
	}
	if c.control.now().Before(d.TimerCheckAfter) {
		return hostedexecution.ErrUnavailable
	}
	if _, err := c.control.update(ctx, s.Proof.Reference, false, func(current *dispatch) error {
		if current.DeadlineRemoved || c.control.now().Before(current.TimerCheckAfter) {
			return hostedexecution.ErrUnavailable
		}
		current.TimerSeen = true
		current.TimerCheckAfter = c.control.now().UTC().Add(30 * time.Second)
		return nil
	}); err != nil {
		return err
	}
	step, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = c.workflows.Delete(step, c.config.ResourceGroup, c.config.name(s), nil)
	if err = timerError(err, false); err != nil && err != hostedexecution.ErrNotFound {
		return err
	}
	if _, err := c.get(ctx, s); err != hostedexecution.ErrNotFound {
		return hostedexecution.ErrUnavailable
	}
	return c.removed(ctx, s)
}

func (c *SDKDeadlines) removed(ctx context.Context, s DeadlineSpec) error {
	_, err := c.control.update(ctx, s.Proof.Reference, false, func(d *dispatch) error { d.DeadlineRemoved = true; return nil })
	return err
}
