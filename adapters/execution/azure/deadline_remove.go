package azureworker

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKDeadlines) Remove(ctx context.Context, s DeadlineSpec) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.load(ctx, s)
	if err != nil || d.DeadlineRemoved {
		return err
	}
	if err := c.provider.authorizeCleanup(ctx, d); err != nil {
		return err
	}
	if _, err := c.provider.jobs.Get(ctx, c.config.Worker.NamePrefix+"-"+fingerprint(s.Reference)[:24]); !errors.Is(err, hostedexecution.ErrNotFound) {
		return hostedexecution.ErrUnavailable
	}
	d, err = c.provider.update(ctx, s.Reference, false, func(d *dispatch) error {
		if d.DeadlineProfile != "" && d.DeadlineProfile != c.config.profile() {
			return hostedexecution.ErrOwnership
		}
		d.DeadlineProfile, d.DeadlineRemoving = c.config.profile(), true
		return nil
	})
	if err != nil {
		return err
	}
	if c.provider.now().Before(d.DeadlineCheckAfter) {
		return hostedexecution.ErrUnavailable
	}
	actual, err := c.get(ctx, s)
	if errors.Is(err, hostedexecution.ErrNotFound) {
		if d.DeadlineCreateAttempted && !d.DeadlineCreateRejected && !d.DeadlineSeen {
			return hostedexecution.ErrUnavailable
		}
		return c.removed(ctx, s)
	}
	if err != nil {
		return err
	}
	if !d.DeadlineCreateAttempted || d.DeadlineCreateRejected {
		return hostedexecution.ErrOwnership
	}
	if err := c.config.verify(s, actual, false); err != nil {
		return err
	}
	if _, err := c.provider.update(ctx, s.Reference, false, func(d *dispatch) error {
		if c.provider.now().Before(d.DeadlineCheckAfter) || d.DeadlineRemoved {
			return hostedexecution.ErrUnavailable
		}
		d.DeadlineSeen = true
		d.DeadlineCheckAfter = c.provider.now().UTC().Add(30 * time.Second)
		return nil
	}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, deleteErr := c.workflows.Delete(ctx, c.config.ResourceGroup, c.config.name(s), nil)
	if err := sdkError(deleteErr, false); err != nil && !errors.Is(err, hostedexecution.ErrNotFound) {
		return err
	}
	if _, err := c.get(ctx, s); !errors.Is(err, hostedexecution.ErrNotFound) {
		return hostedexecution.ErrUnavailable
	}
	return c.removed(ctx, s)
}

func (c *SDKDeadlines) removed(ctx context.Context, s DeadlineSpec) error {
	_, err := c.provider.update(ctx, s.Reference, false, func(d *dispatch) error { d.DeadlineRemoved = true; return nil })
	return err
}
