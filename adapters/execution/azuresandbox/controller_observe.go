package azuresandbox

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (c *Controller) remember(ctx context.Context, plan *Plan, actual Snapshot) error {
	if err := plan.Verify(actual); err != nil {
		return err
	}
	if !knownState(actual.State) {
		return hostedexecution.ErrUnavailable
	}
	prior, row, _, err := c.load(ctx, plan.proof.Reference)
	if err != nil {
		return err
	}
	if prior.Receipt == actual.ID && prior.LastState == actual.State && !prior.InstanceAbsent && !prior.Removed {
		if err := c.store.Commit(ctx, storage.Change{Record: row, Assert: true}); err != nil {
			return hostedexecution.ErrUnavailable
		}
		return nil
	}
	_, err = c.update(ctx, plan.proof.Reference, false, func(d *dispatch) error {
		if !d.CreateAttempted || d.CreateRejected || d.CreationBlocked || d.Receipt != "" && d.Receipt != actual.ID || d.InstanceAbsent || d.Removed || d.Quiesced && !quiescent(actual.State) {
			return hostedexecution.ErrOwnership
		}
		d.Receipt, d.LastState = actual.ID, actual.State
		d.Quiesced = quiescent(actual.State)
		return nil
	})
	return err
}

func (c *Controller) resolve(ctx context.Context, d dispatch, plan *Plan) (Snapshot, error) {
	var actual Snapshot
	if d.Receipt != "" {
		err := bounded(ctx, func(ctx context.Context) error {
			var err error
			actual, err = c.instances.Read(ctx, d.Receipt)
			return err
		})
		if err != nil {
			return actual, err
		}
		if actual.ID != d.Receipt {
			return actual, hostedexecution.ErrOwnership
		}
	} else {
		var items []Snapshot
		err := bounded(ctx, func(ctx context.Context) error {
			var err error
			items, err = c.instances.Inventory(ctx)
			return err
		})
		if err != nil {
			return actual, hostedexecution.ErrUnavailable
		}
		matches := 0
		for _, item := range items {
			if plan.Verify(item) == nil {
				actual = item
				matches++
			}
		}
		if matches > 1 {
			return Snapshot{}, hostedexecution.ErrOwnership
		}
		if matches == 0 {
			return Snapshot{}, hostedexecution.ErrUnavailable
		}
	}
	return actual, c.remember(ctx, plan, actual)
}

func (c *Controller) Observe(ctx context.Context, ref execution.Reference) (state DispatchState, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, plan, err := c.load(ctx, ref)
	if err != nil {
		return state, err
	}
	state = DispatchState{Reference: ref, AdmissionPending: true, CreateAttempted: d.CreateAttempted, Rejected: d.CreateRejected || d.CreationBlocked, Quiesced: d.Quiesced, Removed: d.Removed}
	state.CommandAttempted, state.CommandUnknown, state.CommandRejected = d.CommandAttempted, d.CommandUnknown, d.CommandRejected
	state.CommandExitCode, state.CommandSuppressed = d.CommandExitCode, d.CommandSuppressed
	if d.Removed || state.Rejected || !d.CreateAttempted {
		return state, nil
	}
	actual, err := c.resolve(ctx, d, plan)
	if err != nil {
		return state, err
	}
	state.Instance, state.Quiesced = actual, quiescent(actual.State)
	if d.Stopping || state.Quiesced || actual.State != "Running" {
		return state, nil
	}
	r, _, err := c.transport(ctx, d)
	if err != nil {
		return state, err
	}
	if r.Revoked || !c.now().Before(r.Deadline) {
		return state, nil
	}
	if err := bounded(ctx, func(ctx context.Context) error { return c.deadlines.Verify(ctx, c.deadlineSpec(plan)) }); err != nil {
		if err == hostedexecution.ErrOwnership {
			return state, err
		}
		return state, nil
	}
	if err := bounded(ctx, func(ctx context.Context) error { return c.admission.VerifyInstance(ctx, c.config, actual) }); err != nil {
		if err == hostedexecution.ErrOwnership {
			return state, err
		}
		return state, nil
	}
	for range 8 {
		latest, row, _, err := c.load(ctx, ref)
		if err != nil {
			return state, err
		}
		transport, guard, err := c.transport(ctx, latest)
		if err != nil {
			return state, err
		}
		if latest.Stopping || latest.Removed || latest.Quiesced || latest.Receipt != actual.ID || transport.Revoked || !c.now().Before(transport.Deadline) {
			return state, nil
		}
		err = c.store.Commit(ctx, storage.Change{Record: row, Assert: true}, storage.Change{Record: guard, Assert: true})
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		if err != nil {
			return state, hostedexecution.ErrUnavailable
		}
		state.AdmissionPending = false
		return state, nil
	}
	return state, hostedexecution.ErrUnavailable
}
