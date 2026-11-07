package access

import (
	"context"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"strings"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

type GroupDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Project     string `json:"projectId"`
	ManagedBy   string `json:"managedBy,omitempty"`
}

func (s *Service) ListGroups(ctx context.Context, actor, project, cursor string, limit int, archived bool) (Page, error) {
	if err := s.Auth.Check(ctx, actor, project, "", "groups.read"); err != nil {
		return Page{}, err
	}
	if project != "" {
		if _, err := s.Store.Get(ctx, "projects", project); err != nil {
			return Page{}, err
		}
	}
	page, err := s.page(ctx, "groups", project, queryScope(actor, "groups", project, archived), cursor, limit)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: []storage.Record{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		g, err := storage.Decode[identity.Group](row)
		if err != nil {
			return result, err
		}
		if g.Project != project || g.Project != row.Project || (g.Archived && !archived) {
			continue
		}
		if err := s.Auth.Check(ctx, actor, project, "group/"+g.ID, "groups.read"); err != nil {
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			return result, err
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}

func (s *Service) ReadGroup(ctx context.Context, actor, id string) (storage.Record, error) {
	_, row, err := s.group(ctx, actor, id, "groups.read")
	return row, err
}

func (s *Service) SaveGroup(ctx context.Context, actor string, d GroupDefinition, revision int64) error {
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	var old identity.Group
	r, err := s.Store.Get(ctx, "groups", d.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if r.Revision > 0 {
		old, err = storage.Decode[identity.Group](r)
		if err != nil {
			return err
		}
		if err := s.Auth.Check(ctx, actor, old.Project, "group/"+d.ID, "groups.manage"); err != nil {
			return err
		}
		if old.Project != d.Project || old.Project != r.Project {
			return errors.New("group ownership scope is permanent")
		}
		if d.ManagedBy != "" && d.ManagedBy != old.ManagedBy {
			return errors.New("group membership ownership is permanent")
		}
	}
	if err := s.Auth.Check(ctx, actor, d.Project, "group/"+d.ID, "groups.manage"); err != nil {
		return err
	}
	if r.Revision != revision {
		return storage.ErrConflict
	}
	d.Name = strings.TrimSpace(d.Name)
	if !domain.ValidID(d.ID) || d.Name == "" || len(d.Name) > 256 || len(d.Description) > 4096 || (d.Project != "" && !domain.ValidID(d.Project)) {
		return errors.New("valid group identity, scope, name, and bounded description are required")
	}
	if old.Archived {
		return errors.New("restore the archived group before editing")
	}
	changes := []storage.Change{}
	if r.Revision == 0 && d.ManagedBy != "" {
		provider, err := s.providerGroupOwner(ctx, actor, d.ManagedBy)
		if err != nil {
			return err
		}
		changes = append(changes, storage.Change{Record: provider, Assert: true})
	}
	if d.Project != "" {
		project, err := s.Store.Get(ctx, "projects", d.Project)
		if err != nil {
			return err
		}
		value, err := storage.Decode[struct {
			Archived bool `json:"archived"`
		}](project)
		if err != nil {
			return err
		}
		if value.Archived {
			return errors.New("project is archived")
		}
		changes = append(changes, storage.Change{Record: project, Assert: true})
	}
	v := old
	v.ID, v.Name, v.Description, v.Project = d.ID, d.Name, d.Description, d.Project
	if r.Revision == 0 {
		v.ManagedBy = d.ManagedBy
	}
	c, err := storage.Put("groups", d.ID, d.Project, revision, v)
	if err != nil {
		return err
	}
	return s.commit(ctx, actor, d.Project, "group.save", d.ID, "", old, v, version, append(changes, c)...)
}

func (s *Service) GroupState(ctx context.Context, actor, id, why string, revision int64, archived bool) error {
	if err := reason(why); err != nil {
		return err
	}
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	v, r, err := s.group(ctx, actor, id, "groups.manage")
	if err != nil {
		return err
	}
	if err := s.protectOwnerPrincipal(ctx, actor, permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: id}); err != nil {
		return err
	}
	if err := s.protectGroupDelegation(ctx, actor, id); err != nil {
		return err
	}
	if revision < 1 || r.Revision != revision {
		return storage.ErrConflict
	}
	old := v
	v.Archived = archived
	c, err := storage.Put("groups", id, v.Project, revision, v)
	if err != nil {
		return err
	}
	if archived {
		if err := s.preserveAdministrator(ctx, c); err != nil {
			return err
		}
	}
	action := "group.restore"
	if archived {
		action = "group.archive"
	}
	return s.commit(ctx, actor, v.Project, action, id, why, old, v, version, c)
}
