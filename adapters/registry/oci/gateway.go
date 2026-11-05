package oci

import (
	"context"
	"slices"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type imageDocument struct {
	repository, hash string
	manifest         manifest
	configuration    configuration
	data             []byte
}

func (c *Client) InspectGateway(ctx context.Context, reference string) (hostedexecution.GatewayImage, error) {
	document, err := c.readImage(ctx, reference)
	if err != nil {
		return hostedexecution.GatewayImage{}, err
	}
	config := document.configuration
	labels := config.Config.Labels
	defaultEnvironment := len(config.Config.Env) == 0 || slices.Equal(config.Config.Env, []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"})
	if !slices.Equal(config.Config.Entrypoint, []string{"/gateway"}) || len(config.Config.Cmd) != 0 || !defaultEnvironment || len(config.Config.Volumes) != 0 || config.Config.WorkingDir != "" && config.Config.WorkingDir != "/" || labels["teststudio.gateway-remote"] != browsergateway.RemoteProtocol || labels["teststudio.api-policy"] != "request-policy/v1" || labels["teststudio.api-credentials"] != "gateway-injection/v1" || labels["teststudio.gateway-config"] != "private-config/v1" {
		return hostedexecution.GatewayImage{}, ErrInvalid
	}
	for _, key := range []string{"teststudio.runtime", "teststudio.project", "teststudio.installation", "teststudio.preparation", "teststudio.source"} {
		if labels[key] != "" {
			return hostedexecution.GatewayImage{}, ErrInvalid
		}
	}
	return hostedexecution.GatewayImage{Reference: reference, OS: config.OS, Architecture: config.Architecture, Protocol: labels["teststudio.gateway-remote"], CertificateConfiguration: labels["teststudio.gateway-certificate"]}, nil
}
