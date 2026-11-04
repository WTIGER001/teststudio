package azureworker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type BackendReader interface {
	Environment(context.Context, string) (a.ManagedEnvironment, error)
	Application(context.Context, string) (a.ContainerApp, error)
	Auth(context.Context, string) (a.AuthConfig, error)
	Routes(context.Context, string) ([]a.HTTPRouteConfig, error)
	Revisions(context.Context, string) ([]a.Revision, error)
}

type BackendGuard struct {
	worker    Config
	settings  BackendSettings
	inventory *WorkloadInventoryGuard
	reader    BackendReader
}

func NewBackendGuard(c Config, s BackendSettings, inventory *WorkloadInventoryGuard, reader BackendReader) (*BackendGuard, error) {
	s, err := s.canonical(c)
	if err != nil || inventory == nil || reader == nil || fingerprint(c) != fingerprint(inventory.worker) || fingerprint(s.Exchange) != fingerprint(inventory.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &BackendGuard{worker: c, settings: s, inventory: inventory, reader: reader}, nil
}

func (g *BackendGuard) Check(ctx context.Context, c Config) error {
	_, err := g.inspect(ctx, c)
	return err
}

func (g *BackendGuard) inspect(ctx context.Context, c Config) (_ a.ManagedEnvironment, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	snapshot, err := g.inventory.inspect(ctx, c)
	if err != nil {
		return a.ManagedEnvironment{}, err
	}
	if err := verifyWorkloadPlacements(c, snapshot.applications, "containerApps", g.settings.Environment, g.settings.Application); err != nil {
		return a.ManagedEnvironment{}, err
	}
	if err := verifyWorkloadPlacements(c, snapshot.jobs, "jobs", g.settings.Environment, g.settings.Application); err != nil {
		return a.ManagedEnvironment{}, err
	}
	var environment a.ManagedEnvironment
	var application a.ContainerApp
	var auth a.AuthConfig
	var routes []a.HTTPRouteConfig
	var revisions []a.Revision
	var errs [5]error
	var group sync.WaitGroup
	s := g.settings
	group.Go(func() { environment, errs[0] = g.reader.Environment(ctx, s.Environment) })
	group.Go(func() { application, errs[1] = g.reader.Application(ctx, s.Application) })
	group.Go(func() { auth, errs[2] = g.reader.Auth(ctx, s.Application) })
	group.Go(func() { routes, errs[3] = g.reader.Routes(ctx, s.Environment) })
	group.Go(func() { revisions, errs[4] = g.reader.Revisions(ctx, s.Application) })
	group.Wait()
	for _, err := range errs {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
		}
	}
	for _, err := range errs {
		if err != nil {
			return a.ManagedEnvironment{}, hostedexecution.ErrUnavailable
		}
	}
	if err := verifyPrivateServiceEnvironment(environment, s.Environment, s.Subnet, s.InfrastructureGroup, c.Region); err != nil {
		return a.ManagedEnvironment{}, err
	}
	_, name := resourceParts(s.Application)
	if s.Exchange.BackendOrigin != "https://"+name+"."+value(environment.Properties.DefaultDomain) {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	if len(routes) != 0 || !strings.EqualFold(value(auth.ID), s.Application+"/authConfigs/current") || value(auth.Name) != "current" || !strings.EqualFold(value(auth.Type), "Microsoft.App/containerApps/authConfigs") {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	if auth.Properties == nil || auth.Properties.Platform == nil || auth.Properties.Platform.Enabled == nil {
		return a.ManagedEnvironment{}, hostedexecution.ErrUnavailable
	}
	if value(auth.Properties.Platform.Enabled) {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	if err := g.verifyApplication(application, value(environment.Properties.DefaultDomain)); err != nil {
		return a.ManagedEnvironment{}, err
	}
	expected, _ := s.ApplicationDefinition(c)
	if err := verifyServiceRevisions(application, revisions, s.Application, expected.Properties.Template, backendTemplateDocument); err != nil {
		return a.ManagedEnvironment{}, err
	}
	return environment, nil
}
