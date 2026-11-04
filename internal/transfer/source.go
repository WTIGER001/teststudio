package transfer

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/wtiger001/teststudio/internal/storage"
)

func OpenSource(ctx context.Context, connection string) (*storage.Store, error) {
	if connection == "" {
		return nil, errors.New("a source database is required")
	}
	driver, dialect := "sqlite", "sqlite"
	if strings.HasPrefix(connection, "postgres://") || strings.HasPrefix(connection, "postgresql://") {
		driver, dialect = "pgx", "postgres"
	} else {
		absolute, err := filepath.Abs(connection)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absolute)
		if err != nil || !info.Mode().IsRegular() {
			return nil, errors.New("source must be an existing SQLite database file")
		}
		u := url.URL{Scheme: "file", Path: absolute}
		query := url.Values{"mode": {"ro"}, "_pragma": {"busy_timeout(5000)"}}
		u.RawQuery = query.Encode()
		connection = u.String()
	}
	db, err := sql.Open(driver, connection)
	if err != nil {
		return nil, errors.New("source database cannot be opened")
	}
	if dialect == "sqlite" {
		db.SetMaxOpenConns(1)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.New("source database cannot be read")
	}
	return &storage.Store{DB: db, Dialect: dialect}, nil
}
