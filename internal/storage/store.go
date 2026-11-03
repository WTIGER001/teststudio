package storage

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrConflict = errors.New("record revision conflict")
var ErrNotFound = errors.New("record not found")

type Record struct {
	Bucket   string          `json:"-"`
	ID       string          `json:"id"`
	Project  string          `json:"projectId,omitempty"`
	Revision int64           `json:"revision"`
	Data     json.RawMessage `json:"data"`
	Updated  time.Time       `json:"updatedAt"`
}

type Change struct {
	Record
	Delete bool
	Assert bool
}

type Page struct {
	Items  []Record `json:"items"`
	Cursor string   `json:"cursor,omitempty"`
}

type Store struct {
	DB      *sql.DB
	Dialect string
	tx      *sql.Tx
	failed  error
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("database connection or local file path is required")
	}
	driver, dialect := "sqlite", "sqlite"
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		driver, dialect = "pgx", "postgres"
	} else if dsn != ":memory:" {
		if strings.HasPrefix(dsn, "file:") {
			return nil, errors.New("use a local database path instead of a SQLite URI")
		}
		if err := os.MkdirAll(filepath.Dir(dsn), 0o700); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if dialect == "sqlite" {
		db.SetMaxOpenConns(1)
		for _, statement := range []string{"PRAGMA busy_timeout=5000", "PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON"} {
			if err := retrySQLiteBusy(ctx, func() error { _, err := db.ExecContext(ctx, statement); return err }); err != nil {
				db.Close()
				return nil, err
			}
		}
	}
	s := &Store{DB: db, Dialect: dialect}
	migrate := func() error { return s.migrate(ctx) }
	if dialect == "sqlite" {
		migrate = func() error { return retrySQLiteBusy(ctx, func() error { return s.migrate(ctx) }) }
	}
	if err := migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate(ctx context.Context) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if s.Dialect == "postgres" {
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(884190211)"); err != nil {
			return err
		}
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS records (bucket TEXT NOT NULL, id TEXT NOT NULL, project TEXT NOT NULL DEFAULT '', revision BIGINT NOT NULL, payload TEXT NOT NULL, updated_at TEXT NOT NULL, PRIMARY KEY (bucket, id))`,
		`CREATE INDEX IF NOT EXISTS records_project ON records (project, bucket, id)`,
		`CREATE TABLE IF NOT EXISTS counters (name TEXT PRIMARY KEY, value BIGINT NOT NULL)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate storage: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (1, $1) ON CONFLICT (version) DO NOTHING`, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Get(ctx context.Context, bucket, id string) (Record, error) {
	r := Record{Bucket: bucket, ID: id}
	var data, updated string
	err := s.executor().QueryRowContext(ctx, `SELECT project, revision, payload, updated_at FROM records WHERE bucket=$1 AND id=$2`, bucket, id).Scan(&r.Project, &r.Revision, &data, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.Data = json.RawMessage(data)
	r.Updated, err = time.Parse(time.RFC3339Nano, updated)
	return r, err
}

func (s *Store) List(ctx context.Context, bucket, project, cursor string, limit int) (Page, error) {
	return s.list(ctx, bucket, project, cursor, limit, false)
}

func (s *Store) ListNewest(ctx context.Context, bucket, project, cursor string, limit int) (Page, error) {
	return s.list(ctx, bucket, project, cursor, limit, true)
}

func (s *Store) list(ctx context.Context, bucket, project, cursor string, limit int, newest bool) (Page, error) {
	order, comparison := "asc", ">"
	if newest {
		order, comparison = "desc", "<"
	}
	if limit < 1 || limit > 1000 {
		return Page{}, errors.New("limit must be between 1 and 1000")
	}
	after := ""
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return Page{}, errors.New("invalid cursor")
		}
		values, err := url.ParseQuery(string(decoded))
		if err != nil || values.Get("bucket") != bucket || values.Get("project") != project || values.Get("after") == "" || (values.Get("order") != "" && values.Get("order") != order) {
			return Page{}, errors.New("cursor does not match the query")
		}
		after = values.Get("after")
	}
	rows, err := s.executor().QueryContext(ctx, fmt.Sprintf(`SELECT id, project, revision, payload, updated_at FROM records WHERE bucket=$1 AND ($2='' OR project=$2) AND ($3='' OR id%s$3) ORDER BY id %s LIMIT $4`, comparison, order), bucket, project, after, limit+1)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	page := Page{Items: []Record{}}
	for rows.Next() {
		r := Record{Bucket: bucket}
		var data, updated string
		if err := rows.Scan(&r.ID, &r.Project, &r.Revision, &data, &updated); err != nil {
			return Page{}, err
		}
		r.Data = json.RawMessage(data)
		r.Updated, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, r)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	if len(page.Items) > limit {
		values := url.Values{"bucket": {bucket}, "project": {project}, "after": {page.Items[limit-1].ID}, "order": {order}}
		page.Cursor = base64.RawURLEncoding.EncodeToString([]byte(values.Encode()))
		page.Items = page.Items[:limit]
	}
	return page, nil
}

func (s *Store) All(ctx context.Context, bucket, project string) ([]Record, error) {
	result := []Record{}
	cursor := ""
	for {
		page, err := s.List(ctx, bucket, project, cursor, 1000)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Items...)
		if page.Cursor == "" {
			return result, nil
		}
		cursor = page.Cursor
	}
}

func Put(bucket, id, project string, revision int64, value any) (Change, error) {
	data, err := json.Marshal(value)
	return Change{Record: Record{Bucket: bucket, ID: id, Project: project, Revision: revision, Data: data}}, err
}

func Decode[T any](r Record) (T, error) {
	var value T
	err := json.Unmarshal(r.Data, &value)
	return value, err
}

func (s *Store) Commit(ctx context.Context, changes ...Change) error {
	if len(changes) == 0 || len(changes) > 100 {
		return errors.New("transaction must contain between 1 and 100 changes")
	}
	if s.tx != nil {
		if s.failed != nil {
			return s.failed
		}
		err := s.apply(ctx, changes)
		if err != nil {
			s.failed = err
		}
		return err
	}
	return s.Transaction(ctx, func(bound *Store, _ *sql.Tx) error {
		return bound.Commit(ctx, changes...)
	})
}

func (s *Store) apply(ctx context.Context, changes []Change) error {
	var err error
	seen := map[string]bool{}
	for _, c := range changes {
		key := c.Bucket + "\x00" + c.ID
		if c.Bucket == "" || c.ID == "" || c.Revision < 0 || seen[key] || (!c.Delete && !c.Assert && !json.Valid(c.Data)) || (c.Assert && (c.Delete || c.Revision < 1)) {
			return errors.New("invalid or duplicate transaction record")
		}
		seen[key] = true
		var result sql.Result
		if c.Assert {
			result, err = s.executor().ExecContext(ctx, `UPDATE records SET revision=revision WHERE bucket=$1 AND id=$2 AND revision=$3`, c.Bucket, c.ID, c.Revision)
		} else if c.Delete {
			result, err = s.executor().ExecContext(ctx, `DELETE FROM records WHERE bucket=$1 AND id=$2 AND revision=$3`, c.Bucket, c.ID, c.Revision)
		} else if c.Revision == 0 {
			result, err = s.executor().ExecContext(ctx, `INSERT INTO records (bucket,id,project,revision,payload,updated_at) VALUES ($1,$2,$3,1,$4,$5) ON CONFLICT (bucket,id) DO NOTHING`, c.Bucket, c.ID, c.Project, string(c.Data), time.Now().UTC().Format(time.RFC3339Nano))
		} else {
			result, err = s.executor().ExecContext(ctx, `UPDATE records SET project=$1,revision=revision+1,payload=$2,updated_at=$3 WHERE bucket=$4 AND id=$5 AND revision=$6`, c.Project, string(c.Data), time.Now().UTC().Format(time.RFC3339Nano), c.Bucket, c.ID, c.Revision)
		}
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrConflict
		}
	}
	return nil
}

func (s *Store) Next(ctx context.Context, name string) (int64, error) {
	var result int64
	err := s.executor().QueryRowContext(ctx, `INSERT INTO counters (name,value) VALUES ($1,1) ON CONFLICT (name) DO UPDATE SET value=counters.value+1 RETURNING value`, name).Scan(&result)
	return result, err
}
