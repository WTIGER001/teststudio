package main

import (
	"context"

	permissions "github.com/wtiger001/go-permissions"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
	"github.com/wtiger001/teststudio/internal/storage"
)

func openServiceStorage(ctx context.Context, connection, environment string) (*storage.Store, *authorization.Service, func(), error) {
	db, err := storage.Open(ctx, connection)
	if err != nil {
		return nil, nil, nil, runtimeconfig.DatabaseError(err, environment)
	}
	closeStorage := func() { db.Close() }
	var permissionStore permissions.PermissionStore = permissionsqlite.New(db)
	if db.Dialect == "postgres" {
		pg, err := permissionpg.Open(ctx, connection)
		if err != nil {
			closeStorage()
			return nil, nil, nil, runtimeconfig.DatabaseError(err, environment)
		}
		closeStorage = func() { pg.Close(); db.Close() }
		permissionStore = pg
	}
	auth, err := authorization.New(ctx, permissionStore, &identity.Directory{Store: db})
	if err != nil {
		closeStorage()
		return nil, nil, nil, err
	}
	return db, auth, closeStorage, nil
}
