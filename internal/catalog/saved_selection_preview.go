package catalog

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) PreviewSavedSelection(ctx context.Context, user, project, suite, id, releaseID string) (SavedSelectionPreview, error) {
	value, row, err := s.SavedSelection(ctx, user, project, suite, id)
	if err != nil {
		return SavedSelectionPreview{}, err
	}
	if value.Archived {
		return SavedSelectionPreview{}, errors.New("restore the saved selection before previewing it")
	}
	if err := validateSavedSelection(value); err != nil {
		return SavedSelectionPreview{}, err
	}
	configured, suiteRow, err := s.suite(ctx, project, suite)
	if err != nil {
		return SavedSelectionPreview{}, err
	}
	if releaseID == "" {
		releaseID = configured.ActiveRelease
	}
	if !domain.ValidID(releaseID) {
		return SavedSelectionPreview{}, errors.New("publish a release before previewing a saved selection")
	}
	releaseRow, err := s.Store.Get(ctx, Bucket("releases", project, suite), releaseID)
	if err != nil {
		return SavedSelectionPreview{}, err
	}
	release, err := storage.Decode[Release](releaseRow)
	if err != nil {
		return SavedSelectionPreview{}, err
	}
	roots := slices.Clone(value.Spec.IDs)
	if value.Spec.Kind == "filter" {
		for _, test := range release.Draft.Tests {
			if matchesSavedFilter(test, *value.Spec.Filter) {
				roots = append(roots, test.ID)
			}
		}
	}
	if len(roots) == 0 {
		return SavedSelectionPreview{}, errors.New("saved selection matches no enabled tests in this release")
	}
	slices.Sort(roots)
	tests, err := ResolveSelection(release.Draft.Tests, roots, false)
	if err != nil {
		return SavedSelectionPreview{}, err
	}
	resolved := make([]string, 0, len(tests))
	for _, test := range tests {
		if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+test.ID, "catalog.read"); err != nil {
			return SavedSelectionPreview{}, err
		}
		resolved = append(resolved, test.ID)
	}
	if err := s.CheckCurrentState(ctx, project, suite, tests, false); err != nil {
		return SavedSelectionPreview{}, err
	}
	slices.Sort(resolved)
	now, previewID := time.Now().UTC(), domain.NewID()
	preview := SavedSelectionPreview{ID: previewID, Project: project, Suite: suite, Actor: user, Release: releaseID, Snapshot: SavedSelectionSnapshot{Definition: value, Revision: row.Revision, PreviewID: previewID}, Roots: roots, Resolved: resolved, Created: now, Expires: now.Add(15 * time.Minute)}
	change, err := storage.Put(Bucket("selection-previews", project, suite), previewID, project, 0, preview)
	if err != nil {
		return preview, err
	}
	return preview, s.Store.Commit(ctx, change, storage.Change{Record: row, Assert: true}, storage.Change{Record: suiteRow, Assert: true})
}

func (s *Service) SelectionFromPreview(ctx context.Context, user, project, suite, id string) (SavedSelectionPreview, []storage.Record, error) {
	if !domain.ValidID(id) {
		return SavedSelectionPreview{}, nil, errors.New("valid saved selection preview identifier is required")
	}
	row, err := s.Store.Get(ctx, Bucket("selection-previews", project, suite), id)
	if err != nil {
		return SavedSelectionPreview{}, nil, err
	}
	preview, err := storage.Decode[SavedSelectionPreview](row)
	if err != nil {
		return preview, nil, err
	}
	if preview.Actor != user || preview.Project != project || preview.Suite != suite {
		return preview, nil, authorization.ErrForbidden
	}
	if !time.Now().Before(preview.Expires) || len(preview.Roots) == 0 || len(preview.Resolved) == 0 {
		return preview, nil, errors.New("saved selection preview expired or has no tests; create a new preview")
	}
	current, definition, err := s.SavedSelection(ctx, user, project, suite, preview.Snapshot.Definition.ID)
	if err != nil {
		return preview, nil, err
	}
	if definition.Revision != preview.Snapshot.Revision {
		return preview, nil, storage.ErrConflict
	}
	if current.Archived {
		return preview, nil, errors.New("saved selection is archived")
	}
	for _, test := range preview.Resolved {
		if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+test, "catalog.read"); err != nil {
			return preview, nil, err
		}
	}
	return preview, []storage.Record{row, definition}, nil
}
