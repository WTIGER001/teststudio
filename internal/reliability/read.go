package reliability

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrPending = errors.New("reliability history is still being aggregated; retry shortly")

func (s *Service) Read(ctx context.Context, reader *authorization.RunReadBatch, q Query) (Row, error) {
	row := Row{State: "unknown", Metric: "outcomes", History: []Point{}, Buckets: []Bucket{}}
	if q.MaxAge > 0 {
		row.Metric = "availability"
	}
	if q.Until.IsZero() {
		q.Until = time.Now().UTC()
	}
	if q.Until.After(time.Now().Add(time.Second)) || q.MaxAge < 0 || q.MaxAge > 86400 {
		return row, errors.New("invalid reporting end or monitoring freshness")
	}
	id := q.Scope.ID()
	if q.Since.IsZero() {
		first, err := s.firstVisible(ctx, reader, q)
		if err != nil {
			return row, err
		}
		q.Since = first
	}
	if q.Since.After(q.Until) || q.Until.Sub(q.Since) > 100*366*24*time.Hour {
		return row, errors.New("report boundaries must be ordered and within 100 years")
	}
	if q.Resolution == "" {
		q.Resolution = Resolution(q.Since, q.Until)
	}
	at, err := boundary(q.Since, q.Resolution)
	if err != nil {
		return row, err
	}
	for at.Before(q.Until) {
		next := advance(at, q.Resolution)
		start := at
		if start.Before(q.Since) {
			start = q.Since
		}
		end := minTime(next, q.Until)
		row.Buckets = append(row.Buckets, Bucket{Since: start, Until: end, Partial: !start.Equal(at) || !end.Equal(next)})
		if len(row.Buckets) > 120 {
			return row, errors.New("report resolution exceeds 120 buckets; choose a coarser resolution")
		}
		at = next
	}
	if err = s.EnsurePolicy(ctx, q.Scope, q.MaxAge); err != nil {
		return row, err
	}
	for i := 0; i < 128; i++ {
		var dirty string
		err = s.Store.DB.QueryRowContext(ctx, `SELECT since_at FROM reliability_jobs WHERE series=$1`, id).Scan(&dirty)
		if errors.Is(err, sql.ErrNoRows) || err == nil && dirty >= date(q.Until) {
			break
		}
		if err != nil {
			return row, err
		}
		if i == 127 {
			return row, ErrPending
		}
		if err = s.ProcessSeries(ctx, id); err != nil {
			return row, err
		}
	}
	compact, err := s.uniformSources(ctx, reader, q.Project, id)
	if err != nil {
		return row, err
	}
	summaries := []Summary{}
	var latest []Observation
	var current Observation
	err = s.Store.ReadTransaction(ctx, func(_ *storage.Store, tx *sql.Tx) error {
		var pending string
		if err := tx.QueryRowContext(ctx, `SELECT since_at FROM reliability_jobs WHERE series=$1`, id).Scan(&pending); err == nil && pending < date(q.Until) {
			return ErrPending
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var recentErr error
		latest, current, recentErr = recent(ctx, tx, id, q.Until)
		if recentErr != nil {
			return recentErr
		}
		for cursor := q.Since; cursor.Before(q.Until); {
			step := time.Hour
			resolution := "hour"
			if q.Resolution != "hour" && q.Resolution != "6h" && cursor.Equal(cursor.Truncate(24*time.Hour)) && q.Until.Sub(cursor) >= 24*time.Hour {
				step = 24 * time.Hour
				resolution = "day"
			}
			end := minTime(cursor.Truncate(step).Add(step), q.Until)
			var summary Summary
			if cursor.Equal(cursor.Truncate(step)) && end.Equal(cursor.Add(step)) {
				var data string
				column := "payload"
				if compact {
					column = "compact"
				}
				err := tx.QueryRowContext(ctx, `SELECT `+column+` FROM reliability_summaries WHERE series=$1 AND max_age=$2 AND resolution=$3 AND start_at=$4`, id, q.MaxAge, resolution, date(cursor)).Scan(&data)
				if err == nil {
					if err = json.Unmarshal([]byte(data), &summary); err != nil {
						return err
					}
				} else if errors.Is(err, sql.ErrNoRows) {
					summary = Summary{Since: cursor, Until: end}
				} else {
					return err
				}
			} else {
				var err error
				summary, err = build(ctx, tx, id, cursor, end, q.MaxAge)
				if err != nil {
					return err
				}
			}
			summaries = append(summaries, summary)
			cursor = end
		}
		return nil
	})
	if err != nil {
		return row, err
	}
	allowed := map[string]bool{}
	contexts := map[string]bool{}
	for _, summary := range summaries {
		index := 0
		for index < len(row.Buckets) && !summary.Since.Before(row.Buckets[index].Until) {
			index++
		}
		if index == len(row.Buckets) {
			continue
		}
		for _, part := range summary.Parts {
			key := part.Run
			if key == "" {
				key = string(part.Source)
			}
			permit, ok := allowed[key]
			if !ok {
				err := reader.Check(ctx, q.Project, storage.Record{Bucket: "runs", ID: part.Run, Project: q.Project, Data: part.Source})
				if err != nil && !errors.Is(err, authorization.ErrForbidden) {
					return row, err
				}
				permit = err == nil
				allowed[key] = permit
			}
			if !permit || q.Match != "" && part.Match != q.Match {
				continue
			}
			contexts[part.Context] = true
			row.Buckets[index].Totals.Add(part.Totals)
		}
	}
	for i := range row.Buckets {
		b := &row.Buckets[i]
		finalize(b, q.MaxAge)
		row.Totals.Add(b.Totals)
	}
	if q.MaxAge > 0 {
		row.Totals.Unknown = max(0, q.Until.Sub(q.Since).Seconds()-row.Totals.Up-row.Totals.Down)
	}
	row.PassRate = ratio(float64(row.Totals.Counts.Passed), float64(row.Totals.Counts.Known()))
	row.Availability = availability(row.Totals, q.Since, q.Until)
	row.Contexts = len(contexts)
	for i := len(latest) - 1; i >= 0; i-- {
		o := latest[i]
		err := reader.Check(ctx, q.Project, storage.Record{Bucket: "runs", ID: o.Run, Project: q.Project, Data: o.Source})
		if err != nil && !errors.Is(err, authorization.ErrForbidden) {
			return row, err
		}
		if err != nil {
			continue
		}
		state := o.State
		if q.Match != "" && q.Match != o.Match {
			state = "unknown"
		}
		if !o.At.Before(q.Since) {
			row.History = append(row.History, Point{At: o.At, State: state})
		}
	}
	if current.Run != "" {
		permit := reader.Check(ctx, q.Project, storage.Record{Bucket: "runs", ID: current.Run, Project: q.Project, Data: current.Source})
		if permit != nil && !errors.Is(permit, authorization.ErrForbidden) {
			return row, permit
		}
		if permit == nil && (q.Match == "" || q.Match == current.Match) && (q.MaxAge == 0 || q.Until.Sub(current.At) < time.Duration(q.MaxAge)*time.Second) {
			row.State = current.State
			if q.MaxAge > 0 {
				row.State = current.MonitorState
			}
		}
	}
	return row, reader.Current(ctx)
}

func (s *Service) firstVisible(ctx context.Context, reader *authorization.RunReadBatch, q Query) (time.Time, error) {
	afterAt, afterID := "", ""
	for {
		rows, err := s.Store.DB.QueryContext(ctx, `SELECT id,payload FROM reliability_observations WHERE series=$1 AND at<=$2 AND (at>$3 OR (at=$3 AND id>$4)) ORDER BY at,id LIMIT 100`, q.Scope.ID(), date(q.Until), afterAt, afterID)
		if err != nil {
			return time.Time{}, err
		}
		values := []Observation{}
		ids := []string{}
		for rows.Next() {
			var id, data string
			var o Observation
			if err = rows.Scan(&id, &data); err != nil {
				rows.Close()
				return time.Time{}, err
			}
			if err = json.Unmarshal([]byte(data), &o); err != nil {
				rows.Close()
				return time.Time{}, err
			}
			values = append(values, o)
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return time.Time{}, err
		}
		for _, o := range values {
			err := reader.Check(ctx, q.Project, storage.Record{Bucket: "runs", ID: o.Run, Project: q.Project, Data: o.Source})
			if err == nil {
				retained, err := s.firstRetained(ctx, reader, q)
				if err != nil {
					return time.Time{}, err
				}
				return minTime(o.At, retained), nil
			}
			if !errors.Is(err, authorization.ErrForbidden) {
				return time.Time{}, err
			}
		}
		if len(values) < 100 {
			return s.firstRetained(ctx, reader, q)
		}
		afterAt = date(values[len(values)-1].At)
		afterID = ids[len(ids)-1]
	}
}

func availability(t Totals, since, until time.Time) Availability {
	return Availability{Percent: ratio(t.Up, t.Up+t.Down), Coverage: value(ratio(t.Up+t.Down, until.Sub(since).Seconds())), Up: t.Up, Down: t.Down, Unknown: max(0, until.Sub(since).Seconds()-t.Up-t.Down), Since: since}
}
func value(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
func finalize(b *Bucket, maxAge int) {
	b.PassRate = ratio(float64(b.Totals.Counts.Passed), float64(b.Totals.Counts.Known()))
	b.State = "unknown"
	if maxAge > 0 {
		a := availability(b.Totals, b.Since, b.Until)
		b.Availability = &a
		b.Totals.Unknown = a.Unknown
		if a.Up > 0 {
			b.State = "passed"
		}
		if a.Down > 0 {
			b.State = "flaky"
			if a.Up == 0 && a.Unknown == 0 {
				b.State = "failed"
			}
		}
		if b.Totals.Counts.Failed+b.Totals.Counts.TimedOut+b.Totals.Counts.Flaky > 0 && b.State != "failed" {
			b.State = "flaky"
		}
	} else {
		c := b.Totals.Counts
		if c.Passed > 0 {
			b.State = "passed"
		}
		if c.Flaky > 0 {
			b.State = "flaky"
		}
		if c.TimedOut > 0 {
			b.State = "timed-out"
		}
		if c.Failed > 0 {
			b.State = "failed"
		}
		if c.Known() == 0 && c.Skipped > 0 {
			b.State = "skipped"
		}
	}
}

func recent(ctx context.Context, tx *sql.Tx, id string, until time.Time) ([]Observation, Observation, error) {
	values := []Observation{}
	rows, err := tx.QueryContext(ctx, `SELECT payload FROM reliability_observations WHERE series=$1 AND at<$2 ORDER BY at DESC,created_at DESC,id DESC LIMIT 30`, id, date(until))
	if err != nil {
		return nil, Observation{}, err
	}
	for rows.Next() {
		var data string
		var o Observation
		if err = rows.Scan(&data); err != nil {
			rows.Close()
			return nil, Observation{}, err
		}
		if err = json.Unmarshal([]byte(data), &o); err != nil {
			rows.Close()
			return nil, Observation{}, err
		}
		values = append(values, o)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, Observation{}, err
	}
	var data string
	var current Observation
	err = tx.QueryRowContext(ctx, `SELECT payload FROM reliability_observations WHERE series=$1 AND at<$2 ORDER BY created_at DESC,at DESC,id DESC LIMIT 1`, id, date(until)).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return values, current, nil
	}
	if err != nil {
		return nil, current, err
	}
	err = json.Unmarshal([]byte(data), &current)
	return values, current, err
}
