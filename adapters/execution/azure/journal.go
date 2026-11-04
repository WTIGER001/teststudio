package azureworker

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func terminal(s hostedexecution.Status) bool {
	return s.Phase == "succeeded" || s.Phase == "failed" || s.Phase == "stopped"
}

func (p *Provider) load(ctx context.Context, ref execution.Reference) (dispatch, storage.Record, *Plan, error) {
	row, err := p.store.Get(ctx, Bucket, ref.ID)
	if err != nil {
		return dispatch{}, row, nil, err
	}
	d, err := storage.Decode[dispatch](row)
	if err != nil || row.Project != ref.Project || d.Proof.Reference != ref || row.ID != ref.ID {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	plan, err := Restore(p.config, d.Proof)
	if err != nil || d.DefinitionSeen && (!d.CreateAttempted || d.CreateRejected || d.CreationBlocked) || d.DeadlineAttempted && !d.DefinitionSeen || d.DeadlineVerified && !d.DeadlineAttempted || d.StartAttempted && !d.DeadlineVerified || d.StartRejected && !d.StartAttempted {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	if d.Receipt != nil && (!d.StartAttempted || d.StartRejected || plan.VerifyReceipt(*d.Receipt) != nil) {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	if d.Terminal != nil {
		s := d.Terminal
		if !terminal(*s) || s.Reference != ref || s.Image != d.Proof.Image || !s.Deadline.Equal(d.Proof.Deadline) || s.AdmissionPending || s.ExitCode != nil || s.NativeCompletion != (s.Phase == "succeeded") || d.StartAttempted && !d.StartRejected && d.Receipt == nil {
			return d, row, nil, hostedexecution.ErrOwnership
		}
	}
	return d, row, plan, nil
}

func sameJob(proof Proof, job hostedexecution.Job) bool {
	return proof.Reference == job.Reference && proof.Image == job.Image && proof.Runner == job.Runner && proof.Operation == job.Operation && proof.Deadline.Equal(job.Ticket.Deadline) && proof.TicketHash == fingerprint(job.Ticket.Token) && job.Ticket.Execution == job.Reference.ID
}

func (p *Provider) exchange(ctx context.Context, d dispatch) (workerexchange.Record, storage.Record, error) {
	row, err := p.store.Get(ctx, workerexchange.Bucket, d.Proof.Reference.ID)
	if err != nil {
		return workerexchange.Record{}, row, hostedexecution.ErrUnavailable
	}
	r, err := storage.Decode[workerexchange.Record](row)
	if err != nil || r.Reference != d.Proof.Reference || row.Project != r.Reference.Project || r.Operation != d.Proof.Operation || !r.DispatchGated || !r.Deadline.Equal(d.Proof.Deadline) {
		return r, row, hostedexecution.ErrOwnership
	}
	return r, row, nil
}

func (p *Provider) reserve(ctx context.Context, plan *Plan) (dispatch, bool, error) {
	d := dispatch{Proof: plan.Proof(), CreatedAt: p.now().UTC()}
	r, guard, err := p.exchange(ctx, d)
	if err != nil || r.Revoked || r.DispatchReady {
		return d, false, hostedexecution.ErrUnavailable
	}
	change, err := storage.Put(Bucket, d.Proof.Reference.ID, d.Proof.Reference.Project, 0, d)
	if err != nil {
		return d, false, hostedexecution.ErrUnavailable
	}
	err = p.store.Commit(ctx, change, storage.Change{Record: guard, Assert: true})
	if err == nil {
		return d, true, nil
	}
	if !errors.Is(err, storage.ErrConflict) {
		return d, false, hostedexecution.ErrUnavailable
	}
	d, _, _, err = p.load(ctx, plan.job.Reference)
	if err != nil || !sameJob(d.Proof, plan.job) {
		return d, false, hostedexecution.ErrOwnership
	}
	return d, false, nil
}

func (p *Provider) update(ctx context.Context, ref execution.Reference, dispatching bool, mutate func(*dispatch) error) (dispatch, error) {
	for range 8 {
		d, row, _, err := p.load(ctx, ref)
		if err != nil {
			return d, err
		}
		changes := []storage.Change{}
		if dispatching {
			r, guard, err := p.exchange(ctx, d)
			if err != nil || r.Revoked || r.DispatchReady || !p.now().Before(r.Deadline) || d.Stopping || d.Removed || d.Terminal != nil {
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
		err = p.store.Commit(ctx, append(changes, change)...)
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

func controlError(err error) error {
	for _, known := range []error{hostedexecution.ErrUnavailable, hostedexecution.ErrNotFound, hostedexecution.ErrOwnership, hostedexecution.ErrRejected, hostedexecution.ErrConfiguration} {
		if errors.Is(err, known) {
			return known
		}
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
