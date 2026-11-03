package catalog

import (
	"context"
	"errors"
	"slices"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) SavedSelection(ctx context.Context, user, project, suite, id string) (SavedSelection, storage.Record, error) {
	if !domain.ValidID(project) || !domain.ValidID(suite) || !domain.ValidID(id) {
		return SavedSelection{}, storage.Record{}, errors.New("valid project, suite, and saved selection identifiers are required")
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return SavedSelection{}, storage.Record{}, err
	}
	row, err := s.Store.Get(ctx, Bucket("saved-selections", project, suite), id)
	if err != nil {
		return SavedSelection{}, row, err
	}
	value, err := storage.Decode[SavedSelection](row)
	if err == nil {
		err = s.savedSelectionRead(ctx, user, project, suite, value)
	}
	return value, row, err
}

func (s *Service) savedSelectionRead(ctx context.Context, user, project, suite string, value SavedSelection) error {
	for _, id := range value.Spec.IDs {
		if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+id, "catalog.read"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SavedSelections(ctx context.Context, user, project, suite, cursor string, limit int, archived bool) (storage.Page, error) {
	if !domain.ValidID(project) || !domain.ValidID(suite) || limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("saved selection lists require valid scope and a limit of 1 through 100")
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return storage.Page{}, err
	}
	page, err := s.Store.List(ctx, Bucket("saved-selections", project, suite), project, cursor, limit)
	if err != nil {
		return page, err
	}
	filtered := []storage.Record{}
	for _, row := range page.Items {
		value, err := storage.Decode[SavedSelection](row)
		if err != nil {
			return page, err
		}
		if value.Archived && !archived {
			continue
		}
		if err := s.savedSelectionRead(ctx, user, project, suite, value); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return page, err
		}
		filtered = append(filtered, row)
	}
	page.Items = filtered
	return page, nil
}

func (s *Service) SaveSelection(ctx context.Context, user, project, suite string, value SavedSelection, revision int64) error {
	if err := validateSavedSelection(value); err != nil {
		return err
	}
	return s.withSavedSelectionUpdate(ctx, user, project, suite, "catalog.update", func(bound *Service) error {
		if err := bound.savedSelectionRead(ctx, user, project, suite, value); err != nil {
			return err
		}
		var before any
		if revision > 0 {
			old, _, err := bound.SavedSelection(ctx, user, project, suite, value.ID)
			if err != nil {
				return err
			}
			if old.Archived || value.Archived != old.Archived {
				return errors.New("use saved selection archive or restore actions before editing")
			}
			before = old
		} else if value.Archived {
			return errors.New("new saved selections start active")
		}
		value.Spec.IDs = slices.Clone(value.Spec.IDs)
		slices.Sort(value.Spec.IDs)
		change, err := storage.Put(Bucket("saved-selections", project, suite), value.ID, project, revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "selection.save", suite+"/"+value.ID, before, value)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, audit)
	})
}

func (s *Service) ChangeSelectionState(ctx context.Context, user, project, suite, id, action string, revision int64) error {
	permission := map[string]string{"archive": "catalog.archive", "restore": "catalog.restore"}[action]
	if permission == "" {
		return errors.New("select an archive or restore action")
	}
	return s.withSavedSelectionUpdate(ctx, user, project, suite, permission, func(bound *Service) error {
		before, _, err := bound.SavedSelection(ctx, user, project, suite, id)
		if err != nil {
			return err
		}
		after := before
		after.Archived = action == "archive"
		change, err := storage.Put(Bucket("saved-selections", project, suite), id, project, revision, after)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "selection."+action, suite+"/"+id, before, after)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, audit)
	})
}

func (s *Service) withSavedSelectionUpdate(ctx context.Context, user, project, suite, permission string, update func(*Service) error) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		for _, permission := range []string{"catalog.read", permission} {
			if err := auth.Check(ctx, user, project, "suite/"+suite, permission); err != nil {
				return err
			}
		}
		if _, _, err := bound.suite(ctx, project, suite); err != nil {
			return err
		}
		return update(bound)
	})
}
