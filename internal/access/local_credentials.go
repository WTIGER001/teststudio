package access

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/storage"
)

type LocalCredentialView struct {
	UserID               string     `json:"userId"`
	Configured           bool       `json:"configured"`
	Disabled             bool       `json:"disabled"`
	MustChange           bool       `json:"mustChangePassword"`
	LocalAccountsEnabled bool       `json:"localAccountsEnabled"`
	CanManage            bool       `json:"canManage"`
	CreatedAt            *time.Time `json:"createdAt,omitempty"`
	UpdatedAt            *time.Time `json:"updatedAt,omitempty"`
}

type LocalCredentialRecord struct {
	ID       string              `json:"id"`
	Revision int64               `json:"revision"`
	Data     LocalCredentialView `json:"data"`
}

func (s *Service) credentialView(ctx context.Context, actor, user string) (LocalCredentialRecord, error) {
	if _, err := s.Auth.Directory.User(ctx, user); err != nil {
		return LocalCredentialRecord{}, err
	}
	row, err := s.Store.Get(ctx, "local-credentials", user)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return LocalCredentialRecord{}, err
	}
	enabled, err := s.localModeEnabled(ctx)
	if err != nil {
		return LocalCredentialRecord{}, err
	}
	result := LocalCredentialRecord{ID: user, Revision: row.Revision, Data: LocalCredentialView{UserID: user, Configured: row.Revision > 0, LocalAccountsEnabled: enabled}}
	if row.Revision > 0 {
		credential, err := storage.Decode[identity.LocalCredential](row)
		if err != nil {
			return LocalCredentialRecord{}, err
		}
		if credential.UserID != user {
			return LocalCredentialRecord{}, storage.ErrNotFound
		}
		result.Data.Disabled = credential.Disabled || !passwords.ValidHash(credential.Hash)
		result.Data.MustChange = credential.MustChange
		result.Data.CreatedAt, result.Data.UpdatedAt = &credential.CreatedAt, &row.Updated
	}
	if err := s.Auth.Check(ctx, actor, "", "user/"+user, "users.manage"); err == nil {
		result.Data.CanManage = enabled
	} else if !errors.Is(err, authorization.ErrForbidden) {
		return LocalCredentialRecord{}, err
	}
	return result, nil
}

func (s *Service) OwnLocalCredential(ctx context.Context, user string) (LocalCredentialRecord, error) {
	return s.credentialView(ctx, user, user)
}

func (s *Service) ReadLocalCredential(ctx context.Context, actor, user string) (LocalCredentialRecord, error) {
	if err := s.Auth.Check(ctx, actor, "", "user/"+user, "users.read"); err != nil {
		return LocalCredentialRecord{}, err
	}
	return s.credentialView(ctx, actor, user)
}

func (s *Service) SetLocalPassword(ctx context.Context, actor, user, password, why string, revision int64) (LocalCredentialRecord, error) {
	if enabled, err := s.localModeEnabled(ctx); err != nil {
		return LocalCredentialRecord{}, err
	} else if !enabled {
		return LocalCredentialRecord{}, authorization.ErrForbidden
	}
	if !domain.ValidID(user) {
		return LocalCredentialRecord{}, errors.New("valid user identifier is required")
	}
	if err := s.Auth.Check(ctx, actor, "", "user/"+user, "users.manage"); err != nil {
		return LocalCredentialRecord{}, err
	}
	if err := reason(why); err != nil {
		return LocalCredentialRecord{}, err
	}
	hash, err := passwords.Hash(ctx, password)
	if err != nil {
		return LocalCredentialRecord{}, err
	}
	var result LocalCredentialRecord
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts}
		if err := auth.Check(ctx, actor, "", "user/"+user, "users.manage"); err != nil {
			return err
		}
		before, err := bound.credentialView(ctx, actor, user)
		if err != nil {
			return err
		}
		if !before.Data.LocalAccountsEnabled {
			return authorization.ErrForbidden
		}
		if before.Revision != revision {
			return storage.ErrConflict
		}
		credential := identity.LocalCredential{UserID: user, CreatedAt: time.Now().UTC()}
		if revision > 0 {
			row, err := bound.Store.Get(ctx, "local-credentials", user)
			if err != nil {
				return err
			}
			credential, err = storage.Decode[identity.LocalCredential](row)
			if err != nil {
				return err
			}
		}
		credential.Hash, credential.MustChange = hash, true
		if err := bound.saveCredential(ctx, actor, user, why, "identity.local-password-set", before, credential, revision); err != nil {
			return err
		}
		result, err = bound.credentialView(ctx, actor, user)
		return err
	})
	return result, err
}

func (s *Service) LocalCredentialState(ctx context.Context, actor, user, why string, revision int64, disabled bool) (LocalCredentialRecord, error) {
	if enabled, err := s.localModeEnabled(ctx); err != nil {
		return LocalCredentialRecord{}, err
	} else if !enabled {
		return LocalCredentialRecord{}, authorization.ErrForbidden
	}
	if err := reason(why); err != nil {
		return LocalCredentialRecord{}, err
	}
	var result LocalCredentialRecord
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts}
		if err := auth.Check(ctx, actor, "", "user/"+user, "users.manage"); err != nil {
			return err
		}
		before, err := bound.credentialView(ctx, actor, user)
		if err != nil {
			return err
		}
		if !before.Data.LocalAccountsEnabled {
			return authorization.ErrForbidden
		}
		if revision < 1 || revision != before.Revision {
			return storage.ErrConflict
		}
		row, err := bound.Store.Get(ctx, "local-credentials", user)
		if err != nil {
			return err
		}
		credential, err := storage.Decode[identity.LocalCredential](row)
		if err != nil {
			return err
		}
		if !disabled && !passwords.ValidHash(credential.Hash) {
			return errors.New("reset the local password before enabling this credential")
		}
		credential.Disabled = disabled
		action := "identity.local-credential-enable"
		if disabled {
			action = "identity.local-credential-disable"
		}
		if err := bound.saveCredential(ctx, actor, user, why, action, before, credential, revision); err != nil {
			return err
		}
		result, err = bound.credentialView(ctx, actor, user)
		return err
	})
	return result, err
}

func (s *Service) saveCredential(ctx context.Context, actor, user, why, action string, before LocalCredentialRecord, credential identity.LocalCredential, revision int64) error {
	row, err := s.Store.Get(ctx, "users", user)
	if err != nil {
		return err
	}
	profile, err := storage.Decode[identity.User](row)
	if err != nil {
		return err
	}
	profile.SessionEpoch++
	p, err := storage.Put("users", user, "", row.Revision, profile)
	if err != nil {
		return err
	}
	c, err := storage.Put("local-credentials", user, "", revision, credential)
	if err != nil {
		return err
	}
	if credential.Disabled {
		if err := s.preserveAdministrator(ctx, c); err != nil {
			return err
		}
	}
	after := LocalCredentialView{UserID: user, Configured: true, Disabled: credential.Disabled, MustChange: credential.MustChange, LocalAccountsEnabled: before.Data.LocalAccountsEnabled}
	audit, err := domain.Audit(actor, "", action, user, before.Data, map[string]any{"value": after, "reason": why, "sessionsRevoked": true})
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, p, c, audit)
}

func (s *Service) localModeEnabled(ctx context.Context) (bool, error) {
	if !s.LocalAccounts {
		return false, nil
	}
	return s.Auth.Directory.LocalAccountsEnabled(ctx)
}
