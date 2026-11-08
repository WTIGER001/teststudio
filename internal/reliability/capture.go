package reliability

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

const stamp = "2006-01-02T15:04:05.000000000Z"

func date(t time.Time) string { return t.UTC().Format(stamp) }

func lock(ctx context.Context, tx *sql.Tx, dialect, series string) error {
	if dialect != "postgres" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "reliability/"+series)
	return err
}

func Capture(ctx context.Context, tx *sql.Tx, dialect string, observations []Observation) error {
	observations = append([]Observation{}, observations...)
	ids := map[Scope]string{}
	for _, o := range observations {
		if _, ok := ids[o.Scope]; !ok {
			ids[o.Scope] = o.Scope.ID()
		}
	}
	sort.Slice(observations, func(i, j int) bool { return ids[observations[i].Scope] < ids[observations[j].Scope] })
	type capturedScope struct {
		scope       Scope
		first, last time.Time
		sources     map[string]bool
	}
	captured := map[string]*capturedScope{}
	locked := ""
	for _, o := range observations {
		if o.Run == "" || o.Project == "" || o.Test == "" || o.At.IsZero() || o.Created.IsZero() || !json.Valid(o.Source) {
			return errors.New("invalid reliability observation")
		}
		id := ids[o.Scope]
		if locked != id {
			if err := lock(ctx, tx, dialect, id); err != nil {
				return err
			}
			locked = id
		}
		data, err := json.Marshal(o)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO reliability_observations(id,series,at,created_at,payload) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO NOTHING`, Hash([]string{o.Run, o.Test}), id, date(o.At), date(o.Created), string(data))
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		group, ok := captured[id]
		if !ok {
			group = &capturedScope{scope: o.Scope, first: o.At, last: o.At, sources: map[string]bool{}}
			captured[id] = group
		}
		if o.At.Before(group.first) {
			group.first = o.At
		}
		if o.At.After(group.last) {
			group.last = o.At
		}
		group.sources[string(o.Source)] = true
	}
	for id, group := range captured {
		scope, err := json.Marshal(group.scope)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reliability_series(id,project,payload,first_at,last_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET first_at=CASE WHEN reliability_series.first_at<excluded.first_at THEN reliability_series.first_at ELSE excluded.first_at END,last_at=CASE WHEN reliability_series.last_at>excluded.last_at THEN reliability_series.last_at ELSE excluded.last_at END`, id, group.scope.Project, string(scope), date(group.first), date(group.last))
		if err != nil {
			return err
		}
		for source := range group.sources {
			if _, err = tx.ExecContext(ctx, `INSERT INTO reliability_authorities(series,source) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, source); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO reliability_policies(series,max_age) VALUES($1,0) ON CONFLICT DO NOTHING`, id); err != nil {
			return err
		}
		if err = dirty(ctx, tx, id, group.first, group.last.Add(24*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}

func dirty(ctx context.Context, tx *sql.Tx, id string, since, until time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO reliability_jobs(series,since_at,until_at) VALUES($1,$2,$3) ON CONFLICT(series) DO UPDATE SET since_at=CASE WHEN reliability_jobs.since_at<excluded.since_at THEN reliability_jobs.since_at ELSE excluded.since_at END,until_at=CASE WHEN reliability_jobs.until_at>excluded.until_at THEN reliability_jobs.until_at ELSE excluded.until_at END`, id, date(since), date(until))
	return err
}

type Service struct{ Store *storage.Store }

func (s *Service) EnsurePolicy(ctx context.Context, scope Scope, maxAge int) error {
	if maxAge < 0 || maxAge > 86400 {
		return errors.New("monitor freshness must be between zero and 86400 seconds")
	}
	return s.Store.Transaction(ctx, func(_ *storage.Store, tx *sql.Tx) error {
		id := scope.ID()
		if err := lock(ctx, tx, s.Store.Dialect, id); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO reliability_policies(series,max_age) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, maxAge)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil || count == 0 {
			return err
		}
		var first, last string
		if err = tx.QueryRowContext(ctx, `SELECT first_at,last_at FROM reliability_series WHERE id=$1`, id).Scan(&first, &last); errors.Is(err, sql.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		a, err := time.Parse(stamp, first)
		if err != nil {
			return err
		}
		b, err := time.Parse(stamp, last)
		if err != nil {
			return err
		}
		return dirty(ctx, tx, id, a, b.Add(24*time.Hour))
	})
}

func (s *Service) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			_ = s.Process(ctx, 25)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}

func (s *Service) Process(ctx context.Context, limit int) error {
	rows, err := s.Store.DB.QueryContext(ctx, `SELECT series FROM reliability_jobs ORDER BY since_at LIMIT $1`, limit)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = s.ProcessSeries(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
