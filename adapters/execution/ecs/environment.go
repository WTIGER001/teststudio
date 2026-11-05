package ecsworker

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type EnvironmentAuthority interface {
	GatewayAuthority
	ResumeDispatch(context.Context, execution.Reference) (workerexchange.Ticket, error)
}

type EnvironmentOptions struct {
	Worker    Options
	Gateway   GatewayConfig
	Authority EnvironmentAuthority
	Network   GatewayNetwork
}

type EnvironmentProvider struct {
	worker    *Provider
	gateway   *GatewayProvider
	authority EnvironmentAuthority
}

type environmentDispatch struct {
	GatewayAttempted bool `json:"gatewayAttempted,omitempty"`
	Failed           bool `json:"failed,omitempty"`
}

func NewEnvironment(options EnvironmentOptions) (*EnvironmentProvider, error) {
	worker, err := New(options.Worker)
	if err != nil {
		return nil, err
	}
	gateway, err := NewGateway(GatewayOptions{Worker: options.Worker, Gateway: options.Gateway, Authority: options.Authority, Network: options.Network})
	if err != nil {
		return nil, err
	}
	selected := canonicalGateway(options.Gateway)
	worker.environment = &selected
	return &EnvironmentProvider{worker: worker, gateway: gateway, authority: options.Authority}, nil
}

func newEnvironmentPlan(config Config, gateway GatewayConfig, job hostedexecution.Job, now time.Time) (*Plan, error) {
	if !ticket.MatchString(job.Ticket.Token) || !job.Ticket.Deadline.After(now) || job.Ticket.Deadline.After(now.Add(2*time.Hour)) {
		return nil, hostedexecution.ErrConfiguration
	}
	return newRolePlan(config, job, fingerprint(job.Ticket.Token), nil, &gateway)
}

func (p *EnvironmentProvider) SupportsGateway() bool { return true }

func (p *EnvironmentProvider) GatewayProfile(ctx context.Context) (string, error) {
	image, err := p.gateway.control.inspectGateway(ctx)
	if err != nil {
		return "", err
	}
	return "ecs-gateway-v1-" + fingerprint([]any{image, *p.worker.environment, "cpu512-memory1024"})[:20], nil
}

func (p *EnvironmentProvider) InspectImage(ctx context.Context, reference string) (hostedexecution.Image, error) {
	image, err := p.worker.InspectImage(ctx, reference)
	if err != nil {
		return image, err
	}
	if image.GatewayTransport != browsergateway.RemoteProtocol || image.GatewayAddress != browsergateway.RemoteAddressProtocol || !checksum.MatchString(image.SupervisorRevision) {
		return image, hostedexecution.ErrConfiguration
	}
	image.ExecutionProfile = "fargate-environment-v1-" + fingerprint([]any{canonical(p.worker.config), *p.worker.environment, "cpu2048-memory4096-taskdisk21g-volumes-v1"})[:20]
	return image, nil
}

func (p *EnvironmentProvider) Ensure(ctx context.Context, job hostedexecution.Job) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	plan, err := newEnvironmentPlan(p.worker.config, *p.worker.environment, job, p.worker.now())
	if err != nil {
		return err
	}
	if _, err := p.InspectImage(ctx, job.Image); err != nil {
		return hostedexecution.ErrRejected
	}
	value, _, err := p.worker.reserve(ctx, plan)
	if err != nil {
		return err
	}
	if value.Removed || value.Stopping || value.Environment.Failed || value.RegisterRejected || value.LaunchRejected {
		return hostedexecution.ErrRejected
	}
	claimed := false
	_, err = p.worker.update(ctx, job.Reference, true, func(value *dispatch) error {
		claimed = false
		if !value.Environment.GatewayAttempted {
			value.Environment.GatewayAttempted, claimed = true, true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !claimed {
		if _, _, _, err := p.gateway.control.load(ctx, job.Reference); errors.Is(err, storage.ErrNotFound) {
			return hostedexecution.ErrUnavailable
		} else if err != nil {
			return err
		}
	}
	return p.gateway.Ensure(ctx, job)
}

var _ hostedexecution.Provider = (*EnvironmentProvider)(nil)
var _ hostedexecution.GatewayProvider = (*EnvironmentProvider)(nil)
