package reliability

import (
	"context"
	"encoding/json"

	"github.com/wtiger001/teststudio/internal/authorization"
)

func (s *Service) uniformSources(ctx context.Context, reader *authorization.RunReadBatch, project, id string) (bool, error) {
	rows, err := s.Store.DB.QueryContext(ctx, `SELECT source FROM reliability_authorities WHERE series=$1 ORDER BY source`, id)
	if err != nil {
		return false, err
	}
	sources := []json.RawMessage{}
	for rows.Next() {
		var source string
		if err = rows.Scan(&source); err != nil {
			rows.Close()
			return false, err
		}
		sources = append(sources, json.RawMessage(source))
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return false, err
	}
	if len(sources) == 0 {
		return false, nil
	}
	for _, source := range sources {
		uniform, err := reader.UniformSource(ctx, project, source)
		if err != nil {
			return false, err
		}
		if !uniform {
			return false, nil
		}
	}
	return true, nil
}
