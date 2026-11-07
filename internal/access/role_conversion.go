package access

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RoleConversion struct {
	ID                string                    `json:"id"`
	Project           string                    `json:"projectId"`
	Application       string                    `json:"applicationId,omitempty"`
	PrincipalKind     permissions.PrincipalKind `json:"principalKind"`
	PrincipalID       string                    `json:"principalId"`
	TargetRoles       []string                  `json:"targetRoleIds"`
	Before            []AssignmentRecord        `json:"before"`
	Added             []string                  `json:"addedPermissions"`
	Removed           []string                  `json:"removedPermissions"`
	Expires           time.Time                 `json:"expiresAt"`
	Applied           bool                      `json:"applied"`
	Result            []AssignmentRecord        `json:"result,omitempty"`
	Reason            string                    `json:"reason"`
	Actor             string                    `json:"actor"`
	DirectoryRevision int64                     `json:"directoryRevision"`
	Reverses          string                    `json:"reverses,omitempty"`
}

func (s *Service) conversionAssignments(ctx context.Context, actor, project string, d AssignmentDefinition) ([]AssignmentRecord, error) {
	rows, err := s.Auth.Engine.RoleAssignmentsForPrincipal(ctx, permissions.PrincipalRef{Kind: d.PrincipalKind, ID: d.PrincipalID})
	if err != nil {
		return nil, err
	}
	result := []AssignmentRecord{}
	for _, row := range rows {
		if !strings.HasPrefix(row.RoleID, "builtin.") || !reflect.DeepEqual(row.BindingValues, assignmentKey(project, d).Bindings) {
			continue
		}
		if _, ok := builtinRole(project, row.RoleID); !ok {
			continue
		}
		definition := d
		definition.RoleID = row.RoleID
		if err := s.assignmentAuthority(ctx, actor, project, definition, true); err != nil {
			return nil, err
		}
		view, _, err := s.assignmentRecord(ctx, project, assignmentKey(project, definition))
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	slices.SortFunc(result, func(a, b AssignmentRecord) int { return strings.Compare(a.Data.RoleID, b.Data.RoleID) })
	return result, nil
}

func (s *Service) conversionPermissions(ctx context.Context, roles []string) ([]string, error) {
	result := []string{}
	engine, err := s.Auth.CurrentEngine(ctx)
	if err != nil {
		return nil, err
	}
	for _, id := range roles {
		role, err := engine.RoleDefinition(ctx, id)
		if err != nil {
			return nil, err
		}
		if role.IsDisabled {
			continue
		}
		definition, err := predefinedRole(id)
		if err != nil {
			return nil, err
		}
		for _, action := range definition.Permissions {
			if !slices.Contains(result, action) {
				result = append(result, action)
			}
		}
	}
	slices.Sort(result)
	return result, nil
}

func (s *Service) PreviewRoleConversion(ctx context.Context, actor, project string, d AssignmentDefinition) (storage.Record, error) {
	if !authorization.StandardRole(d.RoleID) {
		return storage.Record{}, errors.New("conversion requires a standard role")
	}
	return s.previewRoleConversion(ctx, actor, project, d, []string{d.RoleID}, "")
}

func (s *Service) previewRoleConversion(ctx context.Context, actor, project string, d AssignmentDefinition, targets []string, reverses string) (storage.Record, error) {
	if err := reason(d.Reason); err != nil {
		return storage.Record{}, err
	}
	d.Application = authorization.ScopedApplication(ctx, project)
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return storage.Record{}, err
	}
	if _, err := s.assignmentScope(ctx, actor, project, "members.manage"); err != nil {
		return storage.Record{}, err
	}
	for _, id := range targets {
		definition := d
		definition.RoleID = id
		if err := s.assignmentAuthority(ctx, actor, project, definition, true); err != nil {
			return storage.Record{}, err
		}
		role, err := s.roleForScope(ctx, project, id)
		if err != nil {
			return storage.Record{}, err
		}
		if role.IsDisabled {
			return storage.Record{}, errors.New("conversion target role is disabled")
		}
	}
	if d.PrincipalKind == permissions.PrincipalGroup {
		if _, _, err := s.group(ctx, actor, d.PrincipalID, "groups.manage"); err != nil {
			return storage.Record{}, err
		}
	}
	if _, _, err := s.assignmentPrincipal(ctx, project, assignmentKey(project, d), true); err != nil {
		return storage.Record{}, err
	}
	before, err := s.conversionAssignments(ctx, actor, project, d)
	if err != nil {
		return storage.Record{}, err
	}
	if len(before) == 0 {
		return storage.Record{}, errors.New("there are no predefined assignments to convert")
	}
	prior := []string{}
	for _, row := range before {
		prior = append(prior, row.Data.RoleID)
	}
	oldPermissions, err := s.conversionPermissions(ctx, prior)
	if err != nil {
		return storage.Record{}, err
	}
	newPermissions, err := s.conversionPermissions(ctx, targets)
	if err != nil {
		return storage.Record{}, err
	}
	value := RoleConversion{ID: domain.NewID(), Project: project, Application: d.Application, PrincipalKind: d.PrincipalKind, PrincipalID: d.PrincipalID, TargetRoles: targets, Before: before, Added: []string{}, Removed: []string{}, Expires: time.Now().UTC().Add(10 * time.Minute), Reason: d.Reason, Actor: actor, DirectoryRevision: version.Revision, Reverses: reverses}
	for _, action := range newPermissions {
		if !slices.Contains(oldPermissions, action) {
			value.Added = append(value.Added, action)
		}
	}
	for _, action := range oldPermissions {
		if !slices.Contains(newPermissions, action) {
			value.Removed = append(value.Removed, action)
		}
	}
	change, err := storage.Put("access-role-conversions", value.ID, project, 0, value)
	if err != nil {
		return storage.Record{}, err
	}
	if err := s.Store.Commit(ctx, change, storage.Change{Record: version, Assert: true}); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, "access-role-conversions", value.ID)
}

