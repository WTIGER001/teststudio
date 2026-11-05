package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *Provider) stopOwned(ctx context.Context, d launchIntent) error {
	if d.Removed {
		return nil
	}
	if err := p.controller.exchange.Revoke(ctx, d.Proof.Reference); err != nil {
		return hostedexecution.ErrUnavailable
	}
	current, _, _, err := p.controller.load(ctx, d.Proof.Reference)
	if err == hostedexecution.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if !d.Proof.matches(current.Proof) {
		return hostedexecution.ErrOwnership
	}
	err = p.controller.Stop(ctx, d.Proof.Reference)
	if err == hostedexecution.ErrUnavailable {
		latest, _, _, lookup := p.controller.load(ctx, d.Proof.Reference)
		if lookup == nil && d.Proof.matches(latest.Proof) && latest.Stopping && latest.Receipt != "" && p.controller.now().Before(latest.StopAfter) {
			return nil
		}
	}
	return err
}

func (p *Provider) Stop(ctx context.Context, ref execution.Reference) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	d, err := p.updateIntent(ctx, ref, func(current *launchIntent) error { current.Stopping = true; return nil })
	if err != nil {
		return err
	}
	return p.stopOwned(ctx, d)
}

func (p *Provider) stoppedStatus(ctx context.Context, d launchIntent) (hostedexecution.Status, error) {
	status := d.status()
	status.AdmissionPending = false
	status.Phase = "stopped"
	if err := p.stopOwned(ctx, d); err != nil {
		return status, err
	}
	current, _, _, err := p.controller.load(ctx, d.Proof.Reference)
	if err == hostedexecution.ErrNotFound {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	if !d.Proof.matches(current.Proof) || d.Removed && !current.Removed {
		return status, hostedexecution.ErrOwnership
	}
	if !current.CreateAttempted || current.CreateRejected || current.CreationBlocked {
		return status, nil
	}
	return p.controller.Status(ctx, d.Proof.Reference)
}

func (p *Provider) Remove(ctx context.Context, ref execution.Reference) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	d, _, err := p.loadIntent(ctx, ref)
	if err != nil || d.Removed {
		return err
	}
	if err := p.controller.exchange.CheckCleanup(ctx, ref); err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := p.Stop(ctx, ref); err != nil {
		return err
	}
	d, _, err = p.loadIntent(ctx, ref)
	if err != nil {
		return err
	}
	current, _, _, err := p.controller.load(ctx, ref)
	if err == nil {
		if !d.Proof.matches(current.Proof) {
			return hostedexecution.ErrOwnership
		}
		if err := p.controller.Remove(ctx, ref); err != nil {
			return err
		}
	} else if err != hostedexecution.ErrNotFound {
		return err
	}
	_, err = p.updateIntent(ctx, ref, func(current *launchIntent) error {
		if !current.Stopping {
			return hostedexecution.ErrUnavailable
		}
		current.Removed = true
		return nil
	})
	return err
}
