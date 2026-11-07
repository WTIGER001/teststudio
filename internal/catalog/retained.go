package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RetainedTest struct {
	Project       string    `json:"projectId"`
	Suite         string    `json:"suiteId"`
	SuiteName     string    `json:"suiteName"`
	SuiteArchived bool      `json:"suiteArchived"`
	Test          Test      `json:"test"`
	Revision      int64     `json:"revision"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Current       bool      `json:"current"`
	Lifecycle     string    `json:"lifecycle"`
}

func (s *Service) RetainedTest(ctx context.Context, user, project, suite, id string) (RetainedTest, error) {
	result := RetainedTest{}
	if !domain.ValidID(project) || !domain.ValidID(suite) || !domain.ValidID(id) {
		return result, errors.New("valid project, suite, and test identifiers are required")
	}
	for _, object := range []string{"suite/" + suite, "test/" + suite + "/" + id} {
		if object == "suite/"+suite {
			if err := s.Auth.CheckSuiteTestList(ctx, user, project, suite); err != nil {
				return result, err
			}
			continue
		}
		if err := s.Auth.Check(ctx, user, project, object, "catalog.read"); err != nil {
			return result, err
		}
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return result, err
	}
	configuration, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return result, err
	}
	row, err := s.Store.Get(ctx, Bucket("tests", project, suite), id)
	current := err == nil
	var test Test
	if current {
		test, err = storage.Decode[Test](row)
	} else if errors.Is(err, storage.ErrNotFound) {
		page, readErr := s.Store.ListNewest(ctx, Bucket("test-revisions", project, suite)+"/"+id, project, "", 1)
		if readErr != nil {
			return result, readErr
		}
		if len(page.Items) == 0 {
			return result, storage.ErrNotFound
		}
		row = page.Items[0]
		revision, readErr := storage.Decode[Revision](row)
		if readErr != nil {
			return result, readErr
		}
		test = revision.Test
		err = nil
	}
	if err != nil {
		return result, err
	}
	if test.ID != id || configuration.ID != suite {
		return result, storage.ErrNotFound
	}
	lifecycle := test.State
	if !current {
		lifecycle = "removed"
	}
	result = RetainedTest{Project: project, Suite: suite, SuiteName: configuration.Name, SuiteArchived: configuration.Archived, Test: test, Revision: row.Revision, UpdatedAt: row.Updated, Current: current, Lifecycle: lifecycle}
	if !current {
		result.Revision = 0
	}
	return result, nil
}