func (s *Service) ReadRoleConversion(ctx context.Context, actor, project, id string) (storage.Record, error) {
	row, err := s.Store.Get(ctx, "access-role-conversions", id)
	if err != nil {
		return row, err
	}
	value, err := storage.Decode[RoleConversion](row)
	if err != nil {
		return row, err
	}
	if row.Project != project || value.Project != project || value.Application != authorization.ScopedApplication(ctx, project) {
		return storage.Record{}, storage.ErrNotFound
	}
	if value.Actor != actor {
		return storage.Record{}, authorization.ErrForbidden
	}
	if _, err := s.assignmentScope(ctx, actor, project, "members.read"); err != nil {
		return storage.Record{}, err
	}
	return row, nil
}

func (s *Service) ApplyRoleConversion(ctx context.Context, actor, project, id string, revision int64) (storage.Record, error) {
	var result storage.Record
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := *s
		bound.Auth, bound.Store = auth, auth.Directory.Store
		row, err := bound.ReadRoleConversion(ctx, actor, project, id)
		if err != nil {
			return err
		}
		value, err := storage.Decode[RoleConversion](row)
		if err != nil {
			return err
		}
		if value.Applied {
			result = row
			return nil
		}
		if row.Revision != revision || !time.Now().Before(value.Expires) {
			return storage.ErrConflict
		}
		version, err := bound.Auth.Directory.Revision(ctx)
		if err != nil {
			return err
		}
		if version.Revision != value.DirectoryRevision+1 {
			return storage.ErrConflict
		}
		d := AssignmentDefinition{Application: value.Application, PrincipalKind: value.PrincipalKind, PrincipalID: value.PrincipalID, Reason: value.Reason}
		current, err := bound.conversionAssignments(ctx, actor, project, d)
		if err != nil {
			return err
		}
		if len(current) != len(value.Before) {
			return storage.ErrConflict
		}
		for i, row := range current {
			if row.ID != value.Before[i].ID || row.Revision != value.Before[i].Revision {
				return storage.ErrConflict
			}
		}
		scopes, err := bound.ownerCheckpoint(ctx)
		if err != nil {
			return err
		}
		definitions := []AssignmentDefinition{}
		for _, before := range value.Before {
			remove := d
			remove.RoleID = before.Data.RoleID
			definitions = append(definitions, remove)
		}
		for _, role := range value.TargetRoles {
			add := d
			add.RoleID = role
			definitions = append(definitions, add)
		}
		ctx, err = bound.authorizeAssignmentBatch(ctx, actor, project, definitions)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, replacingStandardRoleKey{}, true)
		for _, before := range value.Before {
			remove := d
			remove.RoleID = before.Data.RoleID
			if _, err := bound.applyAssignment(ctx, actor, project, remove, before.Revision, true); err != nil {
				return err
			}
		}
		for _, id := range value.TargetRoles {
			add := d
			add.RoleID = id
			view, err := bound.applyAssignment(ctx, actor, project, add, 0, false)
			if err != nil {
				return err
			}
			value.Result = append(value.Result, view)
		}
		if err := bound.preserveOwners(ctx, scopes); err != nil {
			return err
		}
		value.Applied = true
		change, err := storage.Put(row.Bucket, row.ID, project, row.Revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(actor, project, "role.convert", id, value.Before, map[string]any{"roles": value.TargetRoles, "reason": value.Reason, "reverses": value.Reverses})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, audit); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, row.Bucket, row.ID)
		return err
	})
	return result, err
}

func (s *Service) PreviewRoleConversionRollback(ctx context.Context, actor, project, id, why string) (storage.Record, error) {
	row, err := s.ReadRoleConversion(ctx, actor, project, id)
	if err != nil {
		return row, err
	}
	value, err := storage.Decode[RoleConversion](row)
	if err != nil {
		return row, err
	}
	if !value.Applied {
		return storage.Record{}, storage.ErrConflict
	}
	targets := []string{}
	for _, prior := range value.Before {
		targets = append(targets, prior.Data.RoleID)
	}
	return s.previewRoleConversion(ctx, actor, project, AssignmentDefinition{PrincipalKind: value.PrincipalKind, PrincipalID: value.PrincipalID, Reason: why}, targets, id)
}
