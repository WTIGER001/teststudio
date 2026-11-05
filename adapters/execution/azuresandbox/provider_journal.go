package azuresandbox

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) loadIntent(ctx context.Context, ref execution.Reference) (launchIntent, storage.Record, error) {
	if !validAnyReference(ref) {
		return launchIntent{}, storage.Record{}, hostedexecution.ErrOwnership
	}
	row, err := p.controller.store.Get(ctx, IntentBucket, ref.ID)
	if errors.Is(err, storage.ErrNotFound) {
		err = hostedexecution.ErrNotFound
	}
	if err != nil {
		return launchIntent{}, row, controlError(err)
	}
	d, err := storage.Decode[launchIntent](row)
	proof := d.Proof
	if err != nil || row.ID != ref.ID || row.Project != ref.Project || proof.Reference != ref || !validReference(ref, proof.Image.Runner, proof.Operation) || d.CreatedAt.IsZero() || proof.Deadline.IsZero() || proof.Deadline.Year() < 2000 || proof.Deadline.Year() > 2200 || !checksum.MatchString(proof.TicketHash) || !checksum.MatchString(proof.InputHash) || proof.Fingerprint != p.fingerprint(proof) || d.Removed && !d.Stopping {
		return d, row, hostedexecution.ErrOwnership
	}
	image, err := NewImagePlan(p.controller.config, p.images.identity, proof.Image, hash("inspection"))
	if err != nil || image.proof.Image != proof.Image || proof.Image.Project != ref.Project || image.key() != proof.ImageKey {
		return d, row, hostedexecution.ErrOwnership
	}
	return d, row, nil
}

func (p *Provider) reserveIntent(ctx context.Context, job hostedexecution.Job, image hostedexecution.Image) (launchIntent, error) {
	proof := launchProof{Reference: job.Reference, Image: image, Operation: job.Operation, Deadline: job.Ticket.Deadline.UTC(), TicketHash: hash(job.Ticket.Token)}
	imagePlan, err := NewImagePlan(p.controller.config, p.images.identity, image, hash("inspection"))
	if err != nil {
		return launchIntent{}, err
	}
	proof.ImageKey = imagePlan.key()
	work, _, err := p.controller.transport(ctx, dispatch{Proof: proof.executionProof()})
	if err != nil {
		return launchIntent{}, err
	}
	proof.InputHash = work.InputHash
	guards, err := dispatchGuards(ctx, p.controller.exchange, proof.executionProof(), job.Ticket.Token, proof.InputHash, p.controller.now())
	if err != nil {
		return launchIntent{}, err
	}
	proof.Fingerprint = p.fingerprint(proof)
	d := launchIntent{Proof: proof, CreatedAt: p.controller.now().UTC()}
	change, err := storage.Put(IntentBucket, job.Reference.ID, job.Reference.Project, 0, d)
	if err != nil {
		return d, hostedexecution.ErrUnavailable
	}
	err = p.controller.store.Commit(ctx, append(guards, change)...)
	if err != nil && !errors.Is(err, storage.ErrConflict) {
		return d, hostedexecution.ErrUnavailable
	}
	actual, _, err := p.loadIntent(ctx, job.Reference)
	if err == nil && actual.Proof != proof {
		return actual, hostedexecution.ErrOwnership
	}
	return actual, err
}

func (p *Provider) updateIntent(ctx context.Context, ref execution.Reference, fn func(*launchIntent) error) (launchIntent, error) {
	for range 8 {
		d, row, err := p.loadIntent(ctx, ref)
		if err != nil {
			return d, err
		}
		if err := fn(&d); err != nil {
			return d, err
		}
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, d)
		if err != nil {
			return d, hostedexecution.ErrUnavailable
		}
		err = p.controller.store.Commit(ctx, change)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		return d, controlError(err)
	}
	return launchIntent{}, hostedexecution.ErrUnavailable
}

func (p *Provider) validJob(job hostedexecution.Job) bool {
	return validReference(job.Reference, job.Runner, job.Operation) && job.ExchangeOrigin == p.controller.config.ExchangeOrigin && p.controller.config.validImage(job.Image) && job.Ticket.Execution == job.Reference.ID && ticketToken.MatchString(job.Ticket.Token) && job.Ticket.Deadline.After(p.controller.now()) && job.Ticket.Deadline.Sub(p.controller.now()) <= 2*time.Hour && (job.Operation != "discover" || p.images.writer)
}

func sameLaunch(proof launchProof, job hostedexecution.Job) bool {
	return proof.Reference == job.Reference && proof.Operation == job.Operation && proof.Image.Reference == job.Image && proof.Image.Runner == job.Runner && proof.Deadline.Equal(job.Ticket.Deadline) && proof.TicketHash == hash(job.Ticket.Token)
}
