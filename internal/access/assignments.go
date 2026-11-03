package access

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type AssignmentDefinition struct {
	PrincipalKind permissions.PrincipalKind `json:"principalKind"`
	PrincipalID   string                    `json:"principalId"`
	RoleID        string                    `json:"roleId"`
	Reason        string                    `json:"reason"`
}

type assignmentIdentity struct {
	PrincipalKind permissions.PrincipalKind `json:"principalKind"`
	PrincipalID   string                    `json:"principalId"`
	RoleID        string                    `json:"roleId"`
	Bindings      map[string]any            `json:"bindingValues"`
}

type assignmentState struct {
	assignmentIdentity
	Present bool `json:"present"`
}

type AssignmentView struct {
	assignmentIdentity
	PrincipalName   string `json:"principalName"`
	PrincipalActive bool   `json:"principalActive"`
	RoleName        string `json:"roleName"`
	RoleDisabled    bool   `json:"roleDisabled"`
	Managed         bool   `json:"managed"`
}

type AssignmentRecord struct {
	ID       string         `json:"id"`
	Project  string         `json:"projectId,omitempty"`
	Revision int64          `json:"revision"`
	Updated  *time.Time     `json:"updatedAt,omitempty"`
	Data     AssignmentView `json:"data"`
}

type AssignmentsPage struct {
	Items   []AssignmentRecord `json:"items"`
	Cursor  string             `json:"cursor,omitempty"`
	Partial bool               `json:"partial"`
}

func assignmentKey(project string, d AssignmentDefinition) assignmentIdentity {
	bindings := map[string]any{}
	if project != "" {
		bindings["team"] = project
	}
	return assignmentIdentity{PrincipalKind: d.PrincipalKind, PrincipalID: d.PrincipalID, RoleID: d.RoleID, Bindings: bindings}
}

