package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) Ensure(ctx context.Context, job hostedexecution.Job) error {
	if p.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if !p.validJob(job) {
		return hostedexecution.ErrConfiguration
	}
	d, _, err := p.loadIntent(ctx, job.Reference)
	if err == nil {
		if !sameLaunch(d.Proof, job) {
			return hostedexecution.ErrOwnership
		}
		if d.Stopping || d.Removed || d.Rejected {
			return hostedexecution.ErrRejected
		}
	} else {
		if err != hostedexecution.ErrNotFound {
			return err
		}
		image, err := p.InspectImage(ctx, job.Image)
		if err != nil {
			return err
		}
		if image.Project != job.Reference.Project || image.Runner != job.Runner {
			return hostedexecution.ErrOwnership
		}
		d, err = p.reserveIntent(ctx, job, image)
		if err != nil {
			return err
		}
	}
	_, err = p.Observe(ctx, d.Proof.Reference)
	if err == hostedexecution.ErrUnavailable {
		return nil
	}
	return err
}

func (p *Provider) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	if p.closed.Load() {
		return hostedexecution.Status{}, hostedexecution.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	d, _, err := p.loadIntent(ctx, ref)
	if err != nil {
		return hostedexecution.Status{}, err
	}
	status := d.status()
	if d.Stopping || d.Removed {
		return p.stoppedStatus(ctx, d)
	}
	work, _, err := p.controller.transport(ctx, dispatch{Proof: d.Proof.executionProof(), InputHash: d.Proof.InputHash})
	if err != nil {
		return status, err
	}
	if work.Revoked || !p.controller.now().Before(work.Deadline) {
		if err := p.Stop(ctx, ref); err != nil {
			return status, err
		}
		d, _, err = p.loadIntent(ctx, ref)
		if err != nil {
			return status, err
		}
		return p.stoppedStatus(ctx, d)
	}
	if d.Rejected {
		return status, nil
	}
	current, _, _, err := p.controller.load(ctx, ref)
	if err == nil {
		if !d.Proof.matches(current.Proof) {
			return status, hostedexecution.ErrOwnership
		}
		if !current.CreateAttempted && !current.CreateRejected && !current.CreationBlocked && !current.Stopping {
			if err := p.controller.Advance(ctx, ref); err != nil && err != hostedexecution.ErrUnavailable && err != hostedexecution.ErrRejected {
				return status, err
			}
		}
		actual, err := p.controller.Status(ctx, ref)
		if err == hostedexecution.ErrUnavailable && !work.DispatchReady {
			return status, nil
		}
		if err == hostedexecution.ErrNotFound {
			status.Phase = "failed"
			status.AdmissionPending = false
			return status, nil
		}
		return actual, err
	}
	if err != hostedexecution.ErrNotFound {
		return status, err
	}
	if work.DispatchReady {
		return status, hostedexecution.ErrOwnership
	}
	return p.prepareLaunch(ctx, d)
}

func (p *Provider) prepareLaunch(ctx context.Context, d launchIntent) (hostedexecution.Status, error) {
	status := d.status()
	ticket, err := p.controller.exchange.ResumeDispatch(ctx, d.Proof.Reference)
	if err != nil {
		return status, hostedexecution.ErrUnavailable
	}
	if ticket.Execution != d.Proof.Reference.ID || hash(ticket.Token) != d.Proof.TicketHash || !ticket.Deadline.Equal(d.Proof.Deadline) {
		return status, hostedexecution.ErrOwnership
	}
	job := hostedexecution.Job{Reference: d.Proof.Reference, Image: d.Proof.Image.Reference, Runner: d.Proof.Image.Runner, Operation: d.Proof.Operation, ExchangeOrigin: p.controller.config.ExchangeOrigin, Ticket: ticket}
	var image DiskImage
	if job.Operation == "discover" {
		image, err = p.images.Prepare(ctx, job, d.Proof.Image)
	} else {
		image, err = p.images.Lookup(ctx, d.Proof.Image)
	}
	if err == hostedexecution.ErrRejected || err == hostedexecution.ErrNotFound && job.Operation == "" {
		_, saveErr := p.updateIntent(ctx, job.Reference, func(current *launchIntent) error { current.Rejected = true; return nil })
		status.Phase = "failed"
		status.AdmissionPending = false
		return status, saveErr
	}
	if err == hostedexecution.ErrUnavailable {
		return status, nil
	}
	if err != nil {
		return status, err
	}
	if !readyImage(image.State) {
		return status, nil
	}
	plan, err := NewPlan(p.controller.config, job, image, d.Proof.Image, p.controller.now())
	if err != nil {
		return status, err
	}
	latest, row, err := p.loadIntent(ctx, job.Reference)
	if err != nil {
		return status, err
	}
	if latest.Stopping || latest.Removed || latest.Rejected {
		return status, hostedexecution.ErrUnavailable
	}
	guards, err := dispatchGuards(ctx, p.controller.exchange, d.Proof.executionProof(), ticket.Token, d.Proof.InputHash, p.controller.now())
	if err != nil {
		return status, err
	}
	if err := p.controller.store.Commit(ctx, append(guards, storage.Change{Record: row, Assert: true})...); err != nil {
		return status, hostedexecution.ErrUnavailable
	}
	err = p.controller.Ensure(ctx, plan)
	if err == hostedexecution.ErrUnavailable {
		return status, nil
	}
	if err != nil && err != hostedexecution.ErrRejected {
		return status, err
	}
	actual, err := p.controller.Status(ctx, job.Reference)
	if err == hostedexecution.ErrNotFound {
		status.Phase = "failed"
		status.AdmissionPending = false
		return status, nil
	}
	if err == hostedexecution.ErrUnavailable {
		return status, nil
	}
	return actual, err
}
