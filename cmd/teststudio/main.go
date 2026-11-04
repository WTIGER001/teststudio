package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	permissions "github.com/wtiger001/go-permissions"
	artifactconfig "github.com/wtiger001/teststudio/adapters/artifacts/config"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/execution/localplaywright"
	httptrust "github.com/wtiger001/teststudio/adapters/httptrust/filesystem"
	"github.com/wtiger001/teststudio/adapters/notifications/webhook"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	recordinglocal "github.com/wtiger001/teststudio/adapters/recording/localplaywright"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/traceviewer"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	addr := flag.String("addr", "127.0.0.1:8040", "HTTP listen address")
	dsn := flag.String("database", ".data/teststudio.db", "SQLite path or PostgreSQL connection URL")
	databaseEnvironment := flag.String("database-env", "", "Environment variable containing the database connection")
	requirePostgres := flag.Bool("require-postgres", false, "Require PostgreSQL for a hosted or multi-instance installation")
	public := flag.String("public-url", "http://127.0.0.1:8040", "Public origin")
	ui := flag.String("ui", "web/dist/web/browser", "Built Angular application directory")
	artifactAddr := flag.String("artifact-addr", "127.0.0.1:8041", "Private artifact HTTP listen address")
	artifactOrigin := flag.String("artifact-origin", "http://127.0.0.1:8041", "Separate public artifact origin")
	traceDirectory := flag.String("trace-viewer-directory", "", "Trusted pinned trace viewer bundle, served only on the separate artifact origin")
	artifactDirectory := flag.String("artifact-directory", ".data/artifacts", "Private local artifact directory")
	artifactConfiguration := flag.String("artifact-store-config", "", "Operator JSON settings for filesystem, S3, or Azure Blob artifacts")
	location := flag.String("execution-location", "local", "Execution location identifier")
	playwrightImages := flag.String("playwright-images", "", "Comma-separated immutable local Playwright image references; enables the isolated local worker")
	browserGateway := flag.String("browser-gateway-image", "", "Immutable trusted gateway image; enables single-origin environment-backed local Playwright and Go API execution")
	goImages := flag.String("go-images", "", "Comma-separated immutable local Go suite image references; enables the isolated local Go worker")
	preparedSources := flag.Bool("prepared-sources", false, "Enable local prebuilt preparation and project-labelled immutable suite images")
	sourceBundles := flag.Bool("source-bundles", false, "Enable Go ZIP staging for the separate teststudio-prepare worker")
	githubSources := flag.Bool("github-sources", false, "Enable GitHub repository registration and durable sync queues")
	scheduler := flag.Bool("scheduler", true, "Process durable project schedules on this instance")
	notificationWorker := flag.Bool("notification-worker", true, "Process durable notification deliveries on this instance")
	secretBindings := flag.String("secret-bindings", "", "Project and purpose scoped environment or mounted-file secret binding configuration")
	httpTrustBundles := flag.String("http-trust-bundles", "", "Project and exact-origin scoped private HTTPS certificate authority bindings")
	localAccounts := flag.Bool("local-accounts", false, "Enable managed local password accounts alongside configured OIDC providers")
	local := flag.Bool("local", false, "Enable explicit loopback-only local administrator sign-in")
	recoverLocal := flag.Bool("recover-local-admin", false, "Reenable the loopback local administrator and invalidate its earlier sessions; requires --local")
	providers := flag.String("oidc-config", "", "Initial JSON provider configuration file")
	oidcSecretRefs := flag.String("oidc-secret-refs", "", "Comma-separated server credential references approved for OIDC provider configuration")
	bootstrapIssuer := flag.String("bootstrap-issuer", "", "OIDC issuer for the initial installation administrator")
	bootstrapSubject := flag.String("bootstrap-subject", "", "Exact OIDC subject for the initial installation administrator")
	recordingCLI := flag.String("recording-cli", "", "Trusted installed Playwright CLI path; enables native desktop recording with --local")
	recordingHelper := flag.String("recording-helper", "scripts/record-browser.mjs", "Trusted desktop recording helper script")
	recordingDirectory := flag.String("recording-directory", ".data/recordings", "Private local browser recording directory")
	flag.Parse()
	connection, err := runtimeconfig.DatabaseConnection(*dsn, *databaseEnvironment, runtimeconfig.ExplicitFlag(flag.CommandLine, "database"), os.LookupEnv)
	if err != nil {
		return err
	}
	if err := runtimeconfig.RequirePostgres(connection, *requirePostgres); err != nil {
		return err
	}
	if *recordingCLI != "" && !*local {
		return errors.New("native browser recording requires explicit loopback-only local mode")
	}
	if *recoverLocal && !*local {
		return errors.New("recover-local-admin requires explicit loopback-only local mode")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	parsed, err := url.Parse(*public)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return errors.New("public-url must be an HTTP or HTTPS origin")
	}
	if *localAccounts {
		if err := validateLocalAccountOrigin(parsed); err != nil {
			return err
		}
	}
	if err := notifications.ValidateOrigin(*public); err != nil {
		return err
	}
	secretProvider, err := localsecrets.Load(*secretBindings)
	if err != nil {
		return err
	}
	httpTrust, err := httptrust.Load(*httpTrustBundles)
	if err != nil {
		return err
	}
	artifactURL, err := url.Parse(*artifactOrigin)
	if err != nil || artifactURL.Host == "" || (artifactURL.Scheme != "http" && artifactURL.Scheme != "https") || artifactURL.User != nil || artifactURL.Path != "" || artifactURL.RawQuery != "" || artifactURL.Fragment != "" || (artifactURL.Scheme == parsed.Scheme && strings.EqualFold(artifactURL.Host, parsed.Host)) {
		return errors.New("artifact-origin must be a separate HTTP or HTTPS origin")
	}
	uiPath, err := filepath.Abs(*ui)
	if err != nil {
		return err
	}
	artifactSettings, err := artifactconfig.Load(*artifactConfiguration, *artifactDirectory)
	if err != nil {
		return err
	}
	if artifactSettings.Driver == "filesystem" {
		relative, err := filepath.Rel(uiPath, artifactSettings.Directory)
		if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
			return errors.New("artifact-directory must be outside the public UI directory")
		}
	}
	if *local || (*localAccounts && parsed.Scheme == "http") {
		host, _, err := net.SplitHostPort(*addr)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return errors.New("local authentication requires a loopback listen address")
		}
	}
	db, err := storage.Open(ctx, connection)
	if err != nil {
		return runtimeconfig.DatabaseError(err, *databaseEnvironment)
	}
	defer db.Close()
	var permissionStore permissions.PermissionStore = permissionsqlite.New(db)
	if db.Dialect == "postgres" {
		pg, err := permissionpg.Open(ctx, connection)
		if err != nil {
			return runtimeconfig.DatabaseError(err, *databaseEnvironment)
		}
		defer pg.Close()
		permissionStore = pg
	}
	directory := &identity.Directory{Store: db}
	auth, err := authorization.New(ctx, permissionStore, directory)
	if err != nil {
		return err
	}
	localUser := ""
	if *local {
		localUser = "local-administrator"
		if err := bootstrapLocalAdministrator(ctx, auth, *recoverLocal); err != nil {
			return err
		}
	}
	if *bootstrapIssuer != "" || *bootstrapSubject != "" {
		if err := bootstrapExternalAdministrator(ctx, auth, *bootstrapIssuer, *bootstrapSubject); err != nil {
			return err
		}
	}
	approvedOIDCSecrets := []string{}
	for _, value := range strings.Split(*oidcSecretRefs, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !authoidc.ValidSecretReference(value) {
			return errors.New("OIDC credential references require env:VARIABLE names")
		}
		if !slices.Contains(approvedOIDCSecrets, value) {
			approvedOIDCSecrets = append(approvedOIDCSecrets, value)
		}
	}
	if *providers != "" {
		data, err := os.ReadFile(*providers)
		if err != nil {
			return err
		}
		var list []authoidc.Provider
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		for _, p := range list {
			if err := authoidc.ValidateProvider(p); err != nil {
				return err
			}
			if p.SecretRef != "" && !slices.Contains(approvedOIDCSecrets, p.SecretRef) {
				approvedOIDCSecrets = append(approvedOIDCSecrets, p.SecretRef)
			}
			c, err := storage.Put("providers", p.ID, "", 0, p)
			if err != nil {
				return err
			}
			if err := db.Commit(ctx, c); err != nil && !errors.Is(err, storage.ErrConflict) {
				return err
			}
		}
	}
	sessionService := &sessions.Service{LocalAccounts: *localAccounts, Store: db, Directory: directory, Origin: *public, Secure: parsed.Scheme == "https"}
	oidc := &authoidc.Service{Store: db, Directory: directory, Sessions: sessionService, Origin: *public, SecretRefs: approvedOIDCSecrets}
	blobs, closeArtifacts, err := artifactconfig.Open(ctx, artifactSettings)
	if err != nil {
		return err
	}
	defer closeArtifacts()
	catalogService := &catalog.Service{Store: db, Auth: auth}
	evidenceService := &evidence.Service{Store: db, Blobs: blobs}
	runService := &runs.Service{Store: db, Auth: auth, Catalog: catalogService, Evidence: evidenceService, HTTP: &execution.HTTP{Trust: httpTrust}, Location: *location}
	sourceService := &sources.Service{Store: db, Auth: auth, Location: *location, RecoveryInspectors: map[string]preparation.ArtifactInspector{
		"go": &localcontainer.Executor{Runner: "go"}, "playwright": &localcontainer.Executor{Runner: "playwright"},
	}}
	if *sourceBundles || *githubSources {
		sourceService.BundleRunners = []string{"go", "playwright"}
	}
	if *githubSources {
		sourceService.Secrets = secretProvider
		sourceService.Providers = map[string]sourceprovider.Provider{"github": &githubsource.Client{Secrets: secretProvider}}
	}
	if *preparedSources {
		sourceService.Inspectors = map[string]preparation.ArtifactInspector{
			"go":         &localcontainer.Executor{Runner: "go", AllowProjectImages: true, RequireProject: true},
			"playwright": &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true},
		}
	}
	runService.RecoveryDriver = &localplaywright.Executor{}
	runService.RecoveryDrivers = []execution.ManagedExecutor{&localcontainer.Executor{Runner: "go"}, &localcontainer.Executor{Runner: "playwright", Environment: true}, &localcontainer.Executor{Runner: "go", Environment: true}}
	if strings.TrimSpace(*playwrightImages) != "" || *preparedSources || *sourceBundles || *githubSources {
		driver := &localplaywright.Executor{Images: strings.Split(*playwrightImages, ",")}
		if *preparedSources || *sourceBundles || *githubSources {
			driver.ApproveArtifact = sourceService.ApproveArtifact
		}
		runService.Playwright = driver
		if *browserGateway != "" {
			browser := *driver
			browser.Environment, browser.GatewayImage = true, *browserGateway
			browser.Secrets = secretProvider
			runService.Browser = &browser
		}
	}
	if strings.TrimSpace(*goImages) != "" || *preparedSources || *sourceBundles || *githubSources {
		driver := &localcontainer.Executor{Runner: "go", Images: strings.Split(*goImages, ",")}
		if *preparedSources || *sourceBundles || *githubSources {
			driver.ApproveArtifact = sourceService.ApproveArtifact
		}
		runService.Go = driver
		if *browserGateway != "" {
			apiDriver := *driver
			apiDriver.Environment, apiDriver.GatewayImage, apiDriver.Secrets = true, *browserGateway, secretProvider
			runService.GoAPI = &apiDriver
		}
	}
	scheduleService := &schedules.Service{Store: db, Auth: auth, Runs: runService, RepositorySources: sourceService}
	notificationService := &notifications.Service{Store: db, Auth: auth, Secrets: secretProvider, Sender: &webhook.Sender{}, Origin: *public}
	runService.Completion = notificationService
	recordingService := &recording.Service{Store: db, Auth: auth}
	if *recordingCLI != "" {
		node, err := exec.LookPath("node")
		if err != nil {
			return errors.New("native browser recording requires Node.js")
		}
		driver, err := recordinglocal.New(ctx, node, *recordingCLI, *recordingHelper, *recordingDirectory)
		if err != nil {
			return err
		}
		defer driver.Close()
		recordingService.Driver = driver
	}
	var traceBundle *traceviewer.Bundle
	if *traceDirectory != "" {
		traceBundle, err = traceviewer.Load(*traceDirectory)
		if err != nil {
			return err
		}
	}
	api := &httpapi.Server{TraceViewer: traceBundle, Recordings: recordingService, Sources: sourceService, Store: db, Auth: auth, Projects: &projects.Service{Store: db, Auth: auth, Secrets: secretProvider}, Catalog: catalogService, Runs: runService, Schedules: scheduleService, Notifications: notificationService, Evidence: evidenceService, ArtifactOrigin: *artifactOrigin, Sessions: sessionService, OIDC: oidc, LocalUser: localUser, UI: *ui}
	if *localAccounts {
		api.LocalAccounts, err = localauth.New(ctx, sessionService)
		if err != nil {
			return err
		}
	} else if err := localauth.Configure(ctx, directory, false); err != nil {
		return err
	}
	server := &http.Server{Addr: *addr, Handler: api.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second}
	artifactServer := &http.Server{Addr: *artifactAddr, Handler: api.ArtifactHandler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second}
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	artifactListener, err := net.Listen("tcp", *artifactAddr)
	if err != nil {
		return err
	}
	defer artifactListener.Close()
	workerContext, cancelWorkers := context.WithCancel(ctx)
	workersDone := runService.Start(workerContext)
	recordingMaintenanceDone := recordingService.StartMaintenance(workerContext)
	cliMaintenanceDone := (&cliauth.Service{Sessions: sessionService}).StartMaintenance(workerContext)
	maintenanceDone := evidenceService.StartMaintenance(workerContext, 5*time.Second)
	sourceMaintenanceDone := sourceService.StartMaintenance(workerContext, 5*time.Second)
	var preparationDone <-chan struct{}
	preparationDone = sourceService.StartPreparation(workerContext)
	var schedulesDone <-chan struct{}
	if *scheduler {
		schedulesDone = scheduleService.Start(workerContext)
	}
	var notificationsDone <-chan struct{}
	if *notificationWorker {
		notificationsDone = notificationService.Start(workerContext)
	}
	defer func() {
		cancelWorkers()
		<-workersDone
		<-cliMaintenanceDone
		<-recordingMaintenanceDone
		<-maintenanceDone
		<-sourceMaintenanceDone
		if preparationDone != nil {
			<-preparationDone
		}
		if schedulesDone != nil {
			<-schedulesDone
		}
		if notificationsDone != nil {
			<-notificationsDone
		}
	}()
	failed := make(chan error, 2)
	go func() { failed <- server.Serve(listener) }()
	go func() { failed <- artifactServer.Serve(artifactListener) }()
	defer server.Close()
	defer artifactServer.Close()
	fmt.Printf("TestStudio listens at %s\n", *public)
	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return errors.Join(server.Shutdown(shutdown), artifactServer.Shutdown(shutdown))
	}
	return nil
}
