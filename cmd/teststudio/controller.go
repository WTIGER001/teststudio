package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	artifactconfig "github.com/wtiger001/teststudio/adapters/artifacts/config"
	ecsworker "github.com/wtiger001/teststudio/adapters/execution/ecs"
	"github.com/wtiger001/teststudio/adapters/execution/kubernetes"
	httptrust "github.com/wtiger001/teststudio/adapters/httptrust/filesystem"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/identifier"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type controllerOptions struct {
	Address, Location, Connection, DatabaseEnvironment                                        string
	ArtifactConfiguration, ArtifactEnvironment, ArtifactDirectory, PublicOrigin, TrustBundles string
	ECSConfiguration, ECSEnvironment, KubernetesConfiguration, KubernetesEnvironment          string
}

func validateControllerFlags(flags *flag.FlagSet, address, location string, ecs, kubernetes bool) error {
	invalid := flags.NArg() != 0 || ecs && kubernetes || ecs && location != "ecs" || kubernetes && location != "kubernetes" || !ecs && !kubernetes && location != "local"
	explicitAddress := false
	flags.Visit(func(value *flag.Flag) {
		switch value.Name {
		case "addr":
			explicitAddress = true
		case "serve-role", "execution-location", "database", "database-env", "require-postgres", "artifact-directory", "artifact-store-config", "artifact-store-env", "public-url", "http-trust-bundles", "ecs-workers-config", "ecs-workers-env", "kubernetes-workers-config", "kubernetes-workers-env":
		default:
			invalid = true
		}
	})
	_, port, err := net.SplitHostPort(address)
	number, numberErr := strconv.Atoi(port)
	if invalid || !explicitAddress || err != nil || numberErr != nil || number < 0 || number > 65535 || !identifier.Valid(location) {
		return errors.New("the controller role requires an explicit numeric addr and matching local, ECS, or Kubernetes location, with database, artifacts, origin, and optional HTTPS trust settings only")
	}
	return nil
}

func runController(ctx context.Context, o controllerOptions) error {
	if err := notifications.ValidateOrigin(o.PublicOrigin); err != nil {
		return err
	}
	ecsSettings, err := ecsworker.LoadSettings(o.ECSConfiguration, o.ECSEnvironment, os.LookupEnv)
	if err != nil {
		return err
	}
	kubeSettings, err := kubernetes.LoadSettings(o.KubernetesConfiguration, o.KubernetesEnvironment, os.LookupEnv)
	if err != nil {
		return err
	}
	if err := runtimeconfig.RequirePostgres(o.Connection, ecsSettings != nil || kubeSettings != nil); err != nil {
		return err
	}
	trust, err := httptrust.Load(o.TrustBundles)
	if err != nil {
		return err
	}
	settings, err := artifactconfig.LoadEnvironment(o.ArtifactConfiguration, o.ArtifactEnvironment, o.ArtifactDirectory, os.LookupEnv)
	if err != nil {
		return err
	}
	db, auth, closeStorage, err := openServiceStorage(ctx, o.Connection, o.DatabaseEnvironment)
	if err != nil {
		return err
	}
	defer closeStorage()
	blobs, closeArtifacts, err := artifactconfig.Open(ctx, settings)
	if err != nil {
		return err
	}
	defer closeArtifacts()
	run, source, exchange := controllerServices(db, auth, blobs, o.Location, o.PublicOrigin, trust)
	if ecsSettings != nil {
		runtime, err := ecsSettings.Open(ctx, db, source.AuthorizeDiscoveryCleanup)
		if err != nil {
			return err
		}
		defer runtime.Close()
		configureHostedWorkers(run, source, exchange, runtime.Provider, nil, "ecs", ecsSettings.Worker.ExchangeOrigin, false)
	}
	if kubeSettings != nil {
		provider, err := kubeSettings.Open(db, os.LookupEnv, source.AuthorizeDiscoveryCleanup)
		if err != nil {
			return err
		}
		defer provider.Close()
		configureHostedWorkers(run, source, exchange, provider, nil, "kubernetes", kubeSettings.Worker.ExchangeOrigin, false)
	}
	return serveController(ctx, o.Address, db, run)
}

func controllerServices(db *storage.Store, auth *authorization.Service, blobs artifacts.Store, location, origin string, trust execution.HTTPTrustResolver) (*runs.Service, *sources.Service, *workerexchange.Service) {
	source := &sources.Service{Store: db, Auth: auth, Location: location}
	run := &runs.Service{Store: db, Auth: auth, Catalog: &catalog.Service{Store: db, Auth: auth}, Evidence: &evidence.Service{Store: db, Blobs: blobs}, HTTP: &execution.HTTP{Trust: trust}, Location: location, Completion: &notifications.Service{Store: db, Auth: auth, Origin: origin}}
	exchange := &workerexchange.Service{Store: db, Blobs: blobs}
	configureWorkerAuthorization(run, source, exchange)
	return run, source, exchange
}

func serveController(ctx context.Context, address string, db *storage.Store, run *runs.Service) error {
	server := &http.Server{Addr: address, Handler: serviceHealthHandler(db, "controller"), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	return serveServers(ctx, []*http.Server{server}, func() func() {
		worker, cancel := context.WithCancel(ctx)
		done := run.Start(worker)
		return func() { cancel(); <-done }
	})
}
