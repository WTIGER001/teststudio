package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) filterRepositoryReads(ctx context.Context, user, project string, page storage.Page) (storage.Page, error) {
	rows := []storage.Record{}
	for _, row := range page.Items {
		meta, err := storage.Decode[struct {
			Source string `json:"repositorySourceId"`
		}](row)
		if err != nil {
			return storage.Page{}, err
		}
		if meta.Source != "" {
			if err := s.checkRepository(ctx, user, project, meta.Source, "read"); errors.Is(err, authorization.ErrForbidden) {
				continue
			} else if err != nil {
				return storage.Page{}, err
			}
		}
		rows = append(rows, row)
	}
	page.Items = rows
	return page, nil
}
func (s *Service) checkSyncLease(ctx context.Context, sync RepositorySync) error {
	row, err := s.Store.Get(ctx, syncBucket(sync.Project, sync.Source), sync.ID)
	if err != nil {
		return err
	}
	current, err := storage.Decode[RepositorySync](row)
	if err != nil {
		return err
	}
	if current.Worker != sync.Worker || current.State != "fetching" || current.LeaseUntil == nil || !time.Now().Before(*current.LeaseUntil) {
		return storage.ErrConflict
	}
	return nil
}
