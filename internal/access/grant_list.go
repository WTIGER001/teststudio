package access

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ReadGrant(ctx context.Context, actor, project, id string) (GrantRecord, error) {
	for _, action := range []string{"permissions.read", "members.read"} {
		if _, err := s.assignmentScope(ctx, actor, project, action); err != nil {
			return GrantRecord{}, err
		}
	}
	value, row, err := s.loadGrant(ctx, project, id)
	if err != nil {
		return GrantRecord{}, err
	}
	if !value.Present {
		return GrantRecord{}, storage.ErrNotFound
	}
	if err := s.grantAuthority(ctx, actor, value, false); err != nil {
		return GrantRecord{}, err
	}
	if err := s.verifyNativeGrant(ctx, value); err != nil {
		return GrantRecord{}, err
	}
	return s.grantView(ctx, actor, value, row)
}

func (s *Service) ListGrants(ctx context.Context, actor, project, cursor string, limit int) (GrantsPage, error) {
	var projectRow storage.Record
	for _, action := range []string{"permissions.read", "members.read"} {
		row, err := s.assignmentScope(ctx, actor, project, action)
		if err != nil {
			return GrantsPage{}, err
		}
		projectRow = row
	}
	page, err := s.page(ctx, grantBucket(project), project, queryScope(actor, "direct-grants", project, false), cursor, limit)
	if err != nil {
		return GrantsPage{}, err
	}
	result := GrantsPage{Items: []GrantRecord{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	result.CanCreate, err = roleCapability(func() error {
		for _, action := range []string{"permissions.grant", "members.manage"} {
			if err := s.Auth.Check(ctx, actor, project, "", action); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	if projectRow.Revision > 0 {
		state, err := storage.Decode[struct {
			Archived bool `json:"archived"`
		}](projectRow)
		if err != nil {
			return result, err
		}
		result.CanCreate = result.CanCreate && !state.Archived
	}
	for _, row := range page.Items {
		value, stored, err := s.loadGrant(ctx, project, row.ID)
		if err != nil {
			return result, err
		}
		if !value.Present {
			continue
		}
		if err := s.grantAuthority(ctx, actor, value, false); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		if err := s.verifyNativeGrant(ctx, value); err != nil {
			return result, err
		}
		view, err := s.grantView(ctx, actor, value, stored)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, view)
	}
	return result, nil
}
