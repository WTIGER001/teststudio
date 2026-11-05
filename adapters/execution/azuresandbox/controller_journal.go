package azuresandbox

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func validAnyReference(ref execution.Reference) bool {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			if validReference(ref, runner, operation) {
				return true
			}
		}
	}
	return false
}

func (c *Controller) load(ctx context.Context, ref execution.Reference) (dispatch, storage.Record, *Plan, error) {
	if !validAnyReference(ref) {
		return dispatch{}, storage.Record{}, nil, hostedexecution.ErrOwnership
	}
	row, err := c.store.Get(ctx, DispatchBucket, ref.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err = hostedexecution.ErrNotFound
		}
		return dispatch{}, row, nil, err
	}
	d, err := storage.Decode[dispatch](row)
	if err != nil || row.Project != ref.Project || d.Proof.Reference != ref || row.ID != ref.ID || d.CreatedAt.IsZero() || !checksum.MatchString(d.InputHash) {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	plan, err := Restore(c.config, d.Proof)
	if err != nil || d.DeadlineVerified && !d.DeadlineAttempted || d.CreateAttempted && !d.DeadlineVerified || d.CreateRejected && !d.CreateAttempted || d.CreationBlocked && d.CreateAttempted {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	if d.Receipt != "" && (!d.CreateAttempted || d.CreateRejected || !identifier.MatchString(d.Receipt)) || d.LastState != "" && (d.Receipt == "" || !knownState(d.LastState)) || d.Quiesced != quiescent(d.LastState) {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	if d.DeleteAttempted && (!d.Stopping || !d.Quiesced) || d.InstanceAbsent && (!d.Quiesced || d.Receipt == "") || d.DeadlineRemoving && (!d.DeadlineAttempted || !d.Stopping || !settled(d)) || d.DeadlineRemoved && !d.DeadlineRemoving || !d.StopAfter.IsZero() && (!d.Stopping || d.Receipt == "") || !d.DeleteAfter.IsZero() && !d.DeleteAttempted {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	if d.Removed && (!settled(d) || d.DeadlineAttempted && !d.DeadlineRemoved) {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	if d.TimerCreateAttempted && (!d.DeadlineAttempted || !checksum.MatchString(d.TimerProfile)) || d.TimerCreateRejected && !d.TimerCreateAttempted || d.TimerSeen && (!d.TimerCreateAttempted || d.TimerCreateRejected) || !d.TimerCheckAfter.IsZero() && (!d.TimerSeen || !d.DeadlineRemoving) {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	return d, row, plan, nil
}

func settled(d dispatch) bool {
	return !d.CreateAttempted || d.CreateRejected || d.InstanceAbsent
}

func (c *Controller) transport(ctx context.Context, d dispatch) (workerexchange.Record, storage.Record, error) {
	ref := d.Proof.Reference
	row, err := c.store.Get(ctx, workerexchange.Bucket, ref.ID)
	if err != nil {
		return workerexchange.Record{}, row, hostedexecution.ErrUnavailable
	}
	r, err := storage.Decode[workerexchange.Record](row)
	if err != nil || row.Project != ref.Project || r.Reference != ref || !r.DispatchGated || r.Operation != d.Proof.Operation || !r.Deadline.Equal(d.Proof.Deadline) || d.InputHash != "" && d.InputHash != r.InputHash || !matchesWork(d.Proof, r) {
		return r, row, hostedexecution.ErrOwnership
	}
	return r, row, nil
}

func (c *Controller) reserve(ctx context.Context, plan *Plan) (dispatch, error) {
	d := dispatch{Proof: plan.Proof(), CreatedAt: c.now().UTC()}
	r, guard, err := c.transport(ctx, d)
	if err != nil {
		return d, err
	}
	if r.Revoked || r.DispatchReady || !c.now().Before(r.Deadline) {
		return d, hostedexecution.ErrRejected
	}
	d.InputHash = r.InputHash
	change, err := storage.Put(DispatchBucket, d.Proof.Reference.ID, d.Proof.Reference.Project, 0, d)
	if err != nil {
		return d, hostedexecution.ErrUnavailable
	}
	err = c.store.Commit(ctx, change, storage.Change{Record: guard, Assert: true})
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, storage.ErrConflict) {
		return d, hostedexecution.ErrUnavailable
	}
	actual, _, _, err := c.load(ctx, d.Proof.Reference)
	if err == nil && actual.Proof.Fingerprint != d.Proof.Fingerprint {
		err = hostedexecution.ErrOwnership
	}
	return actual, err
}

func (c *Controller) update(ctx context.Context, ref execution.Reference, dispatching bool, mutate func(*dispatch) error) (dispatch, error) {
	for range 8 {
		d, row, _, err := c.load(ctx, ref)
		if err != nil {
			return d, err
		}
		changes := []storage.Change{}
		if dispatching {
			r, guard, err := c.transport(ctx, d)
			if err != nil {
				return d, err
			}
			if r.Revoked || r.DispatchReady || !c.now().Before(r.Deadline) || d.Stopping || d.Removed || d.CreationBlocked || d.CreateRejected {
				return d, hostedexecution.ErrUnavailable
			}
			changes = append(changes, storage.Change{Record: guard, Assert: true})
		}
		if err := mutate(&d); err != nil {
			return d, err
		}
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, d)
		if err != nil {
			return d, hostedexecution.ErrUnavailable
		}
		err = c.store.Commit(ctx, append(changes, change)...)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		if err != nil {
			return d, hostedexecution.ErrUnavailable
		}
		return d, nil
	}
	return dispatch{}, hostedexecution.ErrUnavailable
}
