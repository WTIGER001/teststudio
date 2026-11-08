package recordingworker

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (w *Worker) fence(ctx context.Context) error {
	row, err := w.Store.Get(ctx, PoolBucket, w.ID)
	if err != nil {
		return err
	}
	pool, err := storage.Decode[Pool](row)
	if err != nil {
		return err
	}
	if pool.Instance != w.Instance || !w.now().Before(pool.Until) || w.drivers == nil {
		return errors.New("the recording worker lease is no longer current")
	}
	return nil
}

func (w *Worker) leases(ctx context.Context) ([]storage.Record, error) {
	var result []storage.Record
	cursor := ""
	for {
		page, err := w.Store.List(ctx, LeaseBucket, "", cursor, 100)
		if err != nil {
			return nil, err
		}
		for _, row := range page.Items {
			lease, err := storage.Decode[Lease](row)
			if err != nil {
				return nil, err
			}
			if lease.Worker == w.ID {
				result = append(result, row)
			}
		}
		if page.Cursor == "" {
			return result, nil
		}
		cursor = page.Cursor
	}
}

func (w *Worker) save(ctx context.Context, row storage.Record, lease Lease) (storage.Record, error) {
	change, err := storage.Put(LeaseBucket, lease.Binding.ID, lease.Binding.Project, row.Revision, lease)
	if err != nil {
		return row, err
	}
	if err := w.Store.Commit(ctx, change); err != nil {
		return row, err
	}
	return w.Store.Get(ctx, LeaseBucket, lease.Binding.ID)
}

func (w *Worker) start(ctx context.Context, b Binding) (Response, error) {
	if err := w.authorized(ctx, b); err != nil {
		return Response{}, err
	}
	if !w.now().Before(b.Deadline) || b.Deadline.Sub(w.now()) > recording.Lifetime+time.Second {
		return Response{}, errors.New("recording deadline is expired or exceeds its maximum lifetime")
	}
	target, err := w.target(b)
	if err != nil {
		return Response{}, err
	}
	if existing, err := w.Store.Get(ctx, LeaseBucket, b.ID); err == nil {
		lease, err := storage.Decode[Lease](existing)
		if err != nil {
			return Response{}, err
		}
		if lease.Binding != b || lease.Worker != w.ID {
			return Response{}, storage.ErrConflict
		}
		return Response{Snapshot: lease.Snapshot}, nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return Response{}, err
	}
	ref, err := w.Runtime.Plan(b, target)
	if err != nil {
		return Response{}, err
	}
	lease := Lease{Binding: b, Worker: w.ID, Instance: w.Instance, Reference: ref, Target: target, LastSeen: w.now(), CleanupPending: true, Snapshot: recording.Snapshot{State: "starting"}}
	var row storage.Record
	err = w.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
		poolRow, err := store.Get(ctx, PoolBucket, w.ID)
		if err != nil {
			return err
		}
		pool, err := storage.Decode[Pool](poolRow)
		if err != nil {
			return err
		}
		if pool.Instance != w.Instance || !w.now().Before(pool.Until) {
			return storage.ErrConflict
		}
		bound := &Worker{Store: store, ID: w.ID}
		rows, err := bound.leases(ctx)
		if err != nil {
			return err
		}
		count := 0
		for _, item := range rows {
			value, err := storage.Decode[Lease](item)
			if err != nil {
				return err
			}
			if value.CleanupPending {
				count++
			}
		}
		if count >= w.Capacity {
			return errors.New("all recording browsers are in use; try again after another recording finishes")
		}
		guard, _ := storage.Put(PoolBucket, w.ID, "", poolRow.Revision, pool)
		change, _ := storage.Put(LeaseBucket, b.ID, b.Project, 0, lease)
		if err := store.Commit(ctx, guard, change); err != nil {
			return err
		}
		row, err = store.Get(ctx, LeaseBucket, b.ID)
		return err
	})
	if err != nil {
		return Response{}, err
	}
	driver, startErr := w.Runtime.Start(ctx, b, ref, target)
	if driver != nil {
		w.drivers[b.ID] = driver
	}
	if startErr == nil {
		lease.Snapshot, startErr = driver.Read(ctx, b.ID)
	}
	if startErr != nil {
		lease.Snapshot.State, lease.Snapshot.Error = "failed", "The isolated recording browser could not start. Review the worker configuration and start a new recording."
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		_ = w.cleanup(cleanup, &lease)
	}
	if _, err := w.save(context.WithoutCancel(ctx), row, lease); err != nil {
		return Response{}, err
	}
	if startErr != nil {
		return Response{}, errors.New(lease.Snapshot.Error)
	}
	return Response{Snapshot: lease.Snapshot}, nil
}

func (w *Worker) interact(ctx context.Context, row storage.Record, lease Lease, operation string, action *recording.Action) (Response, error) {
	driver := w.drivers[lease.Binding.ID]
	duplicate := false
	if operation == "input" {
		if action == nil || !action.Valid() {
			return Response{}, errors.New("unsupported recording input")
		}
		data, _ := json.Marshal(action)
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		duplicate = action.Sequence == lease.Sequence && hash == lease.InputHash
		if !duplicate {
			if lease.Pending || action.Sequence != lease.Sequence+1 {
				return Response{}, errors.New("refresh the browser view before continuing; this input sequence is no longer current")
			}
			lease.Sequence, lease.InputHash, lease.Pending = action.Sequence, hash, true
			var err error
			row, err = w.save(ctx, row, lease)
			if err != nil {
				return Response{}, err
			}
		}
	}
	var view recording.View
	var err error
	if operation == "view" || duplicate {
		view, err = driver.View(ctx, lease.Binding.ID)
	} else {
		view, err = driver.Interact(ctx, lease.Binding.ID, *action)
	}
	if err != nil {
		return Response{}, err
	}
	if lease.Pending && view.Sequence != lease.Sequence {
		return Response{}, errors.New("the previous input is unresolved; finish or restart this recording")
	}
	lease.Pending = false
	lease.LastSeen = w.now()
	lease.Snapshot, err = driver.Read(ctx, lease.Binding.ID)
	if err != nil {
		return Response{}, err
	}
	if _, err := w.save(ctx, row, lease); err != nil {
		return Response{}, err
	}
	return Response{Snapshot: lease.Snapshot, View: &view}, nil
}

func (w *Worker) cleanup(ctx context.Context, lease *Lease) error {
	if err := w.Runtime.Cleanup(ctx, lease.Binding, lease.Reference); err != nil {
		return err
	}
	delete(w.drivers, lease.Binding.ID)
	lease.CleanupPending = false
	return nil
}
