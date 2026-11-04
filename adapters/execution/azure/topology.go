package azureworker

import (
	"context"
	"strings"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type TopologyReader interface {
	Environment(context.Context, string) (a.ManagedEnvironment, error)
	VirtualNetwork(context.Context, string) (n.VirtualNetwork, error)
	Subnet(context.Context, string) (n.Subnet, error)
}

type TopologyGuard struct {
	worker   Config
	settings TopologySettings
	reader   TopologyReader
}

func NewTopologyGuard(c Config, s TopologySettings, reader TopologyReader) (*TopologyGuard, error) {
	if s.validate(c) != nil || reader == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &TopologyGuard{worker: c, settings: s, reader: reader}, nil
}

func (g *TopologyGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	worker, err := g.reader.Environment(ctx, c.WorkerEnvironment)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	controller, err := g.reader.Environment(ctx, c.ControllerEnvironment)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := g.environments(worker, controller); err != nil {
		return err
	}
	wnet, err := g.reader.VirtualNetwork(ctx, g.settings.WorkerVNet)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	cnet, err := g.reader.VirtualNetwork(ctx, g.settings.ControllerVNet)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	if err := g.networks(wnet, cnet); err != nil {
		return err
	}
	wsub, err := g.reader.Subnet(ctx, g.settings.WorkerSubnet)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	csub, err := g.reader.Subnet(ctx, g.settings.ControllerSubnet)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	return g.subnets(wnet, cnet, wsub, csub)
}

func environmentMatches(e a.ManagedEnvironment, id, subnet, location string) error {
	_, name := resourceParts(id)
	if !strings.EqualFold(value(e.ID), id) || value(e.Name) != name || !strings.EqualFold(value(e.Type), "Microsoft.App/managedEnvironments") || value(e.Location) != location || e.Properties == nil || e.Properties.VnetConfiguration == nil || !strings.EqualFold(value(e.Properties.VnetConfiguration.InfrastructureSubnetID), subnet) {
		return hostedexecution.ErrOwnership
	}
	if value(e.Properties.ProvisioningState) != a.EnvironmentProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (g *TopologyGuard) environments(worker, controller a.ManagedEnvironment) error {
	c, s := g.worker, g.settings
	if err := environmentMatches(worker, c.WorkerEnvironment, s.WorkerSubnet, c.Region); err != nil {
		return err
	}
	if err := environmentMatches(controller, c.ControllerEnvironment, s.ControllerSubnet, c.Region); err != nil {
		return err
	}
	p := worker.Properties
	v := p.VnetConfiguration
	if !value(v.Internal) || value(p.PublicNetworkAccess) != a.PublicNetworkAccessDisabled || len(p.PrivateEndpointConnections) != 0 || !strings.EqualFold(value(p.InfrastructureResourceGroup), s.InfrastructureGroup) || strings.EqualFold(value(controller.Properties.InfrastructureResourceGroup), s.InfrastructureGroup) || len(p.WorkloadProfiles) != 1 || p.WorkloadProfiles[0] == nil || value(p.WorkloadProfiles[0].Name) != "Consumption" || value(p.WorkloadProfiles[0].WorkloadProfileType) != "Consumption" || p.AppLogsConfiguration == nil || value(p.AppLogsConfiguration.Destination) != "none" || p.AppLogsConfiguration.LogAnalyticsConfiguration != nil || p.DaprAIConnectionString != nil || p.DaprAIInstrumentationKey != nil || value(v.DockerBridgeCidr) != "" || value(v.PlatformReservedCidr) != "" || value(v.PlatformReservedDNSIP) != "" {
		return hostedexecution.ErrOwnership
	}
	if worker.Identity != nil && (value(worker.Identity.Type) != a.ManagedServiceIdentityTypeNone || len(worker.Identity.UserAssignedIdentities) != 0) {
		return hostedexecution.ErrOwnership
	}
	if value(p.WorkloadProfiles[0].MinimumCount) != 0 || value(p.WorkloadProfiles[0].MaximumCount) != 0 {
		return hostedexecution.ErrOwnership
	}
	return g.workerAddress(value(p.StaticIP))
}
