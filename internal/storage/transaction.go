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

func (s *Store) CommitAll(ctx context.Context, changes ...Change) error {
	if len(changes) == 0 || len(changes) > 10000 {
		return errors.New("transaction must contain between 1 and 10000 changes")
	}
	seen := map[string]bool{}
	for _, change := range changes {
		key := change.Bucket + "\x00" + change.ID
		if seen[key] {
			return errors.New("duplicate transaction record")
		}
		seen[key] = true
	}
	if s.tx == nil {
		return s.Transaction(ctx, func(bound *Store, _ *sql.Tx) error { return bound.CommitAll(ctx, changes...) })
	}
	for start := 0; start < len(changes); start += 100 {
		end := min(start+100, len(changes))
		if err := s.Commit(ctx, changes[start:end]...); err != nil {
			return err
		}
	}
	return nil
}
