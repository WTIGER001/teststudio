package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ObserverHostReader interface {
	Application(context.Context, string) (a.ContainerApp, error)
	Revisions(context.Context, string) ([]a.Revision, error)
}

type ObserverHostGuard struct {
	worker   Config
	settings ObserverHostSettings
	coverage *ReadDenyGuard
	reader   ObserverHostReader
}

func NewObserverHostGuard(c Config, s ObserverHostSettings, coverage *ReadDenyGuard, reader ObserverHostReader) (*ObserverHostGuard, error) {
	s, err := s.canonical(c)
	if err != nil || coverage == nil || reader == nil || fingerprint(c) != fingerprint(coverage.worker) || fingerprint(s.Identity) != fingerprint(coverage.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &ObserverHostGuard{worker: c, settings: s, coverage: coverage, reader: reader}, nil
}

func (g *ObserverHostGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 135*time.Second)
	defer cancel()
	if err := g.coverage.Check(ctx, c); err != nil {
		return err
	}
	var app a.ContainerApp
	var revisions []a.Revision
	var errs [2]error
	var group sync.WaitGroup
	group.Go(func() { app, errs[0] = g.reader.Application(ctx, g.settings.Application) })
	group.Go(func() { revisions, errs[1] = g.reader.Revisions(ctx, g.settings.Application) })
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
	return verifyObserverHost(c, g.settings, app, revisions)
}