func (a assignmentIdentity) id() string {
	raw, _ := json.Marshal(a)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (a assignmentIdentity) principal() permissions.PrincipalRef {
	return permissions.PrincipalRef{Kind: a.PrincipalKind, ID: a.PrincipalID}
}

func (s *Service) assignmentPresent(ctx context.Context, key assignmentIdentity) (bool, error) {
	assignments, err := s.Auth.Engine.RoleAssignmentsForPrincipal(ctx, key.principal())
	if err != nil {
		return false, err
	}
	for _, a := range assignments {
		bindings := a.BindingValues
		if bindings == nil {
			bindings = map[string]any{}
		}
		if a.RoleID == key.RoleID && reflect.DeepEqual(bindings, key.Bindings) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) assignmentRecord(ctx context.Context, project string, key assignmentIdentity) (AssignmentRecord, storage.Record, error) {
	metadata, err := s.Store.Get(ctx, "access-assignments", key.id())
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return AssignmentRecord{}, metadata, err
	}
	revision := int64(1)
	managed := false
	if metadata.Revision > 0 {
		state, err := storage.Decode[assignmentState](metadata)
		if err != nil {
			return AssignmentRecord{}, metadata, err
		}
		if !reflect.DeepEqual(state.assignmentIdentity, key) || metadata.Project != project {
			return AssignmentRecord{}, metadata, errors.New("assignment metadata does not match its identity")
		}
		revision = 2 * metadata.Revision
		managed = state.Present
		if !state.Present {
			revision++
		}
	}
	name, active, err := s.assignmentPrincipal(ctx, project, key, false)
	if err != nil {
		return AssignmentRecord{}, metadata, err
	}
	role, err := s.Auth.Engine.RoleDefinition(ctx, key.RoleID)
	if err != nil {
		return AssignmentRecord{}, metadata, err
	}
	view := AssignmentRecord{ID: key.id(), Project: project, Revision: revision, Data: AssignmentView{assignmentIdentity: key, PrincipalName: name, PrincipalActive: active, RoleName: role.Name, RoleDisabled: role.IsDisabled, Managed: managed}}
	if managed {
		view.Updated = &metadata.Updated
	}
	return view, metadata, nil
}

func (s *Service) ReadAssignment(ctx context.Context, actor, project string, d AssignmentDefinition) (AssignmentRecord, error) {
	if _, err := s.assignmentScope(ctx, actor, project, "members.read"); err != nil {
		return AssignmentRecord{}, err
	}
	if err := s.assignmentAuthority(ctx, actor, project, d, false); err != nil {
		return AssignmentRecord{}, err
	}
	key := assignmentKey(project, d)
	present, err := s.assignmentPresent(ctx, key)
	if err != nil {
		return AssignmentRecord{}, err
	}
	if !present {
		return AssignmentRecord{}, storage.ErrNotFound
	}
	view, _, err := s.assignmentRecord(ctx, project, key)
	return view, err
}

func (s *Service) CreateAssignment(ctx context.Context, actor, project string, d AssignmentDefinition) (AssignmentRecord, error) {
	return s.changeAssignment(ctx, actor, project, d, 0, false)
}

func (s *Service) RemoveAssignment(ctx context.Context, actor, project string, d AssignmentDefinition, revision int64) error {
	_, err := s.changeAssignment(ctx, actor, project, d, revision, true)
	return err
}

func (s *Service) changeAssignment(ctx context.Context, actor, project string, d AssignmentDefinition, revision int64, remove bool) (AssignmentRecord, error) {
	if err := reason(d.Reason); err != nil {
		return AssignmentRecord{}, err
	}
	var result AssignmentRecord
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, SessionIdleTimeout: s.SessionIdleTimeout}
		projectRow, err := bound.assignmentScope(ctx, actor, project, "members.manage")
		if err != nil {
			return err
		}
		if err := bound.assignmentAuthority(ctx, actor, project, d, true); err != nil {
			return err
		}
		if !remove && projectRow.Revision > 0 {
			p, err := storage.Decode[struct {
				Archived bool `json:"archived"`
			}](projectRow)
			if err != nil {
				return err
			}
			if p.Archived {
				return errors.New("restore the archived project before adding access")
			}
		}
		key := assignmentKey(project, d)
		if key.PrincipalKind == permissions.PrincipalGroup {
			if _, _, err := bound.group(ctx, actor, key.PrincipalID, "groups.manage"); err != nil {
				return err
			}
		}
		if !remove {
			role, err := auth.Engine.RoleDefinition(ctx, key.RoleID)
			if err != nil {
				return err
			}
			if role.IsDisabled {
				return errors.New("enable the disabled role before adding access")
			}
		}
		if _, _, err := bound.assignmentPrincipal(ctx, project, key, !remove); err != nil {
			return err
		}
		present, err := bound.assignmentPresent(ctx, key)
		if err != nil {
			return err
		}
		if (!remove && present) || (remove && !present) {
			return storage.ErrConflict
		}
		before, metadata, err := bound.assignmentRecord(ctx, project, key)
		if err != nil {
			return err
		}
		if remove && (revision < 1 || before.Revision != revision) {
			return storage.ErrConflict
		}
		hadAdmin := false
		if remove && project == "" {
			hadAdmin, err = bound.hasUsableAdministrator(ctx)
			if err != nil {
				return err
			}
		}
		if remove {
			err = auth.Engine.UnassignRole(ctx, key.principal(), key.RoleID, key.Bindings)
		} else {
			err = auth.Engine.AssignRole(ctx, key.principal(), key.RoleID, key.Bindings)
		}
		if err != nil {
			return err
		}
		if hadAdmin {
			remains, err := bound.hasUsableAdministrator(ctx)
			if err != nil {
				return err
			}
			if !remains {
				return errors.New("the change would remove the last usable installation administrator")
			}
		}
		c, err := storage.Put("access-assignments", key.id(), project, metadata.Revision, assignmentState{assignmentIdentity: key, Present: !remove})
		if err != nil {
			return err
		}
		action := "role.assign"
		var prior, after any = nil, key
		if remove {
			action, prior, after = "role.revoke", before.Data, nil
		}
		audit, err := domain.Audit(actor, project, action, key.id(), prior, map[string]any{"value": after, "reason": d.Reason})
		if err != nil {
			return err
		}
		changes := []storage.Change{c, audit}
		if projectRow.Revision > 0 {
			changes = append(changes, storage.Change{Record: projectRow, Assert: true})
		}
		if err := bound.Store.Commit(ctx, changes...); err != nil {
			return err
		}
		if !remove {
			result, _, err = bound.assignmentRecord(ctx, project, key)
		}
		return err
	})
	if err != nil {
		return AssignmentRecord{}, err
	}
	return result, nil
}
