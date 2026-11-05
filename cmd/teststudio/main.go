package main

import (
	"context"
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

	artifactconfig "github.com/wtiger001/teststudio/adapters/artifacts/config"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	ecsworker "github.com/wtiger001/teststudio/adapters/execution/ecs"
	"github.com/wtiger001/teststudio/adapters/execution/kubernetes"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/execution/localplaywright"
	httptrust "github.com/wtiger001/teststudio/adapters/httptrust/filesystem"
	"github.com/wtiger001/teststudio/adapters/notifications/webhook"
	recordinglocal "github.com/wtiger001/teststudio/adapters/recording/localplaywright"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/httpapi"
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
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	workerAddr := flag.String("worker-addr", "", "Private job transport listen address; requires private HTTPS ingress or direct TLS in the worker role")
	workerOrigin := flag.String("worker-origin", "", "Exact HTTPS authority for direct TLS in the worker-only role")
	workerCertificate := flag.String("worker-tls-certificate", "", "Mounted PEM certificate chain for direct worker backend TLS")
	workerKey := flag.String("worker-tls-key", "", "Mounted PEM private key for direct worker backend TLS")
	ecsConfiguration := flag.String("ecs-workers-config", "", "Private ECS worker and network JSON settings")
	ecsEnvironment := flag.String("ecs-workers-env", "", "Environment variable containing private ECS worker settings")
	kubernetesConfiguration := flag.String("kubernetes-workers-config", "", "Operator JSON settings for isolated Kubernetes workers")
	kubernetesEnvironment := flag.String("kubernetes-workers-env", "", "Environment variable containing isolated Kubernetes worker settings")
	serviceRole := flag.String("serve-role", "all", "Service role: all, api, artifacts, worker, or controller")
	addr := flag.String("addr", "127.0.0.1:8040", "HTTP listen address")
	dsn := flag.String("database", ".data/teststudio.db", "SQLite path or PostgreSQL connection URL")
	databaseEnvironment := flag.String("database-env", "", "Environment variable containing the database connection")
	databaseHost := flag.String("database-host", "", "Bind PostgreSQL to this DNS hostname with verified TLS and no routing overrides")
	requirePostgres := flag.Bool("require-postgres", false, "Require PostgreSQL for a hosted or multi-instance installation")
	public := flag.String("public-url", "http://127.0.0.1:8040", "Public origin")
	ui := flag.String("ui", "web/dist/web/browser", "Built Angular application directory")
	artifactAddr := flag.String("artifact-addr", "127.0.0.1:8041", "Private artifact HTTP listen address")
	artifactOrigin := flag.String("artifact-origin", "http://127.0.0.1:8041", "Separate public artifact origin")
	traceDirectory := flag.String("trace-viewer-directory", "", "Trusted pinned trace viewer bundle, served only on the separate artifact origin")
	artifactDirectory := flag.String("artifact-directory", ".data/artifacts", "Private local artifact directory")
	artifactConfiguration := flag.String("artifact-store-config", "", "Operator JSON settings for filesystem, S3, or Azure Blob artifacts")
	artifactEnvironment := flag.String("artifact-store-env", "", "Environment variable containing operator artifact JSON settings")
	location := flag.String("execution-location", "local", "Execution location identifier")
	playwrightImages := flag.String("playwright-images", "", "Comma-separated immutable local Playwright image references; enables the isolated local worker")
	browserGateway := flag.String("browser-gateway-image", "", "Immutable trusted gateway image; enables single-origin environment-backed local Playwright and Go API execution")
	goImages := flag.String("go-images", "", "Comma-separated immutable local Go suite image references; enables the isolated local Go worker")
	preparedSources := flag.Bool("prepared-sources", false, "Enable source discovery and project-owned immutable suite images for the configured execution provider")
	sourceBundles := flag.Bool("source-bundles", false, "Enable offline ZIP staging for the separate teststudio-prepare worker")
	sourceBundleRunners := flag.String("source-bundle-runners", "go,playwright", "Available source builders: go, playwright, or both separated by a comma")
	githubSources := flag.Bool("github-sources", false, "Enable GitHub repository registration and durable sync queues")
	scheduler := flag.Bool("scheduler", true, "Process durable project schedules on this instance")
	executionWorker := flag.Bool("execution-worker", true, "Process queued test runs on this API instance; disable when using a separate controller")
	notificationWorker := flag.Bool("notification-worker", true, "Process durable notification deliveries on this instance")
	secretBindings := flag.String("secret-bindings", "", "Project and purpose scoped environment or mounted-file secret binding configuration")
	httpTrustBundles := flag.String("http-trust-bundles", "", "Project and exact-origin scoped private HTTPS certificate authority bindings")
	localAccounts := flag.Bool("local-accounts", false, "Enable managed local password accounts alongside configured OIDC providers")
	local := flag.Bool("local", false, "Enable explicit loopback-only local administrator sign-in")
	recoverLocal := flag.Bool("recover-local-admin", false, "Reenable the loopback local administrator and invalidate its earlier sessions; requires --local")
	providers := flag.String("oidc-config", "", "Initial JSON provider configuration file")
	providersEnvironment := flag.String("oidc-config-env", "", "Environment variable containing initial OIDC provider JSON settings")
	oidcCAFile := flag.String("oidc-ca-file", "", "Mounted PEM certificate authorities trusted only for OIDC provider connections")
	oidcSecretRefs := flag.String("oidc-secret-refs", "", "Comma-separated server credential references approved for OIDC provider configuration")
	bootstrapIssuer := flag.String("bootstrap-issuer", "", "OIDC issuer for the initial installation administrator")
	bootstrapSubject := flag.String("bootstrap-subject", "", "Exact OIDC subject for the initial installation administrator")
	recordingCLI := flag.String("recording-cli", "", "Trusted installed Playwright CLI path; enables native desktop recording with --local")
	recordingHelper := flag.String("recording-helper", "scripts/record-browser.mjs", "Trusted desktop recording helper script")
	recordingDirectory := flag.String("recording-directory", ".data/recordings", "Private local browser recording directory")
	flag.Parse()
	role, err := runtimeconfig.ParseServiceRole(*serviceRole)
	if err != nil {
		return err
	}
	if !role.API && runtimeconfig.ExplicitFlag(flag.CommandLine, "execution-worker") {
		return errors.New("execution-worker applies only to the API or all service role")
	}
	if !role.Worker && (*workerOrigin != "" || *workerCertificate != "" || *workerKey != "") {
		return errors.New("direct worker TLS settings require the worker service role")
	}
	if !role.API && !role.Worker && !role.Controller && (*workerAddr != "" || *local || *localAccounts || *recoverLocal || *bootstrapIssuer != "" || *bootstrapSubject != "" || *providers != "" || *providersEnvironment != "" || *oidcSecretRefs != "" || *oidcCAFile != "" || *secretBindings != "" || *httpTrustBundles != "" || *recordingCLI != "" || *preparedSources || *sourceBundles || *githubSources || *goImages != "" || *playwrightImages != "" || *browserGateway != "") {
		return errors.New("the artifact service role cannot configure identity bootstrap, sign-in, source preparation, recording, execution, or private worker transport")
	}
	if !role.API && !role.Worker && !role.Controller && (*kubernetesConfiguration != "" || *kubernetesEnvironment != "" || *ecsConfiguration != "" || *ecsEnvironment != "") {
		return errors.New("the artifact service role cannot configure hosted execution")
	}
	if role.Controller {
		if err := validateControllerFlags(flag.CommandLine, *addr, *location, *ecsConfiguration != "" || *ecsEnvironment != "", *kubernetesConfiguration != "" || *kubernetesEnvironment != ""); err != nil {
			return err
		}
	}
	if role.Worker {
		if err := validateWorkerBackendFlags(flag.CommandLine, *workerAddr, *location); err != nil {
			return err
		}
	}
	connection, err := runtimeconfig.DatabaseConnection(*dsn, *databaseEnvironment, runtimeconfig.ExplicitFlag(flag.CommandLine, "database"), os.LookupEnv)
	if err != nil {
		return err
	}
	if err := runtimeconfig.RequirePostgres(connection, *requirePostgres); err != nil {
		return err
	}
	if err := runtimeconfig.ValidateDatabaseHost(connection, *databaseHost, os.Environ()); err != nil {
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
	if role.Worker {
		return runWorkerBackend(ctx, *workerAddr, *location, connection, *databaseEnvironment, *artifactConfiguration, *artifactEnvironment, *artifactDirectory, workerTLSOptions{Origin: *workerOrigin, Certificate: *workerCertificate, Key: *workerKey})
	}
	if role.Controller {
		return runController(ctx, controllerOptions{Address: *addr, Location: *location, Connection: connection, DatabaseEnvironment: *databaseEnvironment, ArtifactConfiguration: *artifactConfiguration, ArtifactEnvironment: *artifactEnvironment, ArtifactDirectory: *artifactDirectory, PublicOrigin: *public, TrustBundles: *httpTrustBundles, ECSConfiguration: *ecsConfiguration, ECSEnvironment: *ecsEnvironment, KubernetesConfiguration: *kubernetesConfiguration, KubernetesEnvironment: *kubernetesEnvironment})
	}
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
	artifactSettings, err := artifactconfig.LoadEnvironment(*artifactConfiguration, *artifactEnvironment, *artifactDirectory, os.LookupEnv)
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
	oidcClient, err := authoidc.TLSClient(*oidcCAFile)
	if err != nil {
		return err
	}
	initialProviders, err := authoidc.InitialProviders(*providers, *providersEnvironment, os.LookupEnv)
	if err != nil {
		return err
	}
	bundleRunners, err := runtimeconfig.SourceRunners(*sourceBundleRunners)
	if err != nil {
		return err
	}
	if (*kubernetesConfiguration != "" || *kubernetesEnvironment != "") && (*ecsConfiguration != "" || *ecsEnvironment != "") {
		return errors.New("choose one hosted execution provider")
	}
	ecsSettings, err := ecsworker.LoadSettings(*ecsConfiguration, *ecsEnvironment, os.LookupEnv)
	if err != nil {
		return err
	}
	if ecsSettings != nil {
		if *location != "ecs" || *workerAddr == "" || *local || *goImages != "" || *playwrightImages != "" || *browserGateway != "" {
			return errors.New("ECS workers require their execution location and private listener without local coded execution or inspection")
		}
		if _, port, err := net.SplitHostPort(*workerAddr); err != nil || port != "8042" {
			return errors.New("ECS workers require the private listener on port 8042")
		}
		if sameHTTPOrigin(ecsSettings.Worker.ExchangeOrigin, *public) || sameHTTPOrigin(ecsSettings.Worker.ExchangeOrigin, *artifactOrigin) {
			return errors.New("the private worker origin must differ from UI and artifact origins")
		}
		if err := runtimeconfig.RequirePostgres(connection, true); err != nil {
			return err
		}
	}
	kubernetesSettings, err := kubernetes.LoadSettings(*kubernetesConfiguration, *kubernetesEnvironment, os.LookupEnv)
	if err != nil {
		return err
	}
	if kubernetesSettings != nil {
		if *location != "kubernetes" || *workerAddr == "" || *local || *goImages != "" || *playwrightImages != "" || *browserGateway != "" {
			return errors.New("Kubernetes workers require their execution location and private listener without local coded execution or inspection")
		}
		if sameHTTPOrigin(kubernetesSettings.Worker.ExchangeOrigin, *public) || sameHTTPOrigin(kubernetesSettings.Worker.ExchangeOrigin, *artifactOrigin) {
			return errors.New("the private worker origin must differ from UI and artifact origins")
		}
		if err := runtimeconfig.RequirePostgres(connection, true); err != nil {
			return err
		}
	}
	db, auth, closeStorage, err := openServiceStorage(ctx, connection, *databaseEnvironment)
	if err != nil {
		return err
	}
	defer closeStorage()
	directory := auth.Directory
	blobs, closeArtifacts, err := artifactconfig.Open(ctx, artifactSettings)
	if err != nil {
		return err
	}
	defer closeArtifacts()
	evidenceService := &evidence.Service{Store: db, Blobs: blobs}
	if !role.API {
		return serveArtifacts(ctx, *artifactAddr, *public, *artifactOrigin, *traceDirectory, db, auth, evidenceService)
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
	if len(initialProviders) > 0 {
		for _, p := range initialProviders {
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
	oidc := &authoidc.Service{Store: db, Directory: directory, Sessions: sessionService, Origin: *public, SecretRefs: approvedOIDCSecrets, Client: oidcClient}
	catalogService := &catalog.Service{Store: db, Auth: auth}
	runService := &runs.Service{Store: db, Auth: auth, Catalog: catalogService, Evidence: evidenceService, HTTP: &execution.HTTP{Trust: httpTrust}, Location: *location}
	workerExchange := &workerexchange.Service{Store: db, Blobs: blobs, Authorize: runService.AuthorizeWorkerAttempt}
	sourceService := &sources.Service{Store: db, Auth: auth, Location: *location, RecoveryInspectors: map[string]preparation.ArtifactInspector{
		"go": &localcontainer.Executor{Runner: "go"}, "playwright": &localcontainer.Executor{Runner: "playwright"},
	}}
	if *sourceBundles || *githubSources {
		sourceService.BundleRunners = bundleRunners
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
	if ecsSettings != nil {
		runtime, err := ecsSettings.Open(ctx, db, sourceService.AuthorizeDiscoveryCleanup)
		if err != nil {
			return err
		}
		defer runtime.Close()
		configureHostedWorkers(runService, sourceService, workerExchange, runtime.Provider, runtime.Provider, "ecs", ecsSettings.Worker.ExchangeOrigin, *preparedSources)
	}
	if kubernetesSettings != nil {
		provider, err := kubernetesSettings.Open(db, os.LookupEnv, sourceService.AuthorizeDiscoveryCleanup)
		if err != nil {
			return err
		}
		defer provider.Close()
		configureHostedWorkers(runService, sourceService, workerExchange, provider, provider, "kubernetes", kubernetesSettings.Worker.ExchangeOrigin, *preparedSources)
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
	servers := []*http.Server{server}
	if role.Artifacts {
		servers = append(servers, artifactServer)
	}
	if *workerAddr != "" {
		servers = append(servers, &http.Server{Addr: *workerAddr, Handler: workerhttp.Handler(workerExchange), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 16 << 10})
	}
	background := backgroundServices{DisableExecution: !*executionWorker, Runs: runService, Recordings: recordingService, CLI: &cliauth.Service{Sessions: sessionService}, Evidence: evidenceService, Sources: sourceService}
	if *scheduler {
		background.Schedules = scheduleService
	}
	if *notificationWorker {
		background.Notifications = notificationService
	}
	return serveServers(ctx, servers, func() func() {
		fmt.Printf("TestStudio listens at %s\n", *public)
		return background.start(ctx)
	})
}
