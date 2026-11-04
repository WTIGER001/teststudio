package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RegistryReader interface {
	Registry(context.Context, string) (a.Registry, error)
	Connections(context.Context, string) ([]a.PrivateEndpointConnection, error)
	Links(context.Context, string) ([]a.PrivateLinkResource, error)
	Replicas(context.Context, string) ([]a.Replication, error)
}

type RegistryGuard struct {
	worker    Config
	settings  RegistrySettings
	endpoints *EndpointGuard
	reader    RegistryReader
}

func NewRegistryGuard(c Config, s RegistrySettings, endpoints *EndpointGuard, reader RegistryReader) (*RegistryGuard, error) {
	s, err := s.canonical(c)
	if err != nil || endpoints == nil || reader == nil || fingerprint(c) != fingerprint(endpoints.worker) || fingerprint(s.Endpoints) != fingerprint(endpoints.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &RegistryGuard{worker: c, settings: s, endpoints: endpoints, reader: reader}, nil
}

func (g *RegistryGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := g.endpoints.Check(ctx, c); err != nil {
		return err
	}
	var registry a.Registry
	var connections []a.PrivateEndpointConnection
	var links []a.PrivateLinkResource
	var replicas []a.Replication
	var errs [4]error
	var group sync.WaitGroup
	id := g.settings.registry()
	group.Go(func() { registry, errs[0] = g.reader.Registry(ctx, id) })
	group.Go(func() { connections, errs[1] = g.reader.Connections(ctx, id) })
	group.Go(func() { links, errs[2] = g.reader.Links(ctx, id) })
	group.Go(func() { replicas, errs[3] = g.reader.Replicas(ctx, id) })
	group.Wait()
	for _, err := range errs {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return hostedexecution.ErrOwnership
		}
	}
	for _, err := range errs {
		if err != nil {
			return hostedexecution.ErrUnavailable
		}
	}
	if err := g.verifyRegistry(registry); err != nil {
		return err
	}
	observed, err := g.verifyConnections(connections)
	if err != nil {
		return err
	}
	embedded := make([]a.PrivateEndpointConnection, 0, len(registry.Properties.PrivateEndpointConnections))
	for _, connection := range registry.Properties.PrivateEndpointConnections {
		if connection == nil {
			return hostedexecution.ErrOwnership
		}
		embedded = append(embedded, *connection)
	}
	declared, err := g.verifyConnections(embedded)
	if err != nil {
		return err
	}
	if fingerprint(observed) != fingerprint(declared) {
		return hostedexecution.ErrUnavailable
	}
	if err := g.verifyLinks(links); err != nil {
		return err
	}
	return g.verifyReplicas(replicas)
}
