package testkit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	permissions "github.com/wtiger001/go-permissions"
	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	permissionsqlite "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/storage"
)

func localDatabase() (*url.URL, error) {
	u, err := url.Parse(os.Getenv("TESTSTUDIO_TEST_DATABASE"))
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(u.Hostname())
	if (u.Scheme != "postgres" && u.Scheme != "postgresql") || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
		return nil, errors.New("database contract checks require a local PostgreSQL URL")
	}
	return u, nil
}

func Open(t *testing.T, local string) (*storage.Store, error) {
	t.Helper()
	if os.Getenv("TESTSTUDIO_TEST_DATABASE") == "" {
		return storage.Open(t.Context(), local)
	}
	u, err := localDatabase()
	if err != nil {
		return nil, err
	}
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return nil, err
	}
	name := "teststudio_" + hex.EncodeToString(suffix[:])
	admin, err := pgx.Connect(t.Context(), u.String())
	if err != nil {
		return nil, err
	}
	_, err = admin.Exec(t.Context(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	admin.Close(t.Context())
	if err != nil {
		return nil, err
	}
	adminURL := u.String()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, adminURL)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	u.Path = "/" + name
	return storage.Open(t.Context(), u.String())
}

func PermissionStore(t *testing.T, db *storage.Store) permissions.PermissionStore {
	t.Helper()
	if db.Dialect != "postgres" {
		return permissionsqlite.New(db)
	}
	u, err := localDatabase()
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.DB.QueryRowContext(t.Context(), "SELECT current_database()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	store, err := permissionpg.Open(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store
}

func ConnectionString(t *testing.T, db *storage.Store, local string) string {
	t.Helper()
	if db.Dialect != "postgres" {
		return local
	}
	u, err := localDatabase()
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.DB.QueryRowContext(t.Context(), "SELECT current_database()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}
