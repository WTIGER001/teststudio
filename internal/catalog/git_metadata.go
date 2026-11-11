package catalog

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) guardGitMetadata(ctx context.Context, project string, test Test) error {
	if test.RepositorySource == "" {
		return nil
	}
	row, err := s.Store.Get(ctx, "repository-sources/"+project, test.RepositorySource)
	if err != nil {
		return err
	}
	var source struct {
		Configuration struct {
			Mode string `json:"accessMode"`
		} `json:"configuration"`
	}
	source, err = storage.Decode[struct {
		Configuration struct {
			Mode string `json:"accessMode"`
		} `json:"configuration"`
	}](row)
	if err != nil {
		return err
	}
	if source.Configuration.Mode == "read-write" {
		return errors.New("this test uses Git-owned metadata; review and commit changes through Repository metadata")
	}
	return nil
}
