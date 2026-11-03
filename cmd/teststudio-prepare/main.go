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
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/preparation"
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
	location := flag.String("execution-location", "local", "Preparation queue location")
	runtime := flag.String("go-runtime", "", "Trusted immutable local Go source runtime image ID; omit to perform recovery only")
	directory := flag.String("build-directory", ".data/source-builds", "Private local source context directory shared by this location's workers")
	flag.Parse()
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
	store, err := storage.Open(ctx, *database)
	if err != nil {
		return err
	}
	defer store.Close()
	var permissionStore permissions.PermissionStore = permissionsqlite.New(store)
	if store.Dialect == "postgres" {
		pg, err := permissionpg.Open(ctx, *database)
		if err != nil {
			return err
		}
		defer pg.Close()
		permissionStore = pg
	}
	auth, err := authorization.New(ctx, permissionStore, &identity.Directory{Store: store})
	if err != nil {
		return err
	}
	inspector := &localcontainer.Executor{Runner: "go", AllowProjectImages: true, RequireProject: true}
	builder := &localimage.Builder{RuntimeImage: *runtime, TempDirectory: root}
	service := &sources.Service{Store: store, Auth: auth, Location: *location, Inspectors: map[string]preparation.ArtifactInspector{"go": inspector}, RecoveryInspectors: map[string]preparation.ArtifactInspector{"go": inspector}, RecoveryBuilders: map[string]preparation.ArtifactBuilder{"go": builder}}
	if *runtime != "" {
		service.Builders = map[string]preparation.ArtifactBuilder{"go": builder}
	}
	done := service.StartBuildPreparation(ctx)
	<-done
	return nil
}
