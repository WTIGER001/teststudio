package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const placementRefresh = 15 * time.Second

func (c *SDKPlacement) beginPlacement() (uint64, time.Time) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cacheEpoch++
	return c.cacheEpoch, c.now()
}

func (c *SDKPlacement) rememberPlacement(epoch uint64, started time.Time, err error) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if epoch != c.cacheEpoch {
		return
	}
	c.cacheUntil = time.Time{}
	if err == nil && !c.http.closed.Load() {
		c.cacheUntil = started.Add(placementRefresh)
	}
}

func (c *SDKPlacement) runningPlacement(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if ctx.Err() != nil || c.http.closed.Load() {
			return hostedexecution.ErrUnavailable
		}
		c.cacheMutex.Lock()
		if c.now().Before(c.cacheUntil) {
			c.cacheMutex.Unlock()
			return nil
		}
		pending := c.cacheRead
		if pending != nil {
			c.cacheMutex.Unlock()
			select {
			case <-ctx.Done():
				return hostedexecution.ErrUnavailable
			case <-pending:
				continue
			}
		}
		pending = make(chan struct{})
		c.cacheRead = pending
		epoch := c.cacheEpoch
		started := c.now()
		c.cacheMutex.Unlock()
		err := c.readRunningPlacement(ctx, config)
		c.cacheMutex.Lock()
		if epoch == c.cacheEpoch {
			if err == nil {
				c.cacheUntil = started.Add(placementRefresh)
			} else {
				c.cacheUntil = time.Time{}
			}
		}
		c.cacheRead = nil
		close(pending)
		c.cacheMutex.Unlock()
		return err
	}
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
