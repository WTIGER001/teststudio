package kubernetes

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type gatewayInspector interface {
	InspectGateway(context.Context, string) (hostedexecution.GatewayImage, error)
}

func (p *Provider) SupportsGateway() bool {
	_, ok := p.inspector.(gatewayInspector)
	return ok && p.config.Gateway != nil && p.gatewayAuthority != nil
}

func (p *Provider) GatewayProfile(ctx context.Context) (string, error) {
	if !p.SupportsGateway() {
		return "", hostedexecution.ErrConfiguration
	}
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	image, err := p.inspector.(gatewayInspector).InspectGateway(check, p.config.Gateway.Image)
	if err != nil {
		return "", hostedexecution.ErrUnavailable
	}
	if image.Reference != p.config.Gateway.Image || image.OS != "linux" || image.Architecture != p.config.Architecture || image.Protocol != browsergateway.RemoteProtocol {
		return "", hostedexecution.ErrConfiguration
	}
	return "gateway-v1-" + fingerprint([]any{image, p.config.Namespace, p.config.RuntimeClass, p.config.NodeSelector, p.config.Gateway.ServiceAccount, p.config.Gateway.TLSSecret, p.config.Gateway.ExchangeTrustSecret, p.gatewayCA, "cpu1-memory256Mi"})[:20], nil
}
