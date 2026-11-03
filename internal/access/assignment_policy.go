package access

import (
	"context"
	"errors"
	"slices"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func builtinRole(project, role string) (authorization.RoleDefinition, bool) {
	for _, r := range authorization.BuiltInRoles() {
		if r.ID == role && (project == "") == (role == "builtin.teststudio.installation-admin") {
			return r, true
		}
	}
	return authorization.RoleDefinition{}, false
}

func (s *Service) assignmentScope(ctx context.Context, actor, project, permission string) (storage.Record, error) {
	if err := s.Auth.Check(ctx, actor, project, "", permission); err != nil {
		return storage.Record{}, err
	}
	if project == "" {
		return storage.Record{}, nil
	}
	if !domain.ValidID(project) {
		return storage.Record{}, errors.New("valid project identifier is required")
	}
	row, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return row, err
	}
	p, err := storage.Decode[struct {
		ID string `json:"id"`
	}](row)
	if err != nil {
		return row, err
	}
	if p.ID != project || row.Project != project {
		return storage.Record{}, storage.ErrNotFound
	}
	return row, nil
}

func (s *Service) assignmentAuthority(ctx context.Context, actor, project string, d AssignmentDefinition, manage bool) error {
	if (d.PrincipalKind != permissions.PrincipalUser && d.PrincipalKind != permissions.PrincipalGroup) || !domain.ValidID(d.PrincipalID) {
		return errors.New("a valid user or group principal is required")
	}
	if _, err := s.roleForScope(ctx, project, d.RoleID); err != nil {
		return err
	}
	permission, membership := "permissions.read", "members.read"
	if manage {
		permission, membership = "permissions.grant", "members.manage"
	}
	for _, object := range []string{"role/" + d.RoleID, string(d.PrincipalKind) + "/" + d.PrincipalID} {
		if err := s.Auth.Check(ctx, actor, project, object, permission); err != nil {
			return err
		}
		if err := s.Auth.Check(ctx, actor, project, object, membership); err != nil {
			return err
		}
	}
	if manage && !strings.HasPrefix(d.RoleID, "builtin.") {
		return s.customRoleDelegation(ctx, actor, project, d.RoleID)
	}
	return nil
}

func (s *Service) assignmentPrincipal(ctx context.Context, project string, key assignmentIdentity, add bool) (string, bool, error) {
	if key.PrincipalKind == permissions.PrincipalUser {
		u, err := s.Auth.Directory.User(ctx, key.PrincipalID)
		if errors.Is(err, storage.ErrNotFound) && !add {
			return key.PrincipalID, false, nil
		}
		if err != nil {
			return "", false, err
		}
		if add && u.Disabled {
			return "", false, errors.New("restore the disabled user before adding access")
		}
		return u.Name, !u.Disabled, nil
	}
	row, err := s.Store.Get(ctx, "groups", key.PrincipalID)
	if err != nil {
		return "", false, err
	}
	g, err := storage.Decode[identity.Group](row)
	if err != nil {
		return "", false, err
	}
	if g.Project != row.Project || (g.Project != "" && g.Project != project) {
		return "", false, storage.ErrNotFound
	}
	if add && g.Archived {
		return "", false, errors.New("restore the archived group before adding access")
	}
	return g.Name, !g.Archived, nil
}

type RoleView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
	BuiltIn     bool     `json:"builtIn"`
	Disabled    bool     `json:"disabled"`
	Delegatable bool     `json:"delegatable"`
}

func (s *Service) ListRoles(ctx context.Context, actor, project string) ([]RoleView, error) {
	if _, err := s.assignmentScope(ctx, actor, project, "roles.read"); err != nil {
		return nil, err
	}
	roles := []RoleView{}
	for _, definition := range authorization.BuiltInRoles() {
		if _, ok := builtinRole(project, definition.ID); !ok {
			continue
		}
		if err := s.Auth.Check(ctx, actor, project, "role/"+definition.ID, "roles.read"); err != nil {
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			return nil, err
		}
		role, err := s.Auth.Engine.RoleDefinition(ctx, definition.ID)
		if err != nil {
			return nil, err
		}
		scope := "project"
		if project == "" {
			scope = "installation"
		}
		delegate := !role.IsDisabled
		for _, permission := range []string{"permissions.grant", "members.manage"} {
			if err := s.Auth.Check(ctx, actor, project, "role/"+definition.ID, permission); err != nil {
				if !errors.Is(err, authorization.ErrForbidden) {
					return nil, err
				}
				delegate = false
			}
		}
		roles = append(roles, RoleView{ID: definition.ID, Name: definition.Name, Scope: scope, Permissions: slices.Clone(definition.Permissions), BuiltIn: true, Disabled: role.IsDisabled, Delegatable: delegate})
	}
	return roles, nil
}

func (s *Service) assignmentCapability(ctx context.Context, actor string, row AssignmentRecord) (AssignmentRecord, error) {
	d := AssignmentDefinition{PrincipalKind: row.Data.PrincipalKind, PrincipalID: row.Data.PrincipalID, RoleID: row.Data.RoleID}
	err := s.assignmentAuthority(ctx, actor, row.Project, d, true)
	if err == nil && d.PrincipalKind == permissions.PrincipalGroup {
		_, _, err = s.group(ctx, actor, d.PrincipalID, "groups.manage")
	}
	if errors.Is(err, authorization.ErrForbidden) || errors.Is(err, storage.ErrNotFound) {
		return row, nil
	}
	if err != nil {
		return AssignmentRecord{}, err
	}
	row.Data.CanRevoke = true
	return row, nil
}
