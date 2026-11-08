package reliability

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/paging"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Check struct {
	At       time.Time `json:"at"`
	State    string    `json:"state"`
	Duration *float64  `json:"durationMs"`
	Censored bool      `json:"censored"`
	Run      string    `json:"runId,omitempty"`
}

type CheckPage struct {
	Items   []Check `json:"items"`
	Cursor  string  `json:"cursor,omitempty"`
	Partial bool    `json:"partial"`
}

type checkCursor struct {
	At      string    `json:"at"`
	ID      string    `json:"id"`
	Expires time.Time `json:"expires"`
}

func (s *Service) Checks(ctx context.Context, reader *authorization.RunReadBatch, q Query, audience, cursor string, limit int) (CheckPage, error) {
	page := CheckPage{Items: []Check{}}
	if limit < 1 || limit > 100 || q.Until.IsZero() || !q.Since.Before(q.Until) || q.Until.After(time.Now().Add(time.Second)) {
		return page, errors.New("individual checks require ordered boundaries and a page limit from 1 through 100")
	}
	scope := Hash([]any{q.Scope, q.Since.UTC(), q.Until.UTC(), q.Match, audience, limit})
	raw, err := paging.Open(ctx, s.Store, "reliability-check-cursors", scope, cursor)
	if err != nil {
		return page, err
	}
	position := checkCursor{At: date(q.Until), Expires: time.Now().UTC().Add(15 * time.Minute)}
	if raw != "" {
		if json.Unmarshal([]byte(raw), &position) != nil || !time.Now().Before(position.Expires) {
			return page, errors.New("individual checks cursor expired; refresh the results")
		}
	}
	scanned := 0
	more := false
	for scanned < 500 && len(page.Items) < limit {
		batch := min(50, limit-len(page.Items), 500-scanned)
		rows, err := s.Store.DB.QueryContext(ctx, `SELECT id,at,payload FROM reliability_observations WHERE series=$1 AND at>=$2 AND at<$3 AND (at<$4 OR (at=$4 AND id<$5)) ORDER BY at DESC,id DESC LIMIT $6`, q.Scope.ID(), date(q.Since), date(q.Until), position.At, position.ID, batch+1)
		if err != nil {
			return page, err
		}
		type entry struct {
			id, at string
			value  Observation
		}
		values := []entry{}
		for rows.Next() {
			var v entry
			var data string
			if err = rows.Scan(&v.id, &v.at, &data); err == nil {
				err = json.Unmarshal([]byte(data), &v.value)
			}
			if err != nil {
				rows.Close()
				return page, err
			}
			values = append(values, v)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return page, err
		}
		more = len(values) > batch
		if more {
			values = values[:batch]
		}
		for _, v := range values {
			scanned++
			position.At = v.at
			position.ID = v.id
			o := v.value
			err = reader.Check(ctx, q.Project, storage.Record{Bucket: "runs", ID: o.Run, Project: q.Project, Data: o.Source})
			if errors.Is(err, authorization.ErrForbidden) || err == nil && q.Match != "" && o.Match != q.Match {
				continue
			}
			if err != nil {
				return page, err
			}
			page.Items = append(page.Items, Check{At: o.At, State: o.State, Duration: o.DurationMs, Censored: o.Censored, Run: o.Run})
		}
		if !more {
			break
		}
	}
	page.Partial = scanned == 500 && more
	if more {
		data, _ := json.Marshal(position)
		page.Cursor, err = paging.Seal(ctx, s.Store, "reliability-check-cursors", scope, string(data))
		if err != nil {
			return page, err
		}
	}
	return page, reader.Current(ctx)
}
