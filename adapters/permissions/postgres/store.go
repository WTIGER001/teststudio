package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	upstream "github.com/wtiger001/go-permissions/postgres"
)

type Store struct {
	*upstream.Store
	pool *pgxpool.Pool
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "teststudio_permissions"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	s := &Store{Store: upstream.NewStore(pool), pool: pool}
	if err := s.migrate(ctx, cfg.ConnConfig); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) migrate(ctx context.Context, cfg *pgx.ConnConfig) error {
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer conn.Close(context.WithoutCancel(ctx))
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(884190212)"); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, err := conn.Exec(cleanup, "SELECT pg_advisory_unlock(884190212)"); err != nil {
			_ = conn.Close(cleanup)
		}
	}()
	if _, err := conn.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS teststudio_permissions"); err != nil {
		return err
	}
	if err := s.EnsureSchema(ctx); err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "ALTER TABLE teststudio_permissions.roles DROP CONSTRAINT IF EXISTS roles_code_key")
	return err
}
