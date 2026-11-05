package azuresandbox

import (
	"context"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const metadataRefresh = 15 * time.Second

type metadataCache struct {
	mutex   sync.Mutex
	until   time.Time
	epoch   uint64
	pending chan struct{}
}

func (c *metadataCache) begin(now func() time.Time) (uint64, time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.epoch++
	return c.epoch, now()
}

func (c *metadataCache) remember(epoch uint64, started time.Time, err error, closed func() bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if epoch != c.epoch {
		return
	}
	c.until = time.Time{}
	if err == nil && !closed() {
		c.until = started.Add(metadataRefresh)
	}
}

func (c *metadataCache) check(ctx context.Context, now func() time.Time, closed func() bool, read func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if ctx.Err() != nil || closed() {
			return hostedexecution.ErrUnavailable
		}
		c.mutex.Lock()
		if now().Before(c.until) {
			c.mutex.Unlock()
			return nil
		}
		pending := c.pending
		if pending != nil {
			c.mutex.Unlock()
			select {
			case <-ctx.Done():
				return hostedexecution.ErrUnavailable
			case <-pending:
				continue
			}
		}
		pending = make(chan struct{})
		c.pending = pending
		epoch, started := c.epoch, now()
		c.mutex.Unlock()
		err := read(ctx)
		c.mutex.Lock()
		if epoch == c.epoch {
			c.until = time.Time{}
			if err == nil && !closed() {
				c.until = started.Add(metadataRefresh)
			}
		}
		c.pending = nil
		close(pending)
		c.mutex.Unlock()
		return err
	}
}
