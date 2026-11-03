package sessions

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) providerAt(ctx context.Context, id string, revision int64) (storage.Record, error) {
	row, err := s.Store.Get(ctx, "providers", id)
	if errors.Is(err, storage.ErrNotFound) {
		return row, ErrUnauthenticated
	}
	if err != nil {
		return row, err
	}
	provider, err := storage.Decode[struct {
		Disabled bool `json:"disabled"`
	}](row)
	if err != nil {
		return row, err
	}
	if revision < 1 || row.Revision != revision || row.Project != "" || provider.Disabled {
		return row, ErrUnauthenticated
	}
	return row, nil
}
