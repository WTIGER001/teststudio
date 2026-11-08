package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/adapters/recording/isolatedcontainer"
	"github.com/wtiger001/teststudio/adapters/recording/workerhttp"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/recordingworker"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
	"github.com/wtiger001/teststudio/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", "127.0.0.1:8045", "Private recording worker HTTPS address")
	id := flag.String("worker-id", "recording", "Stable recording worker identity")
	database := flag.String("database", "", "Local verification database path")
	databaseEnv := flag.String("database-env", "", "Environment variable containing the shared PostgreSQL connection")
	certificate := flag.String("tls-certificate", "", "Mounted server certificate chain")
	key := flag.String("tls-key", "", "Mounted server private key")
	clientCA := flag.String("client-ca", "", "Dedicated API client certificate authority")
	clientName := flag.String("client-name", "teststudio-api.recording.internal", "Approved API client certificate DNS identity")
	image := flag.String("runtime-image", "", "Immutable isolated recording runtime image")
	gateway := flag.String("gateway-image", "", "Immutable trusted gateway image")
	seccomp := flag.String("seccomp", "", "Pinned Chromium sandbox syscall policy")
	originPolicyFile := flag.String("origin-policy", "", "Reviewed exact additional origins for each recording website")
	validateOriginPolicy := flag.Bool("validate-origin-policy", false, "Validate the selected origin policy and exit without storage or browsers")
	directory := flag.String("directory", "", "Private worker transport directory")
	capacity := flag.Int("capacity", 2, "Concurrent recording browser capacity from 1 through 16")
	idle := flag.Duration("idle-limit", 2*time.Minute, "Disconnected browser limit from 15 seconds through 5 minutes")
	local := flag.Bool("local", false, "Enable loopback-only local verification and private fixture targets")
	flag.Parse()
	if *validateOriginPolicy {
		if *originPolicyFile == "" || flag.NArg() != 0 {
			return errors.New("select one origin policy file to validate")
		}
		_, err := recordingworker.LoadOriginPolicy(*originPolicyFile)
		return err
	}
	if flag.NArg() != 0 || *directory == "" || *capacity < 1 || *capacity > 16 || *idle < 15*time.Second || *idle > 5*time.Minute {
		return errors.New("recording worker requires explicit private storage and no positional arguments")
	}
	host, _, err := net.SplitHostPort(*addr)
	if err != nil || *local && (net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback()) {
		return errors.New("local recording worker verification requires a numeric loopback address")
	}
	originPolicy, err := recordingworker.LoadOriginPolicy(*originPolicyFile)
	if err != nil {
		return err
	}
	connection, err := runtimeconfig.DatabaseConnection(*database, *databaseEnv, runtimeconfig.ExplicitFlag(flag.CommandLine, "database"), os.LookupEnv)
	if err != nil {
		return err
	}
	if err := runtimeconfig.RequirePostgres(connection, !*local); err != nil {
		return err
	}
	tlsConfig, err := workerhttp.ServerTLS(*certificate, *key, *clientCA, *clientName)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	db, err := storage.Open(ctx, connection)
	if err != nil {
		return runtimeconfig.DatabaseError(err, *databaseEnv)
	}
	defer db.Close()
	var permissionStore permissions.PermissionStore = permissionsqlite.New(db)
	if db.Dialect == "postgres" {
		pg, err := permissionpg.Open(ctx, connection)
		if err != nil {
			return runtimeconfig.DatabaseError(err, *databaseEnv)
		}
		defer pg.Close()
		permissionStore = pg
	}
	auth, err := authorization.New(ctx, permissionStore, &identity.Directory{Store: db})
	if err != nil {
		return errors.New("recording worker authorization could not initialize")
	}
	worker := &recordingworker.Worker{Store: db, Auth: auth, ID: *id, Instance: domain.NewID(), Capacity: *capacity, IdleLimit: *idle, AllowPrivate: *local, Origins: originPolicy, Runtime: &isolatedcontainer.Runtime{Image: *image, GatewayImage: *gateway, Seccomp: *seccomp, Directory: *directory}}
	if err := worker.Open(ctx); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_ = worker.Close(cleanup)
	}()
	server := &http.Server{Addr: *addr, Handler: workerhttp.Handler(worker), TLSConfig: tlsConfig, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 40 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return errors.New("private recording worker address is unavailable")
	}
	maintenanceCtx, cancelMaintenance := context.WithCancel(ctx)
	defer cancelMaintenance()
	maintenance := worker.StartMaintenance(maintenanceCtx)
	done := make(chan error, 1)
	go func() { done <- server.ServeTLS(listener, "", "") }()
	select {
	case <-ctx.Done():
	case <-maintenance:
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("private recording worker server stopped")
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cancelMaintenance()
	if err := server.Shutdown(shutdown); err != nil {
		return errors.New("recording worker shutdown remains pending")
	}
	<-maintenance
	return worker.Close(shutdown)
}
