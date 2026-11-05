package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
)

type discoveryRegistrationProvider struct{ hostedexecution.Provider }

func TestDiscoveryOnlyRejectsMixedRolesBeforeConfigurationOrState(t *testing.T) {
	for _, extra := range [][]string{{"--assembly-only"}, {"--registry-go-runtime=writer-runtime"}, {"--go-runtime=local-image"}, {"--build-directory=private-journal"}, {"--github-sources"}, {"--secret-bindings=private-secrets"}, {"--azure-assembly-config=private-file"}, {"--execution-location=local"}, {"--ecs-workers-config=private-file"}} {
		t.Run(strings.Join(extra, " "), func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
			flag.CommandLine = flag.NewFlagSet("owned-discovery-check", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			database := filepath.Join(t.TempDir(), "must-not-exist.db")
			os.Args = append([]string{"teststudio-prepare", "--discovery-only", "--kubernetes-workers-config=unread-settings", "--execution-location=kubernetes", "--database=" + database}, extra...)
			err := run()
			if err == nil || !strings.Contains(err.Error(), "discovery-only preparation requires") || strings.Contains(err.Error(), "private-") {
				t.Fatal("invalid discovery role reached configuration or leaked its values", err)
			}
			if _, err := os.Stat(database); !os.IsNotExist(err) {
				t.Fatal("invalid discovery role created a database", err)
			}
		})
	}
	for _, name := range []string{"ecs", "kubernetes"} {
		flags := flag.NewFlagSet("valid-discovery", flag.ContinueOnError)
		flags.Bool("discovery-only", false, "")
		flags.String("execution-location", "", "")
		flags.String(name+"-workers-config", "", "")
		if err := flags.Parse([]string{"--discovery-only", "--execution-location=" + name, "--" + name + "-workers-config=unread-settings"}); err != nil {
			t.Fatal(err)
		}
		if err := validateDiscoveryOnlyFlags(flags, true, false, name, name == "ecs", name == "kubernetes"); err != nil {
			t.Fatal("valid isolated discovery settings rejected", err)
		}
	}
}

func TestDiscoveryOnlyAttachesBothInspectorsWithoutBuilders(t *testing.T) {
	service := &sources.Service{}
	provider := &discoveryRegistrationProvider{}
	if err := attachHostedDiscovery(service, provider, "owned", "https://exchange.studio.example"); err != nil {
		t.Fatal(err)
	}
	if len(service.Builders)+len(service.RecoveryBuilders) != 0 || len(service.Inspectors) != 2 || len(service.RecoveryInspectors) != 2 {
		t.Fatal("discovery role gained builders or lost recovery")
	}
	for _, runner := range []string{"go", "playwright"} {
		inspector, ok := service.Inspectors[runner].(*hostedexecution.DiscoveryInspector)
		if !ok || inspector.Provider != provider || inspector.Runner != runner || service.RecoveryInspectors[runner] != inspector {
			t.Fatal("discovery runner or recovery uses another provider")
		}
	}
	service.Builders = map[string]preparation.ArtifactBuilder{"go": &assemblyDelegate{}}
	if err := attachHostedDiscovery(service, provider, "owned", "https://exchange.studio.example"); err != hostedexecution.ErrConfiguration {
		t.Fatal("discovery mode accepted a registry builder", err)
	}
}

func TestECSPreparationAttachesAddressCapabilityWithoutGatewayExecution(t *testing.T) {
	service, provider := &sources.Service{}, &discoveryRegistrationProvider{}
	if err := attachHostedDiscovery(service, provider, "ecs", "https://exchange.studio.example"); err != nil {
		t.Fatal(err)
	}
	for _, runner := range []string{"go", "playwright"} {
		inspector := service.Inspectors[runner].(*hostedexecution.DiscoveryInspector)
		if !inspector.RequireGatewayAddress || inspector.Provider != provider || service.RecoveryInspectors[runner] != inspector {
			t.Fatal("ECS preparation lost its offline discovery or capability boundary")
		}
	}
	if len(service.Builders)+len(service.RecoveryBuilders) != 0 {
		t.Fatal("dedicated discovery acquired assembly authority")
	}
}
