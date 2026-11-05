package ecsworker

import (
	"bytes"
	"context"
	"errors"
	"net/netip"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/ecsidentity"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

const GatewayBucket = "ecs-gateway-dispatches"

type GatewayAuthority interface {
	PrepareGateway(context.Context, execution.Reference, workerexchange.GatewaySettings) (workerexchange.GatewayTicket, error)
	BindGatewayAddress(context.Context, execution.Reference, string, string) error
}

type GatewayNetwork interface {
	AdmitGateway(context.Context, Config, GatewayConfig) error
	VerifyGatewayTask(context.Context, Config, GatewayConfig, types.Task) (string, error)
}

type gatewayImageInspector interface {
	InspectGateway(context.Context, string) (hostedexecution.GatewayImage, error)
}

type GatewayOptions struct {
	Worker    Options
	Gateway   GatewayConfig
	Authority GatewayAuthority
	Network   GatewayNetwork
}

type GatewayProvider struct{ control *Provider }

type gatewayControl struct {
	config    GatewayConfig
	authority GatewayAuthority
	network   GatewayNetwork
	inspector gatewayImageInspector
}

func NewGateway(options GatewayOptions) (*GatewayProvider, error) {
	inspector, ok := options.Worker.Inspector.(gatewayImageInspector)
	if !ok || options.Authority == nil || options.Network == nil || validateGateway(options.Worker.Config, options.Gateway) != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	provider, err := newProvider(options.Worker)
	if err != nil {
		return nil, err
	}
	provider.gateway = &gatewayControl{config: canonicalGateway(options.Gateway), authority: options.Authority, network: options.Network, inspector: inspector}
	return &GatewayProvider{control: provider}, nil
}

func (p *GatewayProvider) Ensure(ctx context.Context, job hostedexecution.Job) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	control := p.control
	ticket, err := control.gateway.authority.PrepareGateway(ctx, job.Reference, control.gatewaySettings())
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	plan, err := NewGatewayPlan(control.config, control.gateway.config, job, ticket, control.now())
	if err != nil {
		return err
	}
	return control.ensurePlan(ctx, plan)
}

func (p *Provider) gatewaySettings() workerexchange.GatewaySettings {
	return workerexchange.GatewaySettings{Origin: p.gateway.config.Origin, RequireAddress: true, CertificateAuthority: bytes.Clone(p.gateway.config.CertificateAuthority)}
}

func (p *GatewayProvider) Observe(ctx context.Context, ref execution.Reference) (hostedexecution.Status, error) {
	return p.control.Observe(ctx, ref)
}

func (p *GatewayProvider) Stop(ctx context.Context, ref execution.Reference) error {
	return p.control.Stop(ctx, ref)
}

func (p *GatewayProvider) Remove(ctx context.Context, ref execution.Reference) error {
	return p.control.Remove(ctx, ref)
}

func (p *GatewayProvider) Inventory(ctx context.Context, installation, profile string) ([]execution.Reference, error) {
	return p.control.Inventory(ctx, installation, profile)
}

func (p *Provider) bucket() string {
	if p.gateway != nil {
		return GatewayBucket
	}
	return Bucket
}

func (p *Provider) ownsRole(proof Proof) bool {
	if p.gateway != nil {
		return proof.Environment == nil && proof.Gateway != nil && fingerprint(canonicalGateway(*proof.Gateway)) == fingerprint(p.gateway.config)
	}
	if p.environment != nil {
		return proof.Gateway == nil && proof.Environment != nil && fingerprint(canonicalGateway(*proof.Environment)) == fingerprint(*p.environment)
	}
	return proof.Gateway == nil && proof.Environment == nil
}

func sameGatewayAuthority(settings workerexchange.GatewaySettings, config GatewayConfig) bool {
	return bytes.Equal(settings.CertificateAuthority, config.CertificateAuthority)
}

func (p *Provider) admitTask(ctx context.Context) error {
	if p.gateway == nil {
		return p.network.Admit(ctx, canonical(p.config))
	}
	gateway := p.gateway
	_, err := p.inspectGateway(ctx)
	if err != nil {
		return err
	}
	return gateway.network.AdmitGateway(ctx, canonical(p.config), canonicalGateway(gateway.config))
}

func (p *Provider) inspectGateway(ctx context.Context) (hostedexecution.GatewayImage, error) {
	gateway := p.gateway
	image, err := gateway.inspector.InspectGateway(ctx, gateway.config.Image)
	if err != nil {
		return image, hostedexecution.ErrUnavailable
	}
	if image.Reference != gateway.config.Image || image.OS != "linux" || image.Architecture != p.config.Architecture || image.Protocol != browsergateway.RemoteProtocol || image.CertificateConfiguration != browsergateway.RemoteCertificateProtocol || image.InstanceProtocol != ecsidentity.Protocol {
		return image, hostedexecution.ErrConfiguration
	}
	return image, nil
}

func (p *Provider) verifyTaskNetwork(ctx context.Context, plan *Plan, task types.Task) error {
	if p.gateway == nil {
		return p.network.VerifyTask(ctx, canonical(p.config), task)
	}
	address, err := p.gateway.network.VerifyGatewayTask(ctx, canonical(p.config), canonicalGateway(p.gateway.config), task)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(address)
	if err != nil || !ip.Is4() || !ip.IsPrivate() || ip.String() != address || len(task.Containers) != 1 || len(task.Containers[0].NetworkInterfaces) != 1 || aws.ToString(task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address) != address {
		return hostedexecution.ErrOwnership
	}
	instance, err := plan.GatewayInstance(aws.ToString(task.TaskArn))
	if err != nil {
		return err
	}
	value, _, _, err := p.load(ctx, plan.job.Reference)
	if err != nil {
		return err
	}
	record, _, err := p.exchange(ctx, value)
	if err != nil {
		return err
	}
	if record.Gateway.Instance == instance && record.Gateway.Settings.Address == address {
		return nil
	}
	if err := p.gateway.authority.BindGatewayAddress(ctx, plan.job.Reference, instance, address); err != nil {
		if errors.Is(err, workerexchange.ErrConflict) || errors.Is(err, workerexchange.ErrInvalid) {
			return hostedexecution.ErrOwnership
		}
		return hostedexecution.ErrUnavailable
	}
	return nil
}
