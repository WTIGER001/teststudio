package sessions

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Delegate(ctx context.Context, parent, clientName string, checkpoint func(Session) ([]storage.Change, error)) (Session, *http.Cookie, error) {
	if checkpoint == nil {
		return Session{}, nil, errors.New("session delegation requires an atomic authorization checkpoint")
	}
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return Session{}, nil, err
	}
	current, err := s.Validate(ctx, parent)
	if err != nil {
		return Session{}, nil, err
	}
	if current.MustChangePassword {
		return Session{}, nil, ErrPasswordChangeRequired
	}
	original, err := s.Store.Get(ctx, "sessions", parent)
	if err != nil {
		return Session{}, nil, err
	}
	now := time.Now().UTC()
	token := Random()
	next := current
	next.ID = Digest(token)
	next.CSRF = Random()
	next.CreatedAt = now
	next.LastSeen = now
	next.Client = "publisher-cli"
	next.ClientName = clientName
	if next.AuthenticationUntil.Before(next.ExpiresAt) {
		next.ExpiresAt = next.AuthenticationUntil
	}
	change, err := storage.Put("sessions", next.ID, "", 0, next)
	if err != nil {
		return Session{}, nil, err
	}
	changes, err := checkpoint(next)
	if err != nil {
		return Session{}, nil, err
	}
	audit, err := domain.Audit(next.UserID, "", "session.cli.issue", next.ID, nil, map[string]any{"client": "publisher-cli", "provider": next.Provider, "expiresAt": next.ExpiresAt})
	if err != nil {
		return Session{}, nil, err
	}
	changes = append(changes, change, storage.Change{Record: original, Assert: true}, audit)
	if err := s.Directory.CommitAt(ctx, version, changes...); err != nil {
		return Session{}, nil, err
	}
	cookie := &http.Cookie{Name: "teststudio_session", Value: token, Path: "/", HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode, Expires: next.ExpiresAt}
	return next, cookie, nil
}
