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
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ReadIdentityReader interface {
	Identity(context.Context, string) (m.Identity, error)
	Assignment(context.Context, string) (a.RoleAssignment, error)
	Role(context.Context, string) (a.RoleDefinition, error)
	FederationAbsent(context.Context, string) error
	Protection(context.Context, string) (a.DenyAssignment, error)
	Stack(context.Context, string) (d.DeploymentStack, error)
}

type ReadIdentityGuard struct {
	worker     Config
	settings   ReadIdentitySettings
	protection *NetworkDenyGuard
	reader     ReadIdentityReader
	credential func(string) (azcore.TokenCredential, error)
}

func NewReadIdentityGuard(c Config, s ReadIdentitySettings, protection *NetworkDenyGuard, reader ReadIdentityReader) (*ReadIdentityGuard, error) {
	s, err := s.canonical(c)
	if err != nil || protection == nil || reader == nil || fingerprint(c) != fingerprint(protection.worker) || fingerprint(s.Protection) != fingerprint(protection.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &ReadIdentityGuard{worker: c, settings: s, protection: protection, reader: reader, credential: func(id string) (azcore.TokenCredential, error) {
		return azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ID: azidentity.ClientID(id)})
	}}, nil
}

func (g *ReadIdentityGuard) Check(ctx context.Context, c Config) error {
	_, err := g.inspect(ctx, c)
	return err
}

func (g *ReadIdentityGuard) Credential(ctx context.Context, c Config) (credential azcore.TokenCredential, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	identity, err := g.inspect(ctx, c)
	if err != nil {
		return nil, err
	}
	return g.credential(value(identity.Properties.ClientID))
}

func (g *ReadIdentityGuard) inspect(ctx context.Context, c Config) (identity m.Identity, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return m.Identity{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 125*time.Second)
	defer cancel()
	if err := g.protection.Check(ctx, c); err != nil {
		return m.Identity{}, err
	}
	var assignment a.RoleAssignment
	var role a.RoleDefinition
	var protection a.DenyAssignment
	var stacks [2]d.DeploymentStack
	var errs [7]error
	var group sync.WaitGroup
	group.Go(func() { identity, errs[0] = g.reader.Identity(ctx, g.settings.Identity) })
	group.Go(func() { assignment, errs[1] = g.reader.Assignment(ctx, g.settings.Assignment) })
	group.Go(func() { role, errs[2] = g.reader.Role(ctx, g.settings.Role) })
	group.Go(func() { errs[3] = g.reader.FederationAbsent(ctx, g.settings.Identity) })
	group.Go(func() { protection, errs[4] = g.reader.Protection(ctx, g.settings.ProtectionAssignment) })
	bindings := g.settings.stackBindings()
	for i, binding := range bindings {
		group.Go(func() { stacks[i], errs[i+5] = g.reader.Stack(ctx, binding.id) })
	}
	group.Wait()
	for _, err := range errs {
		if errors.Is(err, hostedexecution.ErrOwnership) {
			return m.Identity{}, hostedexecution.ErrOwnership
		}
	}
	for _, err := range errs {
		if err != nil {
			return m.Identity{}, hostedexecution.ErrUnavailable
		}
	}
	if err := verifyReadIdentity(c, g.settings, identity, assignment, role); err != nil {
		return m.Identity{}, err
	}
	if err := verifyReadIdentityProtection(g.settings, protection); err != nil {
		return m.Identity{}, err
	}
	for i, binding := range bindings {
		if err := verifyProtectionStack(c, binding, stacks[i]); err != nil {
			return m.Identity{}, err
		}
	}
	return identity, nil
}
