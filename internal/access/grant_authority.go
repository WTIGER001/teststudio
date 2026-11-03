package access

import (
	"context"
	"errors"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) grantAuthority(ctx context.Context, actor string, value grantState, manage bool) error {
	actions := []string{"permissions.read", "members.read"}
	groupAction := "groups.read"
	if manage {
		actions = []string{"permissions.grant", "members.manage"}
		groupAction = "groups.manage"
	}
	for _, action := range actions {
		if _, err := s.assignmentScope(ctx, actor, value.Project, action); err != nil {
			return err
		}
		for _, object := range []string{string(value.PrincipalKind) + "/" + value.PrincipalID, "grant/" + value.id(), value.Object} {
			if err := s.Auth.Check(ctx, actor, value.Project, object, action); err != nil {
				return err
			}
		}
	}
	if value.PrincipalKind == permissions.PrincipalGroup {
		if _, _, err := s.group(ctx, actor, value.PrincipalID, groupAction); err != nil {
			return err
		}
	}
	if _, _, err := s.assignmentPrincipal(ctx, value.Project, assignmentIdentity{PrincipalKind: value.PrincipalKind, PrincipalID: value.PrincipalID}, false); err != nil {
		return err
	}
	if manage {
		return s.Auth.Check(ctx, actor, value.Project, value.Object, value.Permission)
	}
	return nil
}

func (s *Service) grantCanAdd(ctx context.Context, actor string, value grantState) (storage.Record, error) {
	row, err := s.assignmentScope(ctx, actor, value.Project, "permissions.grant")
	if err != nil {
		return row, err
	}
	if row.Revision > 0 {
		project, err := storage.Decode[struct {
			Archived bool `json:"archived"`
		}](row)
		if err != nil {
			return row, err
		}
		if project.Archived {
			return row, errors.New("restore the archived project before changing grants")
		}
	}
	if err := s.checkPolicyObject(ctx, value.Project, value.Object); err != nil {
		return row, err
	}
	_, _, err = s.assignmentPrincipal(ctx, value.Project, assignmentIdentity{PrincipalKind: value.PrincipalKind, PrincipalID: value.PrincipalID}, true)
	return row, err
}

func (s *Service) grantView(ctx context.Context, actor string, value grantState, row storage.Record) (GrantRecord, error) {
	name, active, err := s.assignmentPrincipal(ctx, value.Project, assignmentIdentity{PrincipalKind: value.PrincipalKind, PrincipalID: value.PrincipalID}, false)
	if err != nil {
		return GrantRecord{}, err
	}
	view := GrantView{grantIdentity: value.grantIdentity, ExpiresAt: value.ExpiresAt, Expired: value.ExpiresAt != nil && !time.Now().Before(*value.ExpiresAt), PrincipalName: name, PrincipalActive: active}
	view.CanRemove, err = roleCapability(func() error { return s.grantAuthority(ctx, actor, value, true) })
	if err != nil {
		return GrantRecord{}, err
	}
	if view.CanRemove && active {
		view.CanUpdate = true
		if value.Project != "" {
			project, err := s.Store.Get(ctx, "projects", value.Project)
			if err != nil {
				return GrantRecord{}, err
			}
			state, err := storage.Decode[struct {
				Archived bool `json:"archived"`
			}](project)
			if err != nil {
				return GrantRecord{}, err
			}
			view.CanUpdate = !state.Archived
		}
		if view.CanUpdate {
			err := s.checkPolicyObject(ctx, value.Project, value.Object)
			if errors.Is(err, storage.ErrNotFound) {
				view.CanUpdate = false
			} else if err != nil {
				return GrantRecord{}, err
			}
		}
	}
	return GrantRecord{ID: row.ID, Project: row.Project, Revision: row.Revision, Updated: row.Updated, Data: view}, nil
}
