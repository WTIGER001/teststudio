package reliability

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func observations(ctx context.Context, tx *sql.Tx, id string, since, until time.Time) ([]Observation, error) {
	rows, err := tx.QueryContext(ctx, `SELECT payload FROM reliability_observations WHERE series=$1 AND at>=$2 AND at<$3 ORDER BY at,created_at,id`, id, date(since), date(until))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []Observation{}
	for rows.Next() {
		var data string
		var o Observation
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(data), &o); err != nil {
			return nil, err
		}
		values = append(values, o)
	}
	return values, rows.Err()
}

func build(ctx context.Context, tx *sql.Tx, id string, since, until time.Time, maxAge int) (Summary, error) {
	start := since.Add(-time.Duration(maxAge) * time.Second)
	values, err := observations(ctx, tx, id, start, until)
	if err != nil {
		return Summary{}, err
	}
	var previous sql.NullString
	if maxAge > 0 {
		if err = tx.QueryRowContext(ctx, `SELECT MAX(created_at) FROM reliability_observations WHERE series=$1 AND at<$2`, id, date(start)).Scan(&previous); err != nil {
			return Summary{}, err
		}
	}
	return aggregate(values, previous.String, since, until, maxAge), nil
}

func aggregate(values []Observation, previous string, since, until time.Time, maxAge int) Summary {
	result := Summary{Since: since, Until: until, Parts: []Part{}}
	parts := map[string]int{}
	part := func(o Observation) *Part {
		key := Hash([]string{o.Run, o.Context, o.Match})
		i, ok := parts[key]
		if !ok {
			i = len(result.Parts)
			parts[key] = i
			result.Parts = append(result.Parts, Part{Run: o.Run, First: o.At, Source: o.Source, Context: o.Context, Match: o.Match})
		}
		return &result.Parts[i]
	}
	valid := []Observation{}
	latest := previous
	for _, o := range values {
		if !o.At.Before(until) {
			continue
		}
		if !o.At.Before(since) {
			part(o).Totals.Observe(o)
		}
		if maxAge > 0 && date(o.Created) >= latest {
			valid = append(valid, o)
			latest = date(o.Created)
		}
	}
	for i, o := range valid {
		a := o.At
		if a.Before(since) {
			a = since
		}
		b := o.At.Add(time.Duration(maxAge) * time.Second)
		if i+1 < len(valid) && valid[i+1].At.Before(b) {
			b = valid[i+1].At
		}
		if b.After(until) {
			b = until
		}
		if !b.After(a) {
			continue
		}
		switch o.MonitorState {
		case "passed":
			part(o).Totals.Up += b.Sub(a).Seconds()
		case "failed", "timed-out", "flaky":
			part(o).Totals.Down += b.Sub(a).Seconds()
		}
	}
	return result
}

func (s *Service) ProcessSeries(ctx context.Context, id string) error {
	return s.Store.Transaction(ctx, func(_ *storage.Store, tx *sql.Tx) error {
		if err := lock(ctx, tx, s.Store.Dialect, id); err != nil {
			return err
		}
		var first, last string
		if err := tx.QueryRowContext(ctx, `SELECT since_at,until_at FROM reliability_jobs WHERE series=$1`, id).Scan(&first, &last); errors.Is(err, sql.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		since, err := time.Parse(stamp, first)
		if err != nil {
			return err
		}
		until, err := time.Parse(stamp, last)
		if err != nil {
			return err
		}
		since = since.Truncate(24 * time.Hour)
		finish := minTime(until.Truncate(24*time.Hour).Add(24*time.Hour), since.Add(31*24*time.Hour))
		rows, err := tx.QueryContext(ctx, `SELECT max_age FROM reliability_policies WHERE series=$1 ORDER BY max_age`, id)
		if err != nil {
			return err
		}
		policies := []int{}
		for rows.Next() {
			var p int
			if err = rows.Scan(&p); err != nil {
				rows.Close()
				return err
			}
			policies = append(policies, p)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		statement, err := tx.PrepareContext(ctx, `INSERT INTO reliability_summaries(series,max_age,resolution,start_at,payload,compact) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(series,max_age,resolution,start_at) DO UPDATE SET payload=excluded.payload,compact=excluded.compact`)
		if err != nil {
			return err
		}
		defer statement.Close()
		for day := since; day.Before(finish); day = day.Add(24 * time.Hour) {
			for _, policy := range policies {
				start := day.Add(-time.Duration(policy) * time.Second)
				values, err := observations(ctx, tx, id, start, day.Add(24*time.Hour))
				if err != nil {
					return err
				}
				var previous sql.NullString
				if policy > 0 {
					if err = tx.QueryRowContext(ctx, `SELECT MAX(created_at) FROM reliability_observations WHERE series=$1 AND at<$2`, id, date(start)).Scan(&previous); err != nil {
						return err
					}
				}
				for _, resolution := range []string{"day", "hour"} {
					step := 24 * time.Hour
					if resolution == "hour" {
						step = time.Hour
					}
					for at := day; at.Before(day.Add(24 * time.Hour)); at = at.Add(step) {
						summary := aggregate(values, previous.String, at, at.Add(step), policy)
						data, err := json.Marshal(summary)
						if err != nil {
							return err
						}
						compact, err := json.Marshal(compactSummary(summary))
						if err != nil {
							return err
						}
						if _, err = statement.ExecContext(ctx, id, policy, resolution, date(at), string(data), string(compact)); err != nil {
							return err
						}
					}
				}
			}
		}
		if finish.After(until) {
			_, err = tx.ExecContext(ctx, `DELETE FROM reliability_jobs WHERE series=$1`, id)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE reliability_jobs SET since_at=$1 WHERE series=$2`, date(finish), id)
		}
		return err
	})
}

func compactSummary(summary Summary) Summary {
	result := Summary{Since: summary.Since, Until: summary.Until, Parts: []Part{}}
	indices := map[string]int{}
	for _, part := range summary.Parts {
		key := Hash([]string{string(part.Source), part.Context, part.Match})
		index, ok := indices[key]
		if !ok {
			index = len(result.Parts)
			indices[key] = index
			result.Parts = append(result.Parts, Part{First: part.First, Source: part.Source, Context: part.Context, Match: part.Match})
		}
		if part.First.Before(result.Parts[index].First) {
			result.Parts[index].First = part.First
		}
		result.Parts[index].Totals.Add(part.Totals)
	}
	return result
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
