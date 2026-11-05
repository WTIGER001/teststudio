package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	r "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationHostReader interface {
	Application(context.Context, string) (a.ContainerApp, error)
	Environment(context.Context, string) (a.ManagedEnvironment, error)
	Revisions(context.Context, string) ([]a.Revision, error)
	Storage(context.Context, string) (a.ManagedEnvironmentStorage, error)
	Identity(context.Context, string) (m.Identity, error)
	Assignment(context.Context, string) (r.RoleAssignment, error)
	Role(context.Context, string) (r.RoleDefinition, error)
}

type PreparationHostGuard struct {
	settings PreparationSettings
	writer   *PreparationWriterGuard
	reader   PreparationHostReader
}

func NewPreparationHostGuard(s PreparationSettings, writer *PreparationWriterGuard, reader PreparationHostReader) (*PreparationHostGuard, error) {
	if s.validateHost() != nil || writer == nil || reader == nil || fingerprint(s.Worker) != fingerprint(writer.worker) {
		return nil, hostedexecution.ErrConfiguration
	}
	var err error
	s.Writer, err = s.Writer.canonical(s.Worker)
	if err != nil || fingerprint(s.Writer) != fingerprint(writer.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	host, _ := s.Host.canonical(s.Worker, s.Writer, s.ReaderClientID)
	s.Host = &host
	return &PreparationHostGuard{settings: s, writer: writer, reader: reader}, nil
}

func (g *PreparationHostGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	if err := g.writer.Check(ctx, c); err != nil {
		return err
	}
	h := g.settings.Host
	var app a.ContainerApp
	var environment a.ManagedEnvironment
	var revisions []a.Revision
	var storage a.ManagedEnvironmentStorage
	var identity m.Identity
	var assignment r.RoleAssignment
	var role r.RoleDefinition
	var errs [7]error
	var group sync.WaitGroup
	group.Go(func() { app, errs[0] = g.reader.Application(ctx, h.Application) })
	group.Go(func() { environment, errs[1] = g.reader.Environment(ctx, h.Environment) })
	group.Go(func() { revisions, errs[2] = g.reader.Revisions(ctx, h.Application) })
	group.Go(func() { storage, errs[3] = g.reader.Storage(ctx, h.storage()) })
	group.Go(func() { identity, errs[4] = g.reader.Identity(ctx, h.ReaderIdentity) })
	group.Go(func() { assignment, errs[5] = g.reader.Assignment(ctx, h.ReaderAssignment) })
	group.Go(func() { role, errs[6] = g.reader.Role(ctx, h.ReaderRole) })
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
	if err := verifyReadIdentityBinding(c, h.reader(g.settings), identity, assignment, role); err != nil {
		return err
	}
	if err := verifyPreparationHostPlacement(g.settings, environment, storage); err != nil {
		return err
	}
	return verifyPreparationHostApplication(g.settings, app, revisions)
}
