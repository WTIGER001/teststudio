package azureworker

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (p *Provider) Stop(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, plan, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		return hostedexecution.ErrNotFound
	}
	if err != nil || d.Terminal != nil || d.Removed {
		return err
	}
	d, err = p.update(ctx, ref, false, func(current *dispatch) error { current.Stopping = true; return nil })
	if err != nil || !d.StartAttempted || d.StartRejected {
		return err
	}
	definition, err := p.jobs.Get(ctx, plan.Name())
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := plan.VerifyRetainedDefinition(definition); err != nil {
		return err
	}
	actual, receipt, err := p.readExecution(ctx, d, plan)
	if err != nil {
		return err
	}
	observed, observationErr := plan.VerifyExecution(receipt, actual)
	status := statusFor(d, observed.Phase)
	if observationErr == nil && terminal(status) {
		return p.remember(ctx, ref, status, false)
	}
	return p.jobs.Stop(ctx, plan.Name(), value(actual.Name))
}

func (p *Provider) authorizeCleanup(ctx context.Context, d dispatch) error {
	r, _, err := p.exchange(ctx, d)
	if err != nil || !r.Revoked {
		return hostedexecution.ErrUnavailable
	}
	ref := d.Proof.Reference
	if d.Proof.Operation == "discover" {
		if p.discoveryCleanup == nil {
			return hostedexecution.ErrUnavailable
		}
		return p.discoveryCleanup(ctx, ref)
	}
	entry, _, err := (workerstate.Ledger{Store: p.store}).Get(ctx, ref.Run, ref.ID)
	if err != nil || entry.Reference != ref || !entry.ResultSaved || !entry.EvidenceComplete && !entry.EvidenceAbandoned {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (p *Provider) Remove(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, plan, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		return hostedexecution.ErrNotFound
	}
	if err != nil || d.Removed {
		return err
	}
	if err := p.authorizeCleanup(ctx, d); err != nil {
		return err
	}
	if d.CreateAttempted && !d.CreateRejected && !d.CreationBlocked {
		if p.now().Before(d.DeleteCheckAfter) {
			return hostedexecution.ErrUnavailable
		}
		definition, lookupErr := p.jobs.Get(ctx, plan.Name())
		if errors.Is(lookupErr, hostedexecution.ErrNotFound) {
			if !d.DefinitionSeen || d.StartAttempted && !d.StartRejected && d.Terminal == nil {
				return hostedexecution.ErrUnavailable
			}
		} else {
			if lookupErr != nil {
				return hostedexecution.ErrUnavailable
			}
			if err := plan.VerifyRetainedDefinition(definition); err != nil {
				return err
			}
			items, err := p.jobs.Executions(ctx, plan.Name())
			if err != nil {
				return hostedexecution.ErrUnavailable
			}
			if len(items) > 1 || len(items) > 0 && (!d.StartAttempted || d.StartRejected) {
				return hostedexecution.ErrOwnership
			}
			if d.StartAttempted && !d.StartRejected {
				if len(items) == 0 && d.Terminal == nil {
					return hostedexecution.ErrUnavailable
				}
				if len(items) == 1 {
					if d.Receipt == nil {
						if _, _, err := p.findExecution(ctx, d, plan); err != nil {
							return err
						}
						d, _, _, err = p.load(ctx, ref)
						if err != nil {
							return err
						}
					}
					observed, err := plan.VerifyExecution(*d.Receipt, items[0])
					if err != nil {
						return err
					}
					status := statusFor(d, observed.Phase)
					if !terminal(status) {
						return hostedexecution.ErrUnavailable
					}
					if err := p.remember(ctx, ref, status, false); err != nil {
						return err
					}
				}
			}
			if _, err := p.update(ctx, ref, false, func(current *dispatch) error {
				current.DefinitionSeen, current.Deleting = true, true
				current.DeleteCheckAfter = p.now().UTC().Add(30 * time.Second)
				return nil
			}); err != nil {
				return err
			}
			if err := p.jobs.Delete(ctx, plan.Name()); err != nil && !errors.Is(err, hostedexecution.ErrNotFound) {
				return hostedexecution.ErrUnavailable
			}
			if _, err := p.jobs.Get(ctx, plan.Name()); !errors.Is(err, hostedexecution.ErrNotFound) {
				return hostedexecution.ErrUnavailable
			}
		}
	}
	if d.DeadlineAttempted {
		if err := p.deadlines.Remove(ctx, plan.deadlineSpec()); err != nil {
			return hostedexecution.ErrUnavailable
		}
	}
	_, err = p.update(ctx, ref, false, func(current *dispatch) error { current.Removed = true; return nil })
	return err
}
