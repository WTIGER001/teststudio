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
	kubernetesConfig := flag.String("kubernetes-workers-config", "", "Private Kubernetes and registry preparation settings")
	kubernetesEnv := flag.String("kubernetes-workers-env", "", "Environment variable containing private Kubernetes preparation settings")
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
	kubernetesSettings, err := kubernetes.LoadSettings(*kubernetesConfig, *kubernetesEnv, os.LookupEnv)
	if err != nil {
		return err
	}
	if kubernetesSettings != nil && (*location != "kubernetes" || *runtime != "" || *browserRuntime != "") || kubernetesSettings == nil && (*registryGo != "" || *registryPlaywright != "") {
		return errors.New("hosted preparation requires Kubernetes settings and its location without local source runtime flags")
	}
	if err := runtimeconfig.RequirePostgres(connection, *requirePostgres || kubernetesSettings != nil); err != nil {
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
	service := &sources.Service{Store: store, Auth: auth, Location: *location, Inspectors: map[string]preparation.ArtifactInspector{}, RecoveryInspectors: map[string]preparation.ArtifactInspector{}, Builders: map[string]preparation.ArtifactBuilder{}, RecoveryBuilders: map[string]preparation.ArtifactBuilder{}}
	if kubernetesSettings != nil {
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
