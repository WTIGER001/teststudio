package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PreparationWriterReader interface {
	Identity(context.Context, string) (m.Identity, error)
	Assignment(context.Context, string) (a.RoleAssignment, error)
	Role(context.Context, string) (a.RoleDefinition, error)
}

type PreparationWriterGuard struct {
	worker     Config
	settings   PreparationWriterSettings
	registry   *PreparationRegistryGuard
	reader     PreparationWriterReader
	credential func(string) (azcore.TokenCredential, error)
}

func NewPreparationWriterGuard(c Config, s PreparationWriterSettings, registry *PreparationRegistryGuard, reader PreparationWriterReader) (*PreparationWriterGuard, error) {
	s, err := s.canonical(c)
	if err != nil || registry == nil || reader == nil || fingerprint(c) != fingerprint(registry.worker) || fingerprint(s.Registry) != fingerprint(registry.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &PreparationWriterGuard{worker: c, settings: s, registry: registry, reader: reader, credential: func(id string) (azcore.TokenCredential, error) {
		return azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ID: azidentity.ClientID(id)})
	}}, nil
}

func (g *PreparationWriterGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := g.registry.Check(ctx, c); err != nil {
		return err
	}
	var identity m.Identity
	var assignment a.RoleAssignment
	var role a.RoleDefinition
	var errs [3]error
	var group sync.WaitGroup
	group.Go(func() { identity, errs[0] = g.reader.Identity(ctx, g.settings.Identity) })
	group.Go(func() { assignment, errs[1] = g.reader.Assignment(ctx, g.settings.Assignment) })
	group.Go(func() { role, errs[2] = g.reader.Role(ctx, observerScope(c)+preparationPushRole) })
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
	return verifyPreparationWriter(c, g.settings, identity, assignment, role)
}

func (g *PreparationWriterGuard) OpenRegistry(ctx context.Context, c Config) (client *acr.Client, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if err := g.Check(ctx, c); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	credential, err := g.credential(g.settings.ClientID)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	return acr.New(acr.Options{Registry: c.Registry, Tenant: g.settings.TenantID, RepositoryPrefix: c.RepositoryPrefix, Publish: true, Credential: credential, BlobRedirectOrigins: []string{"https://" + g.settings.Registry.Registry.dataHost(c)}})
}
