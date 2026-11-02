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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	permissions "github.com/wtiger001/go-permissions"
	permissionpg "github.com/wtiger001/go-permissions/postgres"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	addr := flag.String("addr", "127.0.0.1:8040", "HTTP listen address")
	dsn := flag.String("database", ".data/teststudio.db", "SQLite path or PostgreSQL connection URL")
	public := flag.String("public-url", "http://127.0.0.1:8040", "Public origin")
	ui := flag.String("ui", "web/dist/web/browser", "Built Angular application directory")
	artifactAddr := flag.String("artifact-addr", "127.0.0.1:8041", "Private artifact HTTP listen address")
	artifactOrigin := flag.String("artifact-origin", "http://127.0.0.1:8041", "Separate public artifact origin")
	artifactDirectory := flag.String("artifact-directory", ".data/artifacts", "Private local artifact directory")
	location := flag.String("execution-location", "local", "Execution location identifier")
	local := flag.Bool("local", false, "Enable explicit loopback-only local administrator sign-in")
	providers := flag.String("oidc-config", "", "Initial JSON provider configuration file")
	bootstrapIssuer := flag.String("bootstrap-issuer", "", "OIDC issuer for the initial installation administrator")
	bootstrapSubject := flag.String("bootstrap-subject", "", "Exact OIDC subject for the initial installation administrator")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	parsed, err := url.Parse(*public)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return errors.New("public-url must be an HTTP or HTTPS origin")
	}
	artifactURL, err := url.Parse(*artifactOrigin)
	if err != nil || artifactURL.Host == "" || (artifactURL.Scheme != "http" && artifactURL.Scheme != "https") || artifactURL.User != nil || artifactURL.Path != "" || artifactURL.RawQuery != "" || artifactURL.Fragment != "" || (artifactURL.Scheme == parsed.Scheme && strings.EqualFold(artifactURL.Host, parsed.Host)) {
		return errors.New("artifact-origin must be a separate HTTP or HTTPS origin")
	}
	uiPath, err := filepath.Abs(*ui)
	if err != nil {
		return err
	}
	artifactPath, err := filepath.Abs(*artifactDirectory)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(uiPath, artifactPath)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("artifact-directory must be outside the public UI directory")
	}
	if *local {
		host, _, err := net.SplitHostPort(*addr)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return errors.New("local authentication requires a loopback listen address")
		}
	}
	db, err := storage.Open(ctx, *dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	var permissionStore permissions.PermissionStore = permissionsqlite.New(db)
	if db.Dialect == "postgres" {
		if _, err := db.DB.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS teststudio_permissions"); err != nil {
			return err
		}
		cfg, err := pgxpool.ParseConfig(*dsn)
		if err != nil {
			return err
		}
		cfg.ConnConfig.RuntimeParams["search_path"] = "teststudio_permissions"
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return err
		}
		defer pool.Close()
		pg := permissionpg.NewStore(pool)
		if err := pg.EnsureSchema(ctx); err != nil {
			return err
		}
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
		if _, err := directory.User(ctx, localUser); errors.Is(err, storage.ErrNotFound) {
			if err := directory.SaveUser(ctx, identity.User{ID: localUser, Name: "Local administrator"}, 0); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := auth.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: localUser}, "builtin.teststudio.installation-admin", nil); err != nil {
			return err
		}
	}
	if *bootstrapIssuer != "" || *bootstrapSubject != "" {
		u, err := directory.ResolveExternal(ctx, *bootstrapIssuer, *bootstrapSubject, "Installation administrator", "", false, true)
		if err != nil {
			return err
		}
		if err := auth.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: u.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
			return err
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
			c, err := storage.Put("providers", p.ID, "", 0, p)
			if err != nil {
				return err
			}
			if err := db.Commit(ctx, c); err != nil && !errors.Is(err, storage.ErrConflict) {
				return err
			}
		}
	}
	sessionService := &sessions.Service{Store: db, Directory: directory, Origin: *public, Secure: parsed.Scheme == "https"}
	oidc := &authoidc.Service{Store: db, Directory: directory, Sessions: sessionService, Origin: *public}
	files, err := filesystem.New(*artifactDirectory)
	if err != nil {
		return err
	}
	defer files.Close()
	catalogService := &catalog.Service{Store: db, Auth: auth}
	evidenceService := &evidence.Service{Store: db, Blobs: files}
	runService := &runs.Service{Store: db, Auth: auth, Catalog: catalogService, Evidence: evidenceService, HTTP: &execution.HTTP{}, Location: *location}
	api := &httpapi.Server{Store: db, Auth: auth, Projects: &projects.Service{Store: db, Auth: auth}, Catalog: catalogService, Runs: runService, Evidence: evidenceService, ArtifactOrigin: *artifactOrigin, Sessions: sessionService, OIDC: oidc, LocalUser: localUser, UI: *ui}
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
	defer func() { cancelWorkers(); <-workersDone }()
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
