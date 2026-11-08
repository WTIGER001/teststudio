package reliability

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) firstRetained(ctx context.Context, reader *authorization.RunReadBatch, q Query) (time.Time, error) {
	after := ""
	for {
		rows, err := s.Store.DB.QueryContext(ctx, `SELECT start_at,payload FROM reliability_summaries WHERE series=$1 AND max_age=0 AND resolution='day' AND start_at>$2 AND start_at<$3 ORDER BY start_at LIMIT 10`, q.Scope.ID(), after, date(q.Until))
		if err != nil {
			return time.Time{}, err
		}
		summaries := []Summary{}
		for rows.Next() {
			var at, data string
			var value Summary
			if err = rows.Scan(&at, &data); err != nil {
				rows.Close()
				return time.Time{}, err
			}
			if err = json.Unmarshal([]byte(data), &value); err != nil {
				rows.Close()
				return time.Time{}, err
			}
			after = at
			summaries = append(summaries, value)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return time.Time{}, err
		}
		for _, summary := range summaries {
			first := q.Until
			for _, part := range summary.Parts {
				if part.First.IsZero() || !part.First.Before(first) {
					continue
				}
				err := reader.Check(ctx, q.Project, storage.Record{Bucket: "runs", ID: part.Run, Project: q.Project, Data: part.Source})
				if err == nil {
					first = part.First
				} else if !errors.Is(err, authorization.ErrForbidden) {
					return time.Time{}, err
				}
			}
			if first.Before(q.Until) {
				return first, nil
			}
		}
		if len(summaries) < 10 {
			return q.Until, nil
		}
	}
}
