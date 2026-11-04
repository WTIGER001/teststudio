package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const exchangeVersion = "2025-07-01"

type SDKExchangeReader struct {
	*sdkServiceReader
	settings     ExchangeSettings
	certificates *a.CertificatesClient
	connections  *a.ManagedEnvironmentPrivateEndpointConnectionsClient
	links        *a.ManagedEnvironmentPrivateLinkResourcesClient
}

func NewSDKExchangeReader(c Config, s ExchangeSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKExchangeReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	core, err := newSDKServiceReader(c, s.environment(), s.Application, map[string]string{s.Certificate: exchangeVersion, s.connections(): exchangeVersion, s.links(): exchangeVersion}, credential, transport)
	if err != nil {
		return nil, err
	}
	core.http.pages[s.connections()], core.http.pages[s.links()] = true, true
	return &SDKExchangeReader{sdkServiceReader: core, settings: s, certificates: core.factory.NewCertificatesClient(), connections: core.factory.NewManagedEnvironmentPrivateEndpointConnectionsClient(), links: core.factory.NewManagedEnvironmentPrivateLinkResourcesClient()}, nil
}

func (c *SDKExchangeReader) Certificate(ctx context.Context, id string) (a.Certificate, error) {
	if id != c.settings.Certificate {
		return a.Certificate{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(c.settings.environment())
	certificate := strings.TrimPrefix(id, c.settings.environment()+"/certificates/")
	response, err := c.certificates.Get(ctx, group, name, certificate, nil)
	return response.Certificate, sdkError(err, false)
}
