package azureworker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type BackendEndpointReader interface {
	EndpointReader
	Connections(context.Context, string) ([]a.PrivateEndpointConnection, error)
	Links(context.Context, string) ([]a.PrivateLinkResource, error)
}

type BackendEndpointGuard struct {
	worker   Config
	settings BackendEndpointSettings
	backend  *BackendGuard
	reader   BackendEndpointReader
}

func NewBackendEndpointGuard(c Config, s BackendEndpointSettings, backend *BackendGuard, reader BackendEndpointReader) (*BackendEndpointGuard, error) {
	s, err := s.canonical(c)
	if err != nil || backend == nil || reader == nil || fingerprint(c) != fingerprint(backend.worker) || fingerprint(s.Backend) != fingerprint(backend.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &BackendEndpointGuard{worker: c, settings: s, backend: backend, reader: reader}, nil
}

func (g *BackendEndpointGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	environment, err := g.backend.inspect(ctx, c)
	if err != nil {
		return err
	}
	s := g.settings
	var endpoint n.PrivateEndpoint
	var nic n.Interface
	var subnet, relay n.Subnet
	var security n.SecurityGroup
	var connections []a.PrivateEndpointConnection
	var links []a.PrivateLinkResource
	var errs [7]error
	var group sync.WaitGroup
	group.Go(func() { endpoint, errs[0] = g.reader.Endpoint(ctx, s.Endpoint) })
	group.Go(func() { nic, errs[1] = g.reader.Interface(ctx, s.Interface) })
	group.Go(func() { subnet, errs[2] = g.reader.Subnet(ctx, s.Subnet) })
	group.Go(func() { security, errs[3] = g.reader.SecurityGroup(ctx, s.SecurityGroup) })
	group.Go(func() { relay, errs[4] = g.reader.Subnet(ctx, s.Backend.Exchange.Subnet) })
	group.Go(func() { connections, errs[5] = g.reader.Connections(ctx, s.Backend.Environment) })
	group.Go(func() { links, errs[6] = g.reader.Links(ctx, s.Backend.Environment) })
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
	observed, err := verifyEnvironmentEndpointConnections(connections, s.Backend.Environment, s.Endpoint)
	if err != nil {
		return err
	}
	embedded := []a.PrivateEndpointConnection{}
	for _, connection := range environment.Properties.PrivateEndpointConnections {
		if connection == nil {
			return hostedexecution.ErrOwnership
		}
		embedded = append(embedded, *connection)
	}
	declared, err := verifyEnvironmentEndpointConnections(embedded, s.Backend.Environment, s.Endpoint)
	if err != nil {
		return err
	}
	if observed != declared {
		return hostedexecution.ErrUnavailable
	}
	member, err := verifyBackendEndpointLink(links, s.Backend.Environment, c.Region)
	if err != nil {
		return err
	}
	role := s.role()
	if err := verifyPrivateEndpoint(role, endpoint, s.Subnet, c.Region); err != nil {
		return err
	}
	if err := verifyPrivateInterface(role, nic, s.Subnet, c.Region); err != nil {
		return err
	}
	if value(nic.Properties.IPConfigurations[0].Properties.PrivateLinkConnectionProperties.RequiredMemberName) != member {
		return hostedexecution.ErrOwnership
	}
	for _, configuration := range endpoint.Properties.IPConfigurations {
		if value(configuration.Properties.MemberName) != member {
			return hostedexecution.ErrOwnership
		}
	}
	if err := verifyEndpointSubnet(subnet, []n.Interface{nic}, s.Subnet, s.SubnetPrefix, s.SecurityGroup, []string{s.Endpoint}); err != nil {
		return err
	}
	if err := verifyEndpointSecurityGroup(security, s.SecurityGroup, s.Subnet, c.Region, s.rules()); err != nil {
		return err
	}
	if err := subnetMatches(relay, s.Backend.Exchange.Subnet); err != nil {
		return err
	}
	prefixes := subnetPrefixes(relay)
	if len(prefixes) != 1 || prefixes[0] != s.RelaySubnetPrefix || len(relay.Properties.Delegations) != 1 || relay.Properties.Delegations[0] == nil || relay.Properties.Delegations[0].Properties == nil {
		return hostedexecution.ErrOwnership
	}
	delegation := relay.Properties.Delegations[0].Properties
	if value(delegation.ServiceName) != "Microsoft.App/environments" {
		return hostedexecution.ErrOwnership
	}
	if value(delegation.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func verifyBackendEndpointLink(links []a.PrivateLinkResource, environment, region string) (string, error) {
	if len(links) == 0 {
		return "", hostedexecution.ErrUnavailable
	}
	if len(links) != 1 {
		return "", hostedexecution.ErrOwnership
	}
	link := links[0]
	if !endpointChild(value(link.ID), value(link.Name), environment, "privateLinkResources") || !strings.EqualFold(value(link.Type), "Microsoft.App/managedEnvironments/privateLinkResources") || link.Properties == nil {
		return "", hostedexecution.ErrOwnership
	}
	p := link.Properties
	if p.GroupID == nil || len(p.RequiredMembers) == 0 || len(p.RequiredZoneNames) == 0 {
		return "", hostedexecution.ErrUnavailable
	}
	if value(p.GroupID) != "managedEnvironment" || len(p.RequiredMembers) != 1 || !endpointChildName.MatchString(value(p.RequiredMembers[0])) || !registryStrings(p.RequiredZoneNames, "privatelink."+region+".azurecontainerapps.io") {
		return "", hostedexecution.ErrOwnership
	}
	return value(p.RequiredMembers[0]), nil
}
