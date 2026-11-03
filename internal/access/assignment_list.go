package access

import (
	"context"
	"errors"
	"reflect"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ListAssignments(ctx context.Context, actor, project, cursor string, limit int) (AssignmentsPage, error) {
	if _, err := s.assignmentScope(ctx, actor, project, "members.read"); err != nil {
		return AssignmentsPage{}, err
	}
	if err := s.Auth.Check(ctx, actor, project, "", "permissions.read"); err != nil {
		return AssignmentsPage{}, err
	}
	reader, ok := s.Auth.Engine.GetStore().(authorization.AssignmentReader)
	if !ok {
		return AssignmentsPage{}, errors.New("permission store does not support bounded assignment reads")
	}
	scope := queryScope(actor, "role-assignments", project, false)
	raw, err := s.openCursor(ctx, scope, cursor)
	if err != nil {
		return AssignmentsPage{}, err
	}
	page, err := reader.AssignmentPage(ctx, raw, limit)
	if err != nil {
		return AssignmentsPage{}, err
	}
	result := AssignmentsPage{Items: []AssignmentRecord{}, Partial: page.Cursor != ""}
	result.Cursor, err = s.sealCursor(ctx, scope, page.Cursor)
	if err != nil {
		return result, err
	}
	for _, hit := range page.Items {
		if hit.Kind != permissions.PrincipalUser && hit.Kind != permissions.PrincipalGroup {
			continue
		}
		if strings.HasPrefix(hit.RoleID, "builtin.") {
			if _, ok := builtinRole(project, hit.RoleID); !ok {
				continue
			}
		} else {
			if _, err := s.roleForScope(ctx, project, hit.RoleID); errors.Is(err, storage.ErrNotFound) {
				continue
			} else if err != nil {
				return result, err
			}
		}
		d := AssignmentDefinition{PrincipalKind: hit.Kind, PrincipalID: hit.ID, RoleID: hit.RoleID}
		key := assignmentKey(project, d)
		bindings := hit.BindingValues
		if bindings == nil {
			bindings = map[string]any{}
		}
		if !reflect.DeepEqual(bindings, key.Bindings) {
			continue
		}
		if err := s.assignmentAuthority(ctx, actor, project, d, false); err != nil {
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			return result, err
		}
		row, _, err := s.assignmentRecord(ctx, project, key)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return result, err
		}
		row, err = s.assignmentCapability(ctx, actor, row)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}
