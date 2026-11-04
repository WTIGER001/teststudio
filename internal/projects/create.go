package projects

import (
	"context"
	"errors"
	"fmt"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
)

var creatorRoles = []string{
	"builtin.teststudio.project-admin",
	"builtin.teststudio.editor",
	"builtin.teststudio.publisher",
	"builtin.teststudio.operator",
}

func (s *Service) createProject(ctx context.Context, actor string, project Project) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := auth.Check(ctx, actor, "", "", "projects.create"); err != nil {
			return err
		}
		if err := capture.Validate(capture.Resolve(project.Capture, nil), capture.ResolveLimits(project.CaptureLimits)); err != nil {
			return err
		}
		if err := timing.Validate(timing.Resolve(project.Timing, nil)); err != nil {
			return err
		}
		if _, err := bound.Store.Get(ctx, "projects", project.ID); err == nil {
			return storage.ErrConflict
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if err := bound.save(ctx, actor, "projects", project.ID, project.ID, 0, project); err != nil {
			return err
		}
		administrator, err := installationAdministrator(ctx, auth, actor)
		if err != nil || administrator {
			return err
		}
		engine, err := auth.CurrentEngine(ctx)
		if err != nil {
			return err
		}
		for _, id := range creatorRoles {
			role, err := engine.RoleDefinition(ctx, id)
			if err != nil {
				return err
			}
			if role.IsDisabled {
				return fmt.Errorf("project setup requires the enabled %s role; no project was created", role.Name)
			}
			key := authorization.AssignmentIdentity{PrincipalKind: permissions.PrincipalUser, PrincipalID: actor, RoleID: id, Bindings: map[string]any{"team": project.ID}}
			if err := auth.Engine.AssignRole(ctx, key.Principal(), id, key.Bindings); err != nil {
				return err
			}
			metadata, err := storage.Put("access-assignments", key.ID(), project.ID, 0, authorization.AssignmentState{AssignmentIdentity: key, Present: true})
			if err != nil {
				return err
			}
			audit, err := domain.Audit(actor, project.ID, "role.assign", key.ID(), nil, map[string]any{"value": key, "reason": "Initialize creator access for the new project"})
			if err != nil {
				return err
			}
			if err := bound.Store.Commit(ctx, metadata, audit); err != nil {
				return err
			}
		}
		return nil
	})
}

func installationAdministrator(ctx context.Context, auth *authorization.Service, actor string) (bool, error) {
	for _, permission := range authorization.PermissionNames() {
		if err := auth.Check(ctx, actor, "", "", permission); errors.Is(err, authorization.ErrForbidden) {
			return false, nil
		} else if err != nil {
			return false, err
		}
	}
	return true, nil
}
