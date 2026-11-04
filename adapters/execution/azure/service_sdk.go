package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type sdkServiceReader struct {
	environment, application string
	factory                  *a.ClientFactory
	environments             *a.ManagedEnvironmentsClient
	applications             *a.ContainerAppsClient
	auth                     *a.ContainerAppsAuthConfigsClient
	routes                   *a.HTTPRouteConfigClient
	revisions                *a.ContainerAppsRevisionsClient
	http                     *armReadHTTP
}

func newSDKServiceReader(c Config, environment, application string, extra map[string]string, credential azcore.TokenCredential, transport policy.Transporter) (*sdkServiceReader, error) {
	if credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	allowed := map[string]string{}
	for _, id := range []string{environment, application, application + "/authConfigs/current", environment + "/httpRouteConfigs", application + "/revisions"} {
		allowed[id] = exchangeVersion
	}
	for id, version := range extra {
		allowed[id] = version
	}
	h := newARMReadHTTP(c, allowed, transport)
	h.pages = map[string]bool{environment + "/httpRouteConfigs": true, application + "/revisions": true}
	factory, err := a.NewClientFactory(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &sdkServiceReader{environment: environment, application: application, factory: factory, environments: factory.NewManagedEnvironmentsClient(), applications: factory.NewContainerAppsClient(), auth: factory.NewContainerAppsAuthConfigsClient(), routes: factory.NewHTTPRouteConfigClient(), revisions: factory.NewContainerAppsRevisionsClient(), http: h}, nil
}

func (c *sdkServiceReader) Close() { c.http.CloseIdleConnections() }

func (c *sdkServiceReader) Environment(ctx context.Context, id string) (a.ManagedEnvironment, error) {
	if id != c.environment {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.environments.Get(ctx, group, name, nil)
	return response.ManagedEnvironment, sdkError(err, false)
}
func (c *sdkServiceReader) Application(ctx context.Context, id string) (a.ContainerApp, error) {
	if id != c.application {
		return a.ContainerApp{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.applications.Get(ctx, group, name, nil)
	return response.ContainerApp, sdkError(err, false)
}
func (c *sdkServiceReader) Auth(ctx context.Context, id string) (a.AuthConfig, error) {
	if id != c.application {
		return a.AuthConfig{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.auth.Get(ctx, group, name, "current", nil)
	return response.AuthConfig, sdkError(err, false)
}

func (c *sdkServiceReader) Routes(ctx context.Context, id string) ([]a.HTTPRouteConfig, error) {
	if id != c.environment {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.routes.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, c.environment+"/httpRouteConfigs", exchangeVersion, 0, 1, pager.More, func(ctx context.Context) ([]*a.HTTPRouteConfig, *string, error) {
		page, err := pager.NextPage(ctx)
		if err == nil && page.Value == nil {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		return page.Value, page.NextLink, err
	})
}
func (c *sdkServiceReader) Revisions(ctx context.Context, id string) ([]a.Revision, error) {
	if id != c.application {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.revisions.NewListRevisionsPager(group, name, nil)
	return armInventory(ctx, c.http, c.application+"/revisions", exchangeVersion, 4, 3, pager.More, func(ctx context.Context) ([]*a.Revision, *string, error) {
		page, err := pager.NextPage(ctx)
		if err == nil && page.Value == nil {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		return page.Value, page.NextLink, err
	})
}
