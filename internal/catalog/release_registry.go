package catalog

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type ReleaseRegistry struct {
	Suite         string    `json:"suiteId"`
	SuiteName     string    `json:"suiteName"`
	SuiteRevision int64     `json:"suiteRevision"`
	Active        string    `json:"activeReleaseId,omitempty"`
	Archived      bool      `json:"archived"`
	CanActivate   bool      `json:"canActivate"`
	Items         []Release `json:"items"`
}

func (s *Service) ReleaseRegistry(ctx context.Context, user, project, suite string) (ReleaseRegistry, error) {
	result := ReleaseRegistry{Items: []Release{}}
	if !domain.ValidID(project) || !domain.ValidID(suite) {
		return result, errors.New("valid project and suite identifiers are required")
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return result, err
	}
	row, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return result, err
	}
	current, err := storage.Decode[projects.Suite](row)
	if err != nil {
		return result, err
	}
	result.Suite, result.SuiteName, result.SuiteRevision, result.Active, result.Archived = suite, current.Name, row.Revision, current.ActiveRelease, current.Archived
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err == nil {
		result.CanActivate = !current.Archived
	} else if !errors.Is(err, authorization.ErrForbidden) {
		return result, err
	}
	rows, err := s.Store.All(ctx, Bucket("releases", project, suite), project)
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		release, err := storage.Decode[Release](row)
		if err != nil {
			return result, err
		}
		if release.Project != project || release.Suite != suite || release.ID != row.ID {
			return result, storage.ErrNotFound
		}
		result.Items = append(result.Items, release)
	}
	slices.SortFunc(result.Items, func(a, b Release) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		return 1
	})
	return result, nil
}
