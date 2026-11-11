package sources

import (
	"context"
	"fmt"

	"github.com/wtiger001/teststudio/internal/storage"
)

func metadataHistoryBucket(project, source, suite string) string {
	return "repository-metadata-history/" + project + "/" + source + "/" + suite
}
func metadataHistoryChange(project, source, suite string, change storage.Change) (storage.Change, error) {
	value, err := storage.Decode[RepositoryMetadata](change.Record)
	if err != nil {
		return storage.Change{}, err
	}
	return storage.Put(metadataHistoryBucket(project, source, suite), fmt.Sprintf("%020d", change.Revision+1), project, 0, value)
}
func (s *Service) RepositoryMetadataHistory(ctx context.Context, user, project, source, suite, cursor string, limit int) (storage.Page, error) {
	if _, _, _, err := s.repositoryBinding(ctx, user, project, source, suite, "read"); err != nil {
		return storage.Page{}, err
	}
	page, err := s.Store.ListNewest(ctx, metadataHistoryBucket(project, source, suite), project, cursor, limit)
	if err != nil {
		return page, err
	}
	for _, row := range page.Items {
		m, err := storage.Decode[RepositoryMetadata](row)
		if err != nil {
			return storage.Page{}, err
		}
		for _, test := range m.Catalog.Tests {
			if err = s.Auth.Check(ctx, user, project, "test/"+suite+"/"+test.ID, "catalog.read"); err != nil {
				return storage.Page{}, err
			}
		}
	}
	return page, nil
}
