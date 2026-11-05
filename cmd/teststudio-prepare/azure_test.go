package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
)

type assemblyDelegate struct {
	builds, cleanups, completions int
	request                       preparation.BuildRequest
	ref                           execution.Reference
}

func (b *assemblyDelegate) Build(_ context.Context, request preparation.BuildRequest) (preparation.Build, error) {
	b.builds++
	b.request = request
	return preparation.Build{Image: "owned-assembly-result"}, nil
}
func (b *assemblyDelegate) CleanupBuild(_ context.Context, ref execution.Reference) error {
	b.cleanups++
	b.ref = ref
	return nil
}
func (b *assemblyDelegate) CompleteAssembly(_ context.Context, ref execution.Reference) error {
	b.completions++
	b.ref = ref
	return nil
}

func TestAzureAssemblyRefreshesBeforeBuildAndPreservesOwnedCleanup(t *testing.T) {
	delegate := &assemblyDelegate{}
	checks := 0
	var gate error
	builder := &azureAssemblyBuilder{builder: delegate, check: func(context.Context) error { checks++; return gate }}
	request := preparation.BuildRequest{Reference: execution.Reference{ID: "owned-assembly", Project: "owned-project", Installation: "preparation-azure"}}
	for i := 1; i <= 2; i++ {
		build, err := builder.Build(t.Context(), request)
		if err != nil || build.Image != "owned-assembly-result" || delegate.builds != i || checks != i || delegate.request.Reference != request.Reference {
			t.Fatal("fresh admission or source ownership was lost", err)
		}
	}
	gate = hostedexecution.ErrOwnership
	_, err := builder.Build(t.Context(), request)
	if !errors.Is(err, gate) || delegate.builds != 2 || checks != 3 {
		t.Fatal("changed registry admission reached source publication", err)
	}
	if err := builder.CleanupBuild(t.Context(), request.Reference); err != nil {
		t.Fatal(err)
	}
	if err := builder.CompleteAssembly(t.Context(), request.Reference); err != nil || delegate.ref != request.Reference || delegate.cleanups != 1 || delegate.completions != 1 || checks != 3 {
		t.Fatal("owned journal recovery depended on new control-plane admission", err)
	}
}

func TestAzureAssemblyAttachesBothRunnersWithoutDiscovery(t *testing.T) {
	registry, err := oci.New(oci.Options{Origin: "https://ownedregistry.azurecr.io", RepositoryPrefix: "teststudio", Architecture: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registry.Close)
	service := &sources.Service{AssemblyOnly: true, Location: azureworker.Location, Builders: map[string]preparation.ArtifactBuilder{}, RecoveryBuilders: map[string]preparation.ArtifactBuilder{}}
	image := "ownedregistry.azurecr.io/teststudio/runtime@sha256:" + strings.Repeat("a", 64)
	err = attachAzureAssembly(service, registry, t.TempDir(), map[string]string{"go": image, "playwright": image}, func(context.Context) error { return nil })
	if err != nil || len(service.Builders) != 2 || len(service.RecoveryBuilders) != 2 || len(service.Inspectors)+len(service.RecoveryInspectors) != 0 {
		t.Fatal("Azure assembly configured discovery or lost a runner", err)
	}
	for _, runner := range []string{"go", "playwright"} {
		if _, ok := service.Builders[runner].(preparation.AssemblyBuilder); !ok {
			t.Fatal("Azure registry builder lost its handoff capability", runner)
		}
	}
	service.AssemblyOnly = false
	if attachAzureAssembly(service, registry, t.TempDir(), nil, func(context.Context) error { return nil }) == nil {
		t.Fatal("combined Azure discovery mode was admitted")
	}
	service.AssemblyOnly = true
	service.Builders = map[string]preparation.ArtifactBuilder{}
	service.RecoveryBuilders = map[string]preparation.ArtifactBuilder{}
	err = attachAzureAssembly(service, registry, t.TempDir(), map[string]string{"go": "foreign.azurecr.io/teststudio/runtime@sha256:" + strings.Repeat("a", 64)}, func(context.Context) error { return nil })
	if err == nil || len(service.Builders)+len(service.RecoveryBuilders) != 0 {
		t.Fatal("invalid runtime made a partial assembly configuration")
	}
}

func TestAzureAssemblyRejectsInvalidNativeModesBeforeStateOrCredentials(t *testing.T) {
	for name, arguments := range map[string][]string{
		"combined":        {"--azure-assembly-config=unread-private-file", "--execution-location=azure-container-apps"},
		"location":        {"--azure-assembly-config=unread-private-file", "--assembly-only"},
		"local-runtime":   {"--azure-assembly-config=unread-private-file", "--assembly-only", "--execution-location=azure-container-apps", "--go-runtime=local-image"},
		"mixed-providers": {"--azure-assembly-config=unread-private-file", "--ecs-workers-config=unread-other-file", "--assembly-only"},
		"file-and-env":    {"--azure-assembly-config=unread-private-file", "--azure-assembly-env=TESTSTUDIO_AZURE_ASSEMBLY", "--assembly-only", "--execution-location=azure-container-apps"},
	} {
		t.Run(name, func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
			flag.CommandLine = flag.NewFlagSet("owned-azure-assembly-check", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			root := t.TempDir()
			database, directory := filepath.Join(root, "must-not-exist.db"), filepath.Join(root, "must-not-exist")
			os.Args = append([]string{"teststudio-prepare", "--database=" + database, "--build-directory=" + directory}, arguments...)
			err := run()
			if err == nil || strings.Contains(err.Error(), "cannot be read") {
				t.Fatal("invalid Azure mode reached configuration IO", err)
			}
			for _, path := range []string{database, directory} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("invalid Azure mode created state", path, err)
				}
			}
		})
	}
}
