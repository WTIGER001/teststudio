package azureworker

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKObserverHostReader struct{ service *sdkServiceReader }

func NewSDKObserverHostReader(c Config, s ObserverHostSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKObserverHostReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	service, err := newSDKServiceReader(c, c.ControllerEnvironment, s.Application, nil, credential, transport)
	if err != nil {
		return nil, err
	}
	service.http.resources = map[string]string{s.Application: exchangeVersion, s.Application + "/revisions": exchangeVersion}
	service.http.pages = map[string]bool{s.Application + "/revisions": true}
	return &SDKObserverHostReader{service: service}, nil
}

func (c *SDKObserverHostReader) Close() { c.service.Close() }

func (c *SDKObserverHostReader) Application(ctx context.Context, id string) (a.ContainerApp, error) {
	return c.service.Application(ctx, id)
}

func (c *SDKObserverHostReader) Revisions(ctx context.Context, id string) ([]a.Revision, error) {
	return c.service.Revisions(ctx, id)
}
