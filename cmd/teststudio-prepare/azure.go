package main

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/adapters/preparation/registryimage"
	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
)

type azureAssemblyBuilder struct {
	check   func(context.Context) error
	builder preparation.AssemblyBuilder
}

func (b *azureAssemblyBuilder) Build(ctx context.Context, request preparation.BuildRequest) (preparation.Build, error) {
	if err := b.check(ctx); err != nil {
		return preparation.Build{}, err
	}
	return b.builder.Build(ctx, request)
}

func (b *azureAssemblyBuilder) CleanupBuild(ctx context.Context, ref execution.Reference) error {
	return b.builder.CleanupBuild(ctx, ref)
}

func (b *azureAssemblyBuilder) CompleteAssembly(ctx context.Context, ref execution.Reference) error {
	return b.builder.CompleteAssembly(ctx, ref)
}

func validateAzureRuntimes(settings azureworker.PreparationSettings, runtimes map[string]string) error {
	if settings.ValidateNative() != nil {
		return hostedexecution.ErrConfiguration
	}
	if runtimes["go"] != settings.Host.GoRuntime || runtimes["playwright"] != settings.Host.PlaywrightRuntime {
		return hostedexecution.ErrConfiguration
	}
	registry, err := oci.New(oci.Options{Origin: "https://" + settings.Worker.Registry, RepositoryPrefix: settings.Worker.RepositoryPrefix, Architecture: "amd64"})
	if err != nil {
		return hostedexecution.ErrConfiguration
	}
	defer registry.Close()
	for _, image := range runtimes {
		if image != "" && registry.ValidateReference(image) != nil {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}

func configureAzurePreparation(ctx context.Context, service *sources.Service, settings azureworker.PreparationSettings, directory string, runtimes map[string]string) (func(), error) {
	if !service.AssemblyOnly || service.Location != azureworker.Location || directory != azureworker.PreparationJournalDirectory || validateAzureRuntimes(settings, runtimes) != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	credential, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ID: azidentity.ClientID(settings.ReaderClientID)})
	if err != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	publisher, err := settings.OpenPublisher(credential, nil)
	if err != nil {
		return nil, err
	}
	startup, stop := context.WithTimeout(ctx, 3*time.Minute)
	defer stop()
	registry, err := awaitAzureRegistry(startup, func(ctx context.Context) (*acr.Client, error) { return publisher.OpenRegistry(ctx, settings.Worker) }, 2*time.Second)
	if err != nil {
		publisher.Close()
		return nil, err
	}
	close := func() { registry.Close(); publisher.Close() }
	if err := attachAzureAssembly(service, registry.Client, directory, runtimes, func(ctx context.Context) error { return publisher.Check(ctx, settings.Worker) }); err != nil {
		close()
		return nil, err
	}
	return close, nil
}

func attachAzureAssembly(service *sources.Service, registry *oci.Client, directory string, runtimes map[string]string, check func(context.Context) error) error {
	if !service.AssemblyOnly || service.Location != azureworker.Location || registry == nil || check == nil || len(service.Inspectors)+len(service.RecoveryInspectors) != 0 {
		return hostedexecution.ErrConfiguration
	}
	for _, image := range runtimes {
		if image != "" && registry.ValidateReference(image) != nil {
			return hostedexecution.ErrConfiguration
		}
	}
	for _, runner := range []string{"go", "playwright"} {
		builder := &azureAssemblyBuilder{check: check, builder: &registryimage.Builder{Registry: registry, RuntimeImage: runtimes[runner], Directory: directory}}
		service.RecoveryBuilders[runner] = builder
		if runtimes[runner] != "" {
			service.Builders[runner] = builder
		}
	}
	return nil
}
