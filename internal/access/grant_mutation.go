package access

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) CreateGrant(ctx context.Context, actor, project string, input GrantDefinition) (GrantRecord, error) {
	if err := reason(input.Reason); err != nil {
		return GrantRecord{}, err
	}
	value, err := normalizeGrant(project, input)
	if err != nil {
		return GrantRecord{}, err
	}
	return s.changeGrant(ctx, actor, project, value.id(), value, input.Reason, 0, "create")
}

func (s *Service) UpdateGrant(ctx context.Context, actor, project, id string, input GrantUpdate, revision int64) (GrantRecord, error) {
	expiry, err := normalizePolicyExpiry(input.ExpiresAt)
	if err != nil {
		return GrantRecord{}, err
	}
	input.ExpiresAt = expiry
	return s.changeGrant(ctx, actor, project, id, grantState{ExpiresAt: input.ExpiresAt}, input.Reason, revision, "update")
}

func (s *Service) RemoveGrant(ctx context.Context, actor, project, id, why string, revision int64) error {
	_, err := s.changeGrant(ctx, actor, project, id, grantState{}, why, revision, "remove")
	return err
}

func (s *Service) changeGrant(ctx context.Context, actor, project, id string, input grantState, why string, revision int64, operation string) (GrantRecord, error) {
	if err := reason(why); err != nil {
		return GrantRecord{}, err
	}
	if operation != "create" && revision < 1 {
		return GrantRecord{}, errors.New("existing grants require a positive revision")
	}
	var result GrantRecord
	err := s.Auth.WithPolicyTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts, SessionIdleTimeout: s.SessionIdleTimeout}
		for _, action := range []string{"permissions.grant", "members.manage"} {
			if _, err := bound.assignmentScope(ctx, actor, project, action); err != nil {
				return err
			}
		}
		prior, row, err := bound.loadGrant(ctx, project, id)
		if err != nil && (!errors.Is(err, storage.ErrNotFound) || operation != "create") {
			return err
		}
		value := input
		if operation != "create" {
			value = prior
			if operation == "update" {
				value.ExpiresAt = input.ExpiresAt
			}
		}
		if err := bound.grantAuthority(ctx, actor, value, true); err != nil {
			return err
		}
		if (operation == "create" && prior.Present) || (operation != "create" && (!prior.Present || row.Revision != revision)) {
			return storage.ErrConflict
		}
		var projectRow storage.Record
		if operation != "remove" {
			projectRow, err = bound.grantCanAdd(ctx, actor, value)
			if err != nil {
				return err
			}
		}
		if prior.Present {
			if err := bound.verifyNativeGrant(ctx, prior); err != nil {
				return err
			}
		}
		if operation == "create" {
			match, err := bound.matchingGrant(ctx, value)
			if err != nil {
				return err
			}
			if match.ID > 0 {
				return storage.ErrConflict
			}
		}
		hadAdmin := false
		ownerScopes, err := bound.ownerCheckpoint(ctx)
		if err != nil {
			return err
		}
		if project == "" {
			hadAdmin, err = bound.hasUsableAdministrator(ctx)
			if err != nil {
				return err
			}
		}
		if operation == "remove" {
			if err := auth.RemovePolicyGrant(ctx, prior.NativeID); err != nil {
				return err
			}
			value.Present = false
		} else if operation == "create" || roleGrantKey(prior.native(prior.ExpiresAt)) != roleGrantKey(value.native(value.ExpiresAt)) {
			if prior.Present {
				if err := auth.RemovePolicyGrant(ctx, prior.NativeID); err != nil {
					return err
				}
			}
			if err := auth.Engine.CreateGrant(ctx, value.native(value.ExpiresAt)); err != nil {
				return err
			}
			native, err := bound.matchingGrant(ctx, value)
			if err != nil {
				return err
			}
			if native.ID < 1 || roleGrantKey(native) != roleGrantKey(value.native(value.ExpiresAt)) {
				return storage.ErrConflict
			}
			value.NativeID = native.ID
		}
		if err := bound.preserveOwners(ctx, ownerScopes); err != nil {
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
		change, err := storage.Put(grantBucket(project), id, project, row.Revision, value)
		if err != nil {
			return err
		}
		var before any
		if prior.Present {
			before = GrantDefinition{PrincipalKind: prior.PrincipalKind, PrincipalID: prior.PrincipalID, Permission: prior.Permission, Effect: prior.Effect, Object: prior.Object, ExpiresAt: prior.ExpiresAt}
		}
		audit, err := domain.Audit(actor, project, "permission.grant."+operation, id, before, map[string]any{"principalKind": value.PrincipalKind, "principalId": value.PrincipalID, "permission": value.Permission, "effect": value.Effect, "object": value.Object, "expiresAt": value.ExpiresAt, "present": value.Present, "reason": why})
		if err != nil {
			return err
		}
		changes := []storage.Change{change, audit}
		if projectRow.Revision > 0 {
			changes = append(changes, storage.Change{Record: projectRow, Assert: true})
		}
		if err := bound.Store.Commit(ctx, changes...); err != nil {
			return err
		}
		if operation == "remove" {
			return nil
		}
		saved, err := bound.Store.Get(ctx, grantBucket(project), id)
		if err != nil {
			return err
		}
		result, err = bound.grantView(ctx, actor, value, saved)
		return err
	})
	if err != nil {
		return GrantRecord{}, err
	}
	return result, nil
}
