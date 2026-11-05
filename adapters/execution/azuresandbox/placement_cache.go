package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const placementRefresh = metadataRefresh

func (c *SDKPlacement) beginPlacement() (uint64, time.Time) {
	return c.cache.begin(c.now)
}

func (c *SDKPlacement) rememberPlacement(epoch uint64, started time.Time, err error) {
	c.cache.remember(epoch, started, err, c.http.closed.Load)
}

func (c *SDKPlacement) runningPlacement(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, func(ctx context.Context) error { return c.readRunningPlacement(ctx, config) })
}

func (c *SDKPlacement) readRunningPlacement(ctx context.Context, config Config) error {
	ids := []string{config.groupID(), config.connectionID(), c.settings.Placement.Subnet}
	values, err := c.observe(ctx, config, ids)
	if err != nil {
		return err
	}
	if err := c.settings.verifyGroup(values[0]); err != nil {
		return err
	}
	if err := c.settings.verifyConnection(values[1]); err != nil {
		return err
	}
	return c.settings.verifySubnet(values[2])
}
