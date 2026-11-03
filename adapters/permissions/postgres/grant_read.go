package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func grantReadResult(value permissions.Grant, err error) (permissions.Grant, error) {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return permissions.Grant{}, storage.ErrNotFound
	}
	return value, err
}

func (s *policyTransaction) ReadGrant(ctx context.Context, id int64) (permissions.Grant, error) {
	return grantReadResult(scanGrant(s.tx.QueryRowContext(ctx, `SELECT `+grantColumns+` FROM teststudio_permissions.permission_grants WHERE id=$1`, id)))
}

func (s *Store) ReadGrant(ctx context.Context, id int64) (permissions.Grant, error) {
	return grantReadResult(scanGrant(s.pool.QueryRow(ctx, `SELECT `+grantColumns+` FROM teststudio_permissions.permission_grants WHERE id=$1`, id)))
}
