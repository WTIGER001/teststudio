package recordingworker

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (w *Worker) Maintain(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.fence(ctx); err != nil {
		return err
	}
	poolRow, err := w.Store.Get(ctx, PoolBucket, w.ID)
	if err != nil {
		return err
	}
	change, _ := storage.Put(PoolBucket, w.ID, "", poolRow.Revision, Pool{Instance: w.Instance, Until: w.now().Add(OwnerLifetime)})
	if err := w.Store.Commit(ctx, change); err != nil {
		return err
	}
	rows, err := w.leases(ctx)
	if err != nil {
		return err
	}
	var failures error
	for _, row := range rows {
		lease, err := storage.Decode[Lease](row)
		if err != nil {
			return err
		}
		if !lease.CleanupPending {
			continue
		}
		if !lease.Closing && lease.Snapshot.State == "recording" && w.now().Before(lease.Binding.Deadline) && w.now().Sub(lease.LastSeen) < w.IdleLimit && w.authorized(ctx, lease.Binding) == nil {
			continue
		}
		if driver := w.drivers[lease.Binding.ID]; driver != nil {
			if captured, err := driver.Read(ctx, lease.Binding.ID); err == nil && captured.Code != "" {
				lease.Snapshot = captured
			}
			if captured, err := driver.Stop(ctx, lease.Binding.ID); err == nil && captured.Code != "" {
				lease.Snapshot = captured
			}
		}
		if !lease.Closing || lease.Snapshot.State != "ready" {
			lease.Snapshot.State, lease.Snapshot.Error = "failed", "Recording stopped after its time limit, a disconnected browser, or a change in access. Captured steps remain available."
		}
		cleanupErr := w.cleanup(ctx, &lease)
		_, saveErr := w.save(ctx, row, lease)
		failures = errors.Join(failures, cleanupErr, saveErr)
	}
	return failures
}

func (w *Worker) StartMaintenance(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTicker(5 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				cycle, cancelCycle := context.WithTimeout(ctx, 45*time.Second)
				err := w.Maintain(cycle)
				cancelCycle()
				if err != nil {
					if w.Ready(ctx) {
						continue
					}
					cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 45*time.Second)
					_ = w.Close(cleanup)
					cancel()
					return
				}
			}
		}
	}()
	return done
}

func (w *Worker) Close(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	rows, err := w.leases(ctx)
	if err != nil {
		return err
	}
	var result error
	for _, row := range rows {
		lease, err := storage.Decode[Lease](row)
		if err != nil {
			return err
		}
		if lease.Instance != w.Instance || !lease.CleanupPending {
			continue
		}
		if driver := w.drivers[lease.Binding.ID]; driver != nil {
			if snapshot, err := driver.Stop(ctx, lease.Binding.ID); err == nil && snapshot.Code != "" {
				lease.Snapshot = snapshot
			}
		}
		lease.Snapshot.State, lease.Snapshot.Error = "failed", "The recording worker stopped. Captured steps remain available."
		cleanupErr := w.cleanup(ctx, &lease)
		_, saveErr := w.save(ctx, row, lease)
		result = errors.Join(result, cleanupErr, saveErr)
	}
	if result != nil {
		return result
	}
	poolRow, err := w.Store.Get(ctx, PoolBucket, w.ID)
	if err != nil {
		return err
	}
	pool, err := storage.Decode[Pool](poolRow)
	if err != nil {
		return err
	}
	if pool.Instance != w.Instance {
		return nil
	}
	change, _ := storage.Put(PoolBucket, w.ID, "", poolRow.Revision, Pool{Instance: w.Instance, Until: w.now()})
	w.drivers = nil
	return w.Store.Commit(ctx, change)
}
