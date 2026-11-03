package access

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) PendingProviderChoices(ctx context.Context, actor, project string) ([]PendingProviderChoice, error) {
	if project == "" {
		return nil, errors.New("pending access requires an explicit project")
	}
	if _, err := s.assignmentScope(ctx, actor, project, "members.manage"); err != nil {
		return nil, err
	}
	if err := s.Auth.Check(ctx, actor, project, "", "permissions.grant"); err != nil {
		return nil, err
	}
	rows, err := s.Store.All(ctx, "providers", "")
	if err != nil {
		return nil, err
	}
	result := []PendingProviderChoice{}
	for _, row := range rows {
		provider, _, err := s.loadProvider(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if provider.Disabled {
			continue
		}
		result = append(result, PendingProviderChoice{ID: provider.ID, Name: provider.Name, Issuer: provider.Issuer, Revision: row.Revision, AutoProvision: provider.AutoProvision})
	}
	return result, nil
}

func (s *Service) CreatePendingAccess(ctx context.Context, actor, project string, input PendingAccessInput) (PendingAccessRecord, error) {
	if err := reason(input.Reason); err != nil {
		return PendingAccessRecord{}, err
	}
	email, err := pendingEmail(input.Email)
	if err != nil {
		return PendingAccessRecord{}, err
	}
	if input.ProviderRevision < 1 || !domain.ValidID(input.ProviderID) {
		return PendingAccessRecord{}, errors.New("choose an enabled provider and its current revision")
	}
	var result PendingAccessRecord
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := *s
		bound.Auth, bound.Store = auth, auth.Directory.Store
		id := domain.NewID()
		projectRow, err := bound.pendingAuthority(ctx, actor, project, id, input.RoleID, true)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(input.RoleID, "builtin.") {
			if err := bound.customRoleDelegation(ctx, actor, project, input.RoleID); err != nil {
				return err
			}
		}
		projectState, err := storage.Decode[struct {
			Archived bool `json:"archived"`
		}](projectRow)
		if err != nil {
			return err
		}
		if projectState.Archived {
			return errors.New("restore the archived project before adding pending access")
		}
		provider, providerRow, err := bound.loadProvider(ctx, input.ProviderID)
		if err != nil {
			return err
		}
		if provider.Disabled {
			return errors.New("choose an enabled identity provider")
		}
		if providerRow.Revision != input.ProviderRevision {
			return storage.ErrConflict
		}
		digest, roleName, err := bound.pendingRoleDigest(ctx, project, input.RoleID)
		if err != nil {
			return err
		}
		value := pendingAccessState{PendingAccessView: PendingAccessView{ID: id, Project: project, ProviderID: provider.ID, ProviderName: provider.Name, ProviderRevision: providerRow.Revision, Issuer: provider.Issuer, Email: email, RequireVerifiedEmail: true, RoleID: input.RoleID, RoleName: roleName, ApprovedBy: actor, Reason: input.Reason, State: "pending", CreatedAt: time.Now().UTC()}, RoleDigest: digest}
		bucket := pendingMatchBucket(provider.ID, email)
		matches, err := bound.Store.List(ctx, bucket, "", "", 100)
		if err != nil {
			return err
		}
		if len(matches.Items) >= 100 || matches.Cursor != "" {
			return errors.New("this provider and email already have 100 pending project roles")
		}
		index, err := storage.Put(bucket, pendingMatchID(value), project, 0, pendingAccessMatch{ID: id, Project: project})
		if err != nil {
			return err
		}
		change, err := storage.Put(pendingAccessBucket, id, project, 0, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(actor, project, "project-access.prepare", id, nil, value.PendingAccessView)
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, index, change, audit, storage.Change{Record: providerRow, Assert: true}, storage.Change{Record: projectRow, Assert: true}); err != nil {
			return err
		}
		change.Record.Revision = 1
		result, err = bound.pendingRecord(ctx, actor, value, change.Record)
		return err
	})
	return result, err
}

func (s *Service) ReadPendingAccess(ctx context.Context, actor, project, id string) (PendingAccessRecord, error) {
	if _, err := s.assignmentScope(ctx, actor, project, "members.read"); err != nil {
		return PendingAccessRecord{}, err
	}
	value, row, err := s.loadPendingAccess(ctx, project, id)
	if err != nil {
		return PendingAccessRecord{}, err
	}
	if err := s.pendingReadAuthority(ctx, actor, value); err != nil {
		return PendingAccessRecord{}, err
	}
	return s.pendingRecord(ctx, actor, value, row)
}

func (s *Service) ListPendingAccess(ctx context.Context, actor, project, cursor string, limit int) (PendingAccessPage, error) {
	if project == "" {
		return PendingAccessPage{}, errors.New("pending access requires an explicit project")
	}
	if _, err := s.assignmentScope(ctx, actor, project, "members.read"); err != nil {
		return PendingAccessPage{}, err
	}
	if err := s.Auth.Check(ctx, actor, project, "", "permissions.read"); err != nil {
		return PendingAccessPage{}, err
	}
	page, err := s.page(ctx, pendingAccessBucket, project, queryScope(actor, "pending-project-access", project, false), cursor, limit)
	if err != nil {
		return PendingAccessPage{}, err
	}
	result := PendingAccessPage{Items: []PendingAccessRecord{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		value, stored, err := s.loadPendingAccess(ctx, project, row.ID)
		if err != nil {
			return result, err
		}
		if err := s.pendingReadAuthority(ctx, actor, value); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		record, err := s.pendingRecord(ctx, actor, value, stored)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, record)
	}
	return result, nil
}

func (s *Service) CancelPendingAccess(ctx context.Context, actor, project, id, why string, revision int64) (PendingAccessRecord, error) {
	if err := reason(why); err != nil {
		return PendingAccessRecord{}, err
	}
	var result PendingAccessRecord
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := *s
		bound.Auth, bound.Store = auth, auth.Directory.Store
		if _, err := bound.assignmentScope(ctx, actor, project, "members.manage"); err != nil {
			return err
		}
		value, row, err := bound.loadPendingAccess(ctx, project, id)
		if err != nil {
			return err
		}
		if _, err := bound.pendingAuthority(ctx, actor, project, id, value.RoleID, true); err != nil {
			return err
		}
		if revision < 1 || row.Revision != revision {
			return storage.ErrConflict
		}
		if value.State != "pending" && value.State != "blocked" {
			return errors.New("only pending or blocked project access can be cancelled")
		}
		before := value.PendingAccessView
		value.State = "cancelled"
		change, err := storage.Put(pendingAccessBucket, id, project, row.Revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(actor, project, "project-access.cancel", id, before, map[string]any{"value": value.PendingAccessView, "reason": why})
		if err != nil {
			return err
		}
		changes := []storage.Change{change, audit}
		if before.State == "pending" {
			index, err := bound.Store.Get(ctx, pendingMatchBucket(value.ProviderID, value.Email), pendingMatchID(value))
			if err != nil {
				return err
			}
			match, err := storage.Decode[pendingAccessMatch](index)
			if err != nil {
				return err
			}
			if match.ID != id || index.Project != project {
				return storage.ErrConflict
			}
			changes = append(changes, storage.Change{Record: index, Delete: true})
		}
		if err := bound.Store.Commit(ctx, changes...); err != nil {
			return err
		}
		change.Record.Revision++
		result, err = bound.pendingRecord(ctx, actor, value, change.Record)
		return err
	})
	return result, err
}
