package access

import (
	"context"
	"errors"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ApplyPendingAccess(ctx context.Context, proof authoidc.SignInIdentity) error {
	if !proof.EmailVerified {
		return nil
	}
	email, err := pendingEmail(proof.Email)
	if err != nil {
		return nil
	}
	if !time.Now().Before(proof.ExpiresAt) {
		return errors.New("sign-in identity expired before pending access activation")
	}
	bucket := pendingMatchBucket(proof.ProviderID, email)
	matches, err := s.Store.List(ctx, bucket, "", "", 100)
	if err != nil {
		return err
	}
	if matches.Cursor != "" {
		return errors.New("pending project access exceeds its activation bound")
	}
	if len(matches.Items) == 0 {
		return nil
	}
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := *s
		bound.Auth, bound.Store = auth, auth.Directory.Store
		provider, providerRow, err := bound.loadProvider(ctx, proof.ProviderID)
		if err != nil {
			return err
		}
		if provider.Disabled || providerRow.Revision != proof.ProviderRevision || provider.Issuer != proof.Issuer || !time.Now().Before(proof.ExpiresAt) {
			return storage.ErrConflict
		}
		link, err := auth.Directory.ExternalLink(ctx, proof.Issuer, proof.Subject)
		if err != nil {
			return err
		}
		if link.UserID != proof.UserID {
			return authorization.ErrForbidden
		}
		user, err := auth.Directory.User(ctx, proof.UserID)
		if err != nil {
			return err
		}
		if user.Disabled {
			return authorization.ErrForbidden
		}
		for _, snapshot := range matches.Items {
			index, err := bound.Store.Get(ctx, bucket, snapshot.ID)
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			match, err := storage.Decode[pendingAccessMatch](index)
			if err != nil {
				return err
			}
			value, row, err := bound.loadPendingAccess(ctx, match.Project, match.ID)
			if err != nil {
				return err
			}
			if index.Project != value.Project || index.ID != pendingMatchID(value) || value.State != "pending" || value.ProviderID != proof.ProviderID || value.Email != email {
				return errors.New("pending project access index is inconsistent")
			}
			blocked, err := bound.pendingActivationCheck(ctx, value, proof)
			if err != nil {
				return err
			}
			before := value.PendingAccessView
			action := "project-access.block"
			if blocked != "" {
				value.State, value.BlockReason = "blocked", blocked
			} else {
				definition := AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: user.ID, RoleID: value.RoleID, Reason: value.Reason}
				key := assignmentKey(value.Project, definition)
				present, err := bound.assignmentPresent(ctx, key)
				if err != nil {
					return err
				}
				if !present {
					if _, err := bound.applyAssignment(ctx, value.ApprovedBy, value.Project, definition, 0, false); err != nil {
						return err
					}
				}
				now := time.Now().UTC()
				value.State, value.ActivatedAt, value.ActivatedUser, value.AssignmentID = "activated", &now, user.ID, key.id()
				action = "project-access.activate"
			}
			change, err := storage.Put(pendingAccessBucket, value.ID, value.Project, row.Revision, value)
			if err != nil {
				return err
			}
			audit, err := domain.Audit(user.ID, value.Project, action, value.ID, before, value.PendingAccessView)
			if err != nil {
				return err
			}
			if err := bound.Store.Commit(ctx, change, audit, storage.Change{Record: index, Delete: true}); err != nil {
				return err
			}
		}
		return bound.Store.Commit(ctx, storage.Change{Record: providerRow, Assert: true})
	})
}

func (s *Service) pendingActivationCheck(ctx context.Context, value pendingAccessState, proof authoidc.SignInIdentity) (string, error) {
	if value.ProviderRevision != proof.ProviderRevision || value.Issuer != proof.Issuer {
		return "provider-changed", nil
	}
	role, err := s.roleForScope(ctx, value.Project, value.RoleID)
	if errors.Is(err, storage.ErrNotFound) {
		return "role-changed", nil
	}
	if err != nil {
		return "", err
	}
	if role.IsDisabled {
		return "role-changed", nil
	}
	digest, _, err := s.pendingRoleDigest(ctx, value.Project, value.RoleID)
	if errors.Is(err, storage.ErrNotFound) {
		return "role-changed", nil
	}
	if err != nil {
		return "", err
	}
	if digest != value.RoleDigest {
		return "role-changed", nil
	}
	projectRow, err := s.pendingAuthority(ctx, value.ApprovedBy, value.Project, value.ID, value.RoleID, true)
	if errors.Is(err, authorization.ErrForbidden) || errors.Is(err, storage.ErrNotFound) {
		return "authority-changed", nil
	}
	if err != nil {
		return "", err
	}
	project, err := storage.Decode[struct {
		Archived bool `json:"archived"`
	}](projectRow)
	if err != nil {
		return "", err
	}
	if project.Archived {
		return "project-unavailable", nil
	}
	definition := AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: proof.UserID, RoleID: value.RoleID}
	err = s.assignmentAuthority(ctx, value.ApprovedBy, value.Project, definition, true)
	if errors.Is(err, authorization.ErrForbidden) || errors.Is(err, storage.ErrNotFound) {
		return "authority-changed", nil
	}
	return "", err
}
