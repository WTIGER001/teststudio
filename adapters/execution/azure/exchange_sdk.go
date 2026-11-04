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
	settings     ExchangeSettings
	environments *a.ManagedEnvironmentsClient
	applications *a.ContainerAppsClient
	certificates *a.CertificatesClient
	auth         *a.ContainerAppsAuthConfigsClient
	connections  *a.ManagedEnvironmentPrivateEndpointConnectionsClient
	links        *a.ManagedEnvironmentPrivateLinkResourcesClient
	routes       *a.HTTPRouteConfigClient
	revisions    *a.ContainerAppsRevisionsClient
	http         *armReadHTTP
}

func NewSDKExchangeReader(c Config, s ExchangeSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKExchangeReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	allowed := map[string]string{}
	for _, id := range []string{s.environment(), s.Application, s.Certificate, s.auth(), s.connections(), s.links(), s.routes(), s.revisions()} {
		allowed[id] = exchangeVersion
	}
	h := newARMReadHTTP(c, allowed, transport)
	h.pages = map[string]bool{s.connections(): true, s.links(): true, s.routes(): true, s.revisions(): true}
	factory, err := a.NewClientFactory(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKExchangeReader{settings: s, environments: factory.NewManagedEnvironmentsClient(), applications: factory.NewContainerAppsClient(), certificates: factory.NewCertificatesClient(), auth: factory.NewContainerAppsAuthConfigsClient(), connections: factory.NewManagedEnvironmentPrivateEndpointConnectionsClient(), links: factory.NewManagedEnvironmentPrivateLinkResourcesClient(), routes: factory.NewHTTPRouteConfigClient(), revisions: factory.NewContainerAppsRevisionsClient(), http: h}, nil
}

func (c *SDKExchangeReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKExchangeReader) Environment(ctx context.Context, id string) (a.ManagedEnvironment, error) {
	if id != c.settings.environment() {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.environments.Get(ctx, group, name, nil)
	return response.ManagedEnvironment, sdkError(err, false)
}
func (c *SDKExchangeReader) Application(ctx context.Context, id string) (a.ContainerApp, error) {
	if id != c.settings.Application {
		return a.ContainerApp{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.applications.Get(ctx, group, name, nil)
	return response.ContainerApp, sdkError(err, false)
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
func (c *SDKExchangeReader) Auth(ctx context.Context, id string) (a.AuthConfig, error) {
	if id != c.settings.Application {
		return a.AuthConfig{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.auth.Get(ctx, group, name, "current", nil)
	return response.AuthConfig, sdkError(err, false)
}
