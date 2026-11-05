package azureworker

import (
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RegistryServiceBinding struct {
	Registry, Login, Region string
	Endpoints               []string
}

type RegistryServiceObservation struct {
	Registry    a.Registry
	Connections []a.PrivateEndpointConnection
	Links       []a.PrivateLinkResource
	Replicas    []a.Replication
}

func (g *RegistryGuard) binding() RegistryServiceBinding {
	return RegistryServiceBinding{Registry: g.settings.registry(), Login: g.worker.Registry, Region: g.settings.region(), Endpoints: g.settings.consumers()}
}

func (s RegistryServiceBinding) Verify(actual RegistryServiceObservation) error {
	if err := s.verifyRegistry(actual.Registry); err != nil {
		return err
	}
	observed, err := VerifyRegistryEndpointConnections(s.Registry, s.Endpoints, actual.Connections)
	if err != nil {
		return err
	}
	embedded := make([]a.PrivateEndpointConnection, 0, len(actual.Registry.Properties.PrivateEndpointConnections))
	for _, connection := range actual.Registry.Properties.PrivateEndpointConnections {
		if connection == nil {
			return hostedexecution.ErrOwnership
		}
		embedded = append(embedded, *connection)
	}
	declared, err := VerifyRegistryEndpointConnections(s.Registry, s.Endpoints, embedded)
	if err != nil {
		return err
	}
	if fingerprint(observed) != fingerprint(declared) {
		return hostedexecution.ErrUnavailable
	}
	if err := s.verifyLinks(actual.Links); err != nil {
		return err
	}
	return s.verifyReplicas(actual.Replicas)
}
