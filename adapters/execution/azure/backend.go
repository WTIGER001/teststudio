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

func (g *BackendGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	snapshot, err := g.inventory.inspect(ctx, c)
	if err != nil {
		return err
	}
	if err := verifyWorkloadPlacements(c, snapshot.applications, "containerApps", g.settings.Environment, g.settings.Application); err != nil {
		return err
	}
	if err := verifyWorkloadPlacements(c, snapshot.jobs, "jobs", g.settings.Environment, g.settings.Application); err != nil {
		return err
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
			return hostedexecution.ErrOwnership
		}
	}
	for _, err := range errs {
		if err != nil {
			return hostedexecution.ErrUnavailable
		}
	}
	if err := verifyPrivateServiceEnvironment(environment, s.Environment, s.Subnet, s.InfrastructureGroup, c.Region); err != nil {
		return err
	}
	_, name := resourceParts(s.Application)
	if s.Exchange.BackendOrigin != "https://"+name+"."+value(environment.Properties.DefaultDomain) {
		return hostedexecution.ErrOwnership
	}
	if len(routes) != 0 || !strings.EqualFold(value(auth.ID), s.Application+"/authConfigs/current") || value(auth.Name) != "current" || !strings.EqualFold(value(auth.Type), "Microsoft.App/containerApps/authConfigs") {
		return hostedexecution.ErrOwnership
	}
	if auth.Properties == nil || auth.Properties.Platform == nil || auth.Properties.Platform.Enabled == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(auth.Properties.Platform.Enabled) {
		return hostedexecution.ErrOwnership
	}
	if err := g.verifyApplication(application, value(environment.Properties.DefaultDomain)); err != nil {
		return err
	}
	expected, _ := s.ApplicationDefinition(c)
	return verifyServiceRevisions(application, revisions, s.Application, expected.Properties.Template, backendTemplateDocument)
}
