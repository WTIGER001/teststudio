package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	permissions "github.com/wtiger001/go-permissions"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	ecsworker "github.com/wtiger001/teststudio/adapters/execution/ecs"
	"github.com/wtiger001/teststudio/adapters/execution/kubernetes"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	database := flag.String("database", ".data/teststudio.db", "SQLite path or PostgreSQL URL shared with the API")
	databaseEnvironment := flag.String("database-env", "", "Environment variable containing the database connection")
	requirePostgres := flag.Bool("require-postgres", false, "Require PostgreSQL for a hosted or multi-instance installation")
	ecsConfig := flag.String("ecs-workers-config", "", "Private ECS worker and network preparation settings")
	ecsEnv := flag.String("ecs-workers-env", "", "Environment variable containing private ECS preparation settings")
	kubernetesConfig := flag.String("kubernetes-workers-config", "", "Private Kubernetes and registry preparation settings")
	kubernetesEnv := flag.String("kubernetes-workers-env", "", "Environment variable containing private Kubernetes preparation settings")
	azureConfig := flag.String("azure-assembly-config", "", "Private Azure assembly-only settings")
	azureEnv := flag.String("azure-assembly-env", "", "Environment variable containing private Azure assembly-only settings")
	assemblyOnly := flag.Bool("assembly-only", false, "Publish registry source bundles and queue isolated discovery without running it in this process")
	registryGo := flag.String("registry-go-runtime", "", "Trusted supervised Go runtime distribution digest")
	registryPlaywright := flag.String("registry-playwright-runtime", "", "Trusted supervised Playwright runtime distribution digest")
	location := flag.String("execution-location", "local", "Preparation queue location")
	browserRuntime := flag.String("playwright-runtime", "", "Trusted immutable local Playwright source runtime image ID")
	runtime := flag.String("go-runtime", "", "Trusted immutable local Go source runtime image ID")
	directory := flag.String("build-directory", ".data/source-builds", "Private local source context directory shared by this location's workers")
	githubSources := flag.Bool("github-sources", false, "Process durable GitHub source synchronization")
	secretBindings := flag.String("secret-bindings", "", "Project-scoped repository access credential bindings")
	flag.Parse()
	connection, err := runtimeconfig.DatabaseConnection(*database, *databaseEnvironment, runtimeconfig.ExplicitFlag(flag.CommandLine, "database"), os.LookupEnv)
	if err != nil {
		return err
	}
	providers := 0
	for _, configured := range []bool{*ecsConfig != "" || *ecsEnv != "", *kubernetesConfig != "" || *kubernetesEnv != "", *azureConfig != "" || *azureEnv != ""} {
		if configured {
			providers++
		}
	}
	if providers > 1 {
		return errors.New("choose one hosted execution provider")
	}
	if (*azureConfig != "" || *azureEnv != "") && (!*assemblyOnly || *location != azureworker.Location || *runtime != "" || *browserRuntime != "") {
		return errors.New("Azure preparation requires assembly-only mode and its execution location without local source runtime flags")
	}
	azureSettings, err := azureworker.LoadPreparationSettings(*azureConfig, *azureEnv, os.LookupEnv)
	if err != nil {
		return err
	}
	if azureSettings != nil && azureSettings.ValidateNative() != nil {
		return errors.New("Azure assembly requires valid private host and network settings with a separate explicit reader client identifier")
	}
	if azureSettings != nil && *directory != azureworker.PreparationJournalDirectory {
		return errors.New("Azure assembly requires the configured private journal mount directory")
	}
	if azureSettings != nil && validateAzureRuntimes(*azureSettings, map[string]string{"go": *registryGo, "playwright": *registryPlaywright}) != nil {
		return errors.New("Azure assembly runtimes require immutable distribution digests in the configured registry and repository prefix")
	}
	ecsSettings, err := ecsworker.LoadSettings(*ecsConfig, *ecsEnv, os.LookupEnv)
	if err != nil {
		return err
	}
	if ecsSettings != nil && (*location != "ecs" || *runtime != "" || *browserRuntime != "") {
		return errors.New("ECS preparation requires its execution location without local source runtime flags")
	}
	kubernetesSettings, err := kubernetes.LoadSettings(*kubernetesConfig, *kubernetesEnv, os.LookupEnv)
	if err != nil {
		return err
	}
	if kubernetesSettings != nil && (*location != "kubernetes" || *runtime != "" || *browserRuntime != "") || kubernetesSettings == nil && ecsSettings == nil && azureSettings == nil && (*registryGo != "" || *registryPlaywright != "") {
		return errors.New("hosted preparation requires provider settings and the matching location without local source runtime flags")
	}
	if *assemblyOnly && ecsSettings == nil && kubernetesSettings == nil && azureSettings == nil {
		return errors.New("assembly-only preparation requires hosted registry settings; local image cleanup cannot preserve its artifact for handoff")
	}
	if err := runtimeconfig.RequirePostgres(connection, *requirePostgres || kubernetesSettings != nil || ecsSettings != nil || azureSettings != nil); err != nil {
		return err
	}
	if !domain.ValidID(*location) {
		return errors.New("a valid preparation location is required")
	}
	root, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return errors.New("build-directory must be a private directory without a symbolic link")
	}
	if azureSettings != nil {
		if err := validateAzureJournal(root, azureSettings.Host.JournalAccount, azureSettings.Host.JournalShare); err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := storage.Open(ctx, connection)
	if err != nil {
		return runtimeconfig.DatabaseError(err, *databaseEnvironment)
	}
	defer store.Close()
	var permissionStore permissions.PermissionStore = permissionsqlite.New(store)
	if store.Dialect == "postgres" {
		pg, err := permissionpg.Open(ctx, connection)
		if err != nil {
			return runtimeconfig.DatabaseError(err, *databaseEnvironment)
		}
		defer pg.Close()
		permissionStore = pg
	}
	auth, err := authorization.New(ctx, permissionStore, &identity.Directory{Store: store})
	if err != nil {
		return err
	}
	service := &sources.Service{AssemblyOnly: *assemblyOnly, Store: store, Auth: auth, Location: *location, Inspectors: map[string]preparation.ArtifactInspector{}, RecoveryInspectors: map[string]preparation.ArtifactInspector{}, Builders: map[string]preparation.ArtifactBuilder{}, RecoveryBuilders: map[string]preparation.ArtifactBuilder{}}
	if azureSettings != nil {
		close, err := configureAzurePreparation(ctx, service, *azureSettings, root, map[string]string{"go": *registryGo, "playwright": *registryPlaywright})
		if err != nil {
			return err
		}
		defer close()
	} else if ecsSettings != nil {
		runtime, err := ecsSettings.Open(ctx, store, service.AuthorizeDiscoveryCleanup)
		if err != nil {
			return err
		}
		defer runtime.Close()
		if err := attachRegistryPreparation(service, runtime.Provider, runtime.Registry, "ecs", ecsSettings.Worker.ExchangeOrigin, root, map[string]string{"go": *registryGo, "playwright": *registryPlaywright}); err != nil {
			return err
		}
	} else if kubernetesSettings != nil {
		close, err := configureRegistryPreparation(service, *kubernetesSettings, root, map[string]string{"go": *registryGo, "playwright": *registryPlaywright}, os.LookupEnv)
		if err != nil {
			return err
		}
		defer close()
	} else {
		for runner, image := range map[string]string{"go": *runtime, "playwright": *browserRuntime} {
			inspector := &localcontainer.Executor{Runner: runner, AllowProjectImages: true, RequireProject: true}
			builder := &localimage.Builder{RuntimeImage: image, TempDirectory: root}
			service.RecoveryInspectors[runner] = inspector
			service.RecoveryBuilders[runner] = builder
			if image != "" {
				service.Inspectors[runner] = inspector
				service.Builders[runner] = builder
			}
		}
	}
	for _, runner := range []string{"go", "playwright"} {
		if service.Builders[runner] != nil {
			service.BundleRunners = append(service.BundleRunners, runner)
		}
	}
	var syncDone <-chan struct{}
	if *githubSources && len(service.Builders) > 0 {
		provider, err := localsecrets.Load(*secretBindings)
		if err != nil {
			return err
		}
		service.Secrets = provider
		service.Providers = map[string]sourceprovider.Provider{"github": &githubsource.Client{Secrets: provider}}
		syncDone = service.StartRepositorySynchronization(ctx)
	}
	done := service.StartBuildPreparation(ctx)
	<-done
	if syncDone != nil {
		<-syncDone
	}
	return nil
}
