package storage

import (
	"context"
	"database/sql"
	"errors"
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) executor() executor {
	if s.tx != nil {
		return s.tx
	}
	return s.DB
}

func (s *Store) Transaction(ctx context.Context, fn func(*Store, *sql.Tx) error) error {
	if s.tx != nil {
		return errors.New("nested storage transactions are not supported")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	bound := &Store{DB: s.DB, Dialect: s.Dialect, tx: tx}
	if err := fn(bound, tx); err != nil {
		return err
	}
	if bound.failed != nil {
		return bound.failed
	}
	return tx.Commit()
}
