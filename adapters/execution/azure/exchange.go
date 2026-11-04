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

type ExchangeReader interface {
	Environment(context.Context, string) (a.ManagedEnvironment, error)
	Application(context.Context, string) (a.ContainerApp, error)
	Certificate(context.Context, string) (a.Certificate, error)
	Auth(context.Context, string) (a.AuthConfig, error)
	Connections(context.Context, string) ([]a.PrivateEndpointConnection, error)
	Links(context.Context, string) ([]a.PrivateLinkResource, error)
	Routes(context.Context, string) ([]a.HTTPRouteConfig, error)
	Revisions(context.Context, string) ([]a.Revision, error)
}

type ExchangeGuard struct {
	worker   Config
	settings ExchangeSettings
	registry *RegistryGuard
	reader   ExchangeReader
}

func NewExchangeGuard(c Config, s ExchangeSettings, registry *RegistryGuard, reader ExchangeReader) (*ExchangeGuard, error) {
	s, err := s.canonical(c)
	if err != nil || registry == nil || reader == nil || fingerprint(c) != fingerprint(registry.worker) || fingerprint(s.Registry) != fingerprint(registry.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &ExchangeGuard{worker: c, settings: s, registry: registry, reader: reader}, nil
}

func (g *ExchangeGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	if err := g.registry.Check(ctx, c); err != nil {
		return err
	}
	var env a.ManagedEnvironment
	var app a.ContainerApp
	var cert a.Certificate
	var auth a.AuthConfig
	var connections []a.PrivateEndpointConnection
	var links []a.PrivateLinkResource
	var routes []a.HTTPRouteConfig
	var revisions []a.Revision
	var errs [8]error
	var group sync.WaitGroup
	s := g.settings
	group.Go(func() { env, errs[0] = g.reader.Environment(ctx, s.environment()) })
	group.Go(func() { app, errs[1] = g.reader.Application(ctx, s.Application) })
	group.Go(func() { cert, errs[2] = g.reader.Certificate(ctx, s.Certificate) })
	group.Go(func() { auth, errs[3] = g.reader.Auth(ctx, s.Application) })
	group.Go(func() { connections, errs[4] = g.reader.Connections(ctx, s.environment()) })
	group.Go(func() { links, errs[5] = g.reader.Links(ctx, s.environment()) })
	group.Go(func() { routes, errs[6] = g.reader.Routes(ctx, s.environment()) })
	group.Go(func() { revisions, errs[7] = g.reader.Revisions(ctx, s.Application) })
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
	if err := g.verifyEnvironment(env); err != nil {
		return err
	}
	observed, err := g.verifyExchangeConnections(connections)
	if err != nil {
		return err
	}
	embedded := []a.PrivateEndpointConnection{}
	for _, entry := range env.Properties.PrivateEndpointConnections {
		if entry == nil {
			return hostedexecution.ErrOwnership
		}
		embedded = append(embedded, *entry)
	}
	declared, err := g.verifyExchangeConnections(embedded)
	if err != nil {
		return err
	}
	if observed != declared {
		return hostedexecution.ErrUnavailable
	}
	if err := g.verifyExchangeLinks(links); err != nil {
		return err
	}
	if len(routes) != 0 {
		return hostedexecution.ErrOwnership
	}
	if !strings.EqualFold(value(auth.ID), s.auth()) || value(auth.Name) != "current" || !strings.EqualFold(value(auth.Type), "Microsoft.App/containerApps/authConfigs") {
		return hostedexecution.ErrOwnership
	}
	if auth.Properties == nil || auth.Properties.Platform == nil || auth.Properties.Platform.Enabled == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(auth.Properties.Platform.Enabled) {
		return hostedexecution.ErrOwnership
	}
	if err := g.verifyCertificate(cert, time.Now()); err != nil {
		return err
	}
	if err := g.verifyApplication(app, value(env.Properties.DefaultDomain)); err != nil {
		return err
	}
	return g.verifyRevisions(app, revisions)
}
