package localauth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Change(ctx context.Context, current sessions.Session, oldPassword, newPassword, peer string) (sessions.Session, *http.Cookie, error) {
	verified, err := s.verify(ctx, current.UserID, oldPassword, peer)
	if err != nil {
		return sessions.Session{}, nil, err
	}
	if passwords.Normalize(oldPassword) == passwords.Normalize(newPassword) {
		return sessions.Session{}, nil, errors.New("choose a different new password")
	}
	hash, err := passwords.Hash(ctx, newPassword)
	if err != nil {
		return sessions.Session{}, nil, err
	}
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return sessions.Session{}, nil, err
	}
	var result sessions.Session
	var cookie *http.Cookie
	err = s.Store.Transaction(ctx, func(bound *storage.Store, _ *sql.Tx) error {
		directory := &identity.Directory{Store: bound}
		if err := directory.CommitAt(ctx, version); err != nil {
			return err
		}
		sessionService := *s.Sessions
		sessionService.Store, sessionService.Directory = bound, directory
		active, err := sessionService.Validate(ctx, current.ID)
		if err != nil {
			return err
		}
		if active.UserID != current.UserID || active.UserEpoch != current.UserEpoch {
			return sessions.ErrUnauthenticated
		}
		record, err := bound.Get(ctx, "local-credentials", current.UserID)
		if err != nil {
			return err
		}
		if record.Revision != verified.Revision {
			return storage.ErrConflict
		}
		credential, err := storage.Decode[identity.LocalCredential](record)
		if err != nil {
			return err
		}
		profile, err := bound.Get(ctx, "users", current.UserID)
		if err != nil {
			return err
		}
		user, err := storage.Decode[identity.User](profile)
		if err != nil {
			return err
		}
		user.SessionEpoch++
		credential.Hash, credential.MustChange = hash, false
		c, err := storage.Put("local-credentials", user.ID, "", record.Revision, credential)
		if err != nil {
			return err
		}
		p, err := storage.Put("users", user.ID, "", profile.Revision, user)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user.ID, "", "identity.local-password-change", user.ID, nil, map[string]bool{"mustChangePassword": false, "sessionsRevoked": true})
		if err != nil {
			return err
		}
		if err := bound.Commit(ctx, c, p, audit); err != nil {
			return err
		}
		result, cookie, err = sessionService.CreatePassword(ctx, user.ID, record.Revision+1, current.ID)
		return err
	})
	if err != nil {
		return sessions.Session{}, nil, err
	}
	return result, cookie, nil
}
