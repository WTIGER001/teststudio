package sqlite

import (
	"context"
	"database/sql"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Store) WithAssignmentTransaction(ctx context.Context, metadata *storage.Store, fn func(*storage.Store, permissions.PermissionStore) error) error {
	if metadata.DB != s.db.DB || metadata.Dialect != "sqlite" {
		return errors.New("assignment transaction requires the same SQLite database")
	}
	return metadata.Transaction(ctx, func(bound *storage.Store, _ *sql.Tx) error {
		return fn(bound, New(bound))
	})
}
