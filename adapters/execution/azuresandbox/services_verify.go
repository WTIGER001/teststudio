package azuresandbox

import (
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKServices) verify(values []placementResource, collections map[string][]placementResource) error {
	if len(values) != 7 {
		return hostedexecution.ErrUnavailable
	}
	s := c.settings
	p, v := s.Placement, s.Placement.Services
	relay, err := resourceModel[a.ManagedEnvironment](values[0])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateServiceEnvironment(relay, p.Private.ExchangeEnvironment, v.Relay.Subnet, v.Relay.InfrastructureGroup, s.Worker.Region); err != nil {
		return err
	}
	connections, err := resourceModels[a.PrivateEndpointConnection](collections[p.Private.ExchangeEnvironment+"/privateEndpointConnections"])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyServiceEndpointConnections(relay, connections, p.Private.ExchangeEndpoint); err != nil {
		return err
	}
	links, err := resourceModels[a.PrivateLinkResource](collections[p.Private.ExchangeEnvironment+"/privateLinkResources"])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyServicePrivateLinks(links, p.Private.ExchangeEnvironment, s.Worker.Region); err != nil {
		return err
	}
	certificate, err := resourceModel[a.Certificate](values[2])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyServiceCertificate(certificate, v.Relay.Certificate, s.Worker.Region, s.Worker.ExchangeOrigin, c.now()); err != nil {
		return err
	}
	backend, err := resourceModel[a.ManagedEnvironment](values[4])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateServiceEnvironment(backend, v.Backend.Environment, v.Backend.Subnet, v.Backend.InfrastructureGroup, s.Worker.Region); err != nil {
		return err
	}
	for _, service := range []struct {
		index         int
		backend       bool
		id, domain    string
		configuration int
	}{{1, false, v.Relay.Application, *relay.Properties.DefaultDomain, 3}, {5, true, v.Backend.Application, *backend.Properties.DefaultDomain, 6}} {
		app, err := resourceModel[a.ContainerApp](values[service.index])
		if err != nil {
			return err
		}
		auth, err := resourceModel[a.AuthConfig](values[service.configuration])
		if err != nil {
			return err
		}
		if err := azureworker.VerifyServiceAuth(auth, service.id); err != nil {
			return err
		}
		revisions, err := resourceModels[a.Revision](collections[service.id+"/revisions"])
		if err != nil {
			return err
		}
		if err := c.verifyApplication(app, revisions, service.domain, service.backend); err != nil {
			return err
		}
	}
	if len(collections[p.Private.ExchangeEnvironment+"/httpRouteConfigs"])+len(collections[v.Backend.Environment+"/httpRouteConfigs"]) != 0 {
		return hostedexecution.ErrOwnership
	}
	return nil
}
