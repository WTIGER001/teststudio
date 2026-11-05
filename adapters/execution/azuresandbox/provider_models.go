package azuresandbox

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const IntentBucket = "azure-sandbox-launches"

type Inspector interface {
	InspectImage(context.Context, string) (hostedexecution.Image, error)
}

type ProviderOptions struct {
	Mode       string
	Controller *Controller
	Images     *ImageManager
	Inspector  Inspector
}

type Provider struct {
	closed     atomic.Bool
	mode       string
	controller *Controller
	images     *ImageManager
	inspector  Inspector
}

type launchProof struct {
	Reference   execution.Reference   `json:"reference"`
	Image       hostedexecution.Image `json:"image"`
	Operation   string                `json:"operation,omitempty"`
	Deadline    time.Time             `json:"deadline"`
	TicketHash  string                `json:"ticketHash"`
	InputHash   string                `json:"inputHash"`
	ImageKey    string                `json:"imageKey"`
	Fingerprint string                `json:"fingerprint"`
}

type launchIntent struct {
	Proof     launchProof `json:"proof"`
	CreatedAt time.Time   `json:"createdAt"`
	Stopping  bool        `json:"stopping"`
	Rejected  bool        `json:"rejected"`
	Removed   bool        `json:"removed"`
}

func NewProvider(o ProviderOptions) (*Provider, error) {
	if o.Mode != "" && o.Mode != "execution" && o.Mode != "discovery" || o.Controller == nil || o.Controller.commands == nil || o.Images == nil || o.Inspector == nil || o.Controller.exchange != o.Images.exchange || hash(o.Controller.config) != hash(o.Images.config) || o.Mode == "discovery" && !o.Images.writer {
		return nil, hostedexecution.ErrConfiguration
	}
	return &Provider{controller: o.Controller, images: o.Images, inspector: o.Inspector, mode: o.Mode}, nil
}

func (p *Provider) permits(operation string) bool {
	return p.mode == "" || p.mode == "execution" && operation == "" || p.mode == "discovery" && operation == "discover"
}

func (p *Provider) Close() { p.closed.Store(true); p.controller.Close() }

func (p *Provider) InspectImage(ctx context.Context, reference string) (hostedexecution.Image, error) {
	if p.closed.Load() {
		return hostedexecution.Image{}, hostedexecution.ErrUnavailable
	}
	if !p.controller.config.validImage(reference) {
		return hostedexecution.Image{}, hostedexecution.ErrConfiguration
	}
	var image hostedexecution.Image
	err := bounded(ctx, func(ctx context.Context) error {
		var e error
		image, e = p.inspector.InspectImage(ctx, reference)
		return e
	})
	if err != nil {
		return hostedexecution.Image{}, err
	}
	if image.Reference != reference {
		return hostedexecution.Image{}, hostedexecution.ErrOwnership
	}
	plan, err := NewImagePlan(p.controller.config, p.images.identity, image, hash("inspection"))
	if err != nil {
		return hostedexecution.Image{}, err
	}
	return plan.proof.Image, nil
}

func (p *Provider) fingerprint(proof launchProof) string {
	proof.Fingerprint = ""
	return hash([]any{"azure-sandbox-launch/v1", p.controller.config, proof})
}

func (proof launchProof) executionProof() Proof {
	return Proof{Reference: proof.Reference, Image: proof.Image.Reference, SourceDigest: proof.Image.SourceDigest, Sandbox: proof.Image.Sandbox, Runner: proof.Image.Runner, Operation: proof.Operation, Deadline: proof.Deadline, TicketHash: proof.TicketHash, EnvironmentHash: ""}
}

func (proof launchProof) matches(actual Proof) bool {
	return actual.Reference == proof.Reference && actual.Image == proof.Image.Reference && actual.SourceDigest == proof.Image.SourceDigest && actual.Sandbox == proof.Image.Sandbox && actual.Runner == proof.Image.Runner && actual.Operation == proof.Operation && actual.Deadline.Equal(proof.Deadline) && actual.TicketHash == proof.TicketHash
}

func (d launchIntent) status() hostedexecution.Status {
	phase := "pending"
	if d.Rejected {
		phase = "failed"
	}
	return hostedexecution.Status{Reference: d.Proof.Reference, Image: d.Proof.Image.Reference, Deadline: d.Proof.Deadline, Phase: phase, AdmissionPending: !d.Rejected}
}

var _ hostedexecution.Provider = (*Provider)(nil)
