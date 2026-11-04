package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	artifactconfig "github.com/wtiger001/teststudio/adapters/artifacts/config"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/identifier"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func validateWorkerBackendFlags(flags *flag.FlagSet, address, location string) error {
	invalid := flags.NArg() != 0
	flags.Visit(func(value *flag.Flag) {
		switch value.Name {
		case "serve-role", "worker-addr", "execution-location", "database", "database-env", "require-postgres", "artifact-directory", "artifact-store-config", "artifact-store-env", "worker-origin", "worker-tls-certificate", "worker-tls-key":
		default:
			invalid = true
		}
	})
	if invalid {
		return errors.New("the worker service role accepts only worker address, execution location, database, artifact storage, and direct worker TLS settings")
	}
	_, port, err := net.SplitHostPort(address)
	number, numberErr := strconv.Atoi(port)
	if err != nil || numberErr != nil || number < 0 || number > 65535 {
		return errors.New("the worker service role requires an explicit worker-addr with a numeric port")
	}
	if !identifier.Valid(location) {
		return errors.New("the worker service role requires a valid execution-location identifier")
	}
	return nil
}

func runWorkerBackend(ctx context.Context, address, location, connection, databaseEnvironment, artifactConfiguration, artifactEnvironment, artifactDirectory string, options workerTLSOptions) error {
	tlsConfig, authority, err := options.load(time.Now())
	if err != nil {
		return err
	}
	settings, err := artifactconfig.LoadEnvironment(artifactConfiguration, artifactEnvironment, artifactDirectory, os.LookupEnv)
	if err != nil {
		return err
	}
	db, auth, closeStorage, err := openServiceStorage(ctx, connection, databaseEnvironment)
	if err != nil {
		return err
	}
	defer closeStorage()
	blobs, closeArtifacts, err := artifactconfig.Open(ctx, settings)
	if err != nil {
		return err
	}
	defer closeArtifacts()
	exchange := workerBackendExchange(db, auth, blobs, location)
	server := workerBackendServer(address, db, exchange)
	server.TLSConfig = tlsConfig
	server.Handler = workerAuthorityHandler(server.Handler, authority)
	return serveServers(ctx, []*http.Server{server}, nil)
}

func workerBackendExchange(db *storage.Store, auth *authorization.Service, blobs artifacts.Store, location string) *workerexchange.Service {
	run := &runs.Service{Store: db, Auth: auth, Catalog: &catalog.Service{Store: db, Auth: auth}, Location: location}
	source := &sources.Service{Store: db, Auth: auth, Location: location}
	exchange := &workerexchange.Service{Store: db, Blobs: blobs}
	configureWorkerAuthorization(run, source, exchange)
	return exchange
}

func configureWorkerAuthorization(run *runs.Service, source *sources.Service, exchange *workerexchange.Service) {
	exchange.Authorize = func(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
		if strings.HasSuffix(ref.Profile, "-discovery-v1") {
			return source.AuthorizeDiscovery(ctx, ref)
		}
		return run.AuthorizeWorkerAttempt(ctx, ref)
	}
	exchange.AuthorizeDiscoveryCleanup = source.AuthorizeDiscoveryCleanup
}

func workerBackendServer(address string, db *storage.Store, exchange *workerexchange.Service) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/internal/worker/", workerhttp.Handler(exchange))
	health := func(ready bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if ready {
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				defer cancel()
				if err := db.DB.PingContext(ctx); err != nil {
					w.WriteHeader(http.StatusServiceUnavailable)
					json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
					return
				}
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "up", "service": "teststudio"})
		}
	}
	mux.HandleFunc("GET /healthz", health(true))
	mux.HandleFunc("GET /livez", health(false))
	return &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 16 << 10}
}
